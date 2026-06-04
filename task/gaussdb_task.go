package task

import (
	"github.com/pkg/errors"
	"github.com/qaqcatz/impomysql/connector"
	"github.com/qaqcatz/impomysql/mutation/oracle"
	"github.com/qaqcatz/impomysql/mutation/stage1"
	"github.com/qaqcatz/impomysql/mutation/stage2"
	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"time"
)

// RunTaskGaussDB: Run task for GaussDB/openGauss M mode
// Uses PostgreSQL protocol with pgx driver
func RunTaskGaussDB(config *TaskConfig, publicLogger *logrus.Logger) (*TaskResult, error) {
	// 1 init
	startTime := time.Now().String()
	// **************************************************
	// 1.1 init TaskConfig.GetTaskPath()
	_ = os.RemoveAll(config.GetTaskPath())
	_ = os.MkdirAll(config.GetTaskPath(), 0777)
	// 1.2 create logger
	loggerPath := path.Join(config.GetTaskPath(), "task.log")
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	file, err := os.OpenFile(loggerPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return nil, errors.Wrap(err, "[RunTaskGaussDB]create logger error")
	}
	defer file.Close()
	writers := []io.Writer{file}
	multiWriter := io.MultiWriter(writers...)
	logger.SetOutput(multiWriter)
	logger.SetLevel(logrus.InfoLevel)

	// 1.3 create connector for GaussDB-M
	logger.Info("create GaussDB connector (dbms: ", config.DBMS, ")")
	conn, err := connector.NewOpenGaussConnector(config.Host, config.Port, config.Username, config.Password, config.DbName)
	if err != nil {
		logger.Error("create connector error: " + err.Error())
		return nil, err
	}

	// 1.5 read ddl, execute ddl (no InitDB for GaussDB)
	logger.Info("init ddl")
	ddlData, err := ioutil.ReadFile(config.DDLPath)
	if err != nil {
		logger.Error("read ddl error: " + err.Error())
		return nil, errors.Wrap(err, "[RunTaskGaussDB]read ddl error")
	}
	ddlSqls := connector.ExtractSQL(string(ddlData))
	// Execute DDL directly (database already exists in M mode)
	for _, ddlSql := range ddlSqls {
		result := conn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			logger.Error("ddl exec error: " + result.Err.Error() + " sql: " + ddlSql.Sql)
			// Continue for some errors (e.g., DROP TABLE IF EXISTS when table doesn't exist)
		}
	}

	// 1.6 read dml
	logger.Info("init dml")
	dmlData, err := ioutil.ReadFile(config.DMLPath)
	if err != nil {
		logger.Error("read dml error: " + err.Error())
		return nil, errors.Wrap(err, "[RunTaskGaussDB]read dml error")
	}
	dmlSqls := connector.ExtractSQL(string(dmlData))
	// **************************************************
	endInitTime := time.Now().String()
	// end 1

	// 2. run
	logger.Info("Running **************************************************")
	taskResult := &TaskResult{
		StartTime:              startTime,
		DDLSqlsNum:             len(ddlSqls),
		DMLSqlsNum:             len(dmlSqls),
		EndInitTime:            endInitTime,
		Stage1ErrNum:           0,
		Stage1ExecErrNum:       0,
		Stage2ErrNum:           0,
		Stage2UnitNum:          0,
		Stage2UnitErrNum:       0,
		Stage2UnitExecErrNum:   0,
		ImpoBugsNum:            0,
		SaveBugErrNum:          0,
		EndTime:                "",
		PotentialFalsePositivesNum: 0,
		FalsePositiveDetails:   make([]FalsePositiveRecord, 0),
	}
	// Create false positive detector (v0.5.0 ported to GaussDB-M)
	fpDetector := oracle.NewFalsePositiveDetector(conn, 3, 0.67, 5)
	// **************************************************
	// for each sql, do:
	total := len(dmlSqls)
	cur := 0
	for i, dmlSql := range dmlSqls {

		// rate
		if cur > total/20 {
			cur = 0
			logger.Info("[Rate]", i, "/", total)
		} else {
			cur += 1
		}

		// 2.1 stage1.InitAndExecForMMode
		stage1Result := stage1.InitAndExecForMMode(dmlSql.Sql, conn)

		// Handle skipped SQL (M-mode unsupported syntax)
		if stage1Result.Skipped {
			taskResult.Stage1SkippedNum += 1
			logger.Info("[Skipped] sqlId = ", dmlSql.Id, " reason: ", stage1Result.SkipReason)
			continue
		}

		// handle stage1 error
		if stage1Result.Err != nil {
			taskResult.Stage1ErrNum += 1
			continue
		}
		// handle stage1 execute error
		if stage1Result.ExecResult.Err != nil {
			taskResult.Stage1ExecErrNum += 1
			continue
		}

		originalSql := stage1Result.InitSql
		originalResult := stage1Result.ExecResult

		// 2.2 stage2.MutateAllAndExecForMMode
		stage2Result := stage2.MutateAllAndExecForMMode(originalSql, config.Seed+int64(i), conn)
		// handle stage2 error
		if stage2Result.Err != nil {
			taskResult.Stage2ErrNum += 1
			continue
		}
		// for each mutation unit
		taskResult.Stage2UnitNum += len(stage2Result.MutateUnits)
		for _, mutateUnit := range stage2Result.MutateUnits {
			// handle stage2 unit error
			if mutateUnit.Err != nil {
				taskResult.Stage2UnitErrNum += 1
				continue
			}
			// handle stage2 unit exec error
			if mutateUnit.ExecResult.Err != nil {
				taskResult.Stage2UnitExecErrNum += 1

				// Error Oracle (v0.6.0): if original succeeded but upper mutation errors
				if mutateUnit.IsUpper && originalResult.Err == nil {
					reExecResult := conn.ExecSQL(mutateUnit.Sql)
					if reExecResult.Err != nil {
						bugId := taskResult.ImpoBugsNum
						taskResult.ImpoBugsNum += 1
						taskResult.ErrorOracleBugsNum += 1

						errMsg := mutateUnit.ExecResult.Err.Error()
						logger.Info("ERROR ORACLE bug! bugId=", bugId, " sqlId=", dmlSql.Id,
							" mutation=", mutateUnit.Name, " error=", errMsg)

						bugReport := &BugReport{
							ReportTime:     time.Now().String(),
							BugId:          bugId,
							SqlId:          dmlSql.Id,
							MutationName:   mutateUnit.Name,
							IsUpper:        mutateUnit.IsUpper,
							OriginalSql:    originalSql,
							OriginalResult: originalResult,
							MutatedSql:     mutateUnit.Sql,
							MutatedResult:  mutateUnit.ExecResult,
							IsErrorOracle:  true,
							ErrorMsg:       errMsg,
						}
						err2 := bugReport.SaveBugReport(config.GetTaskBugsPath())
						if err2 != nil {
							taskResult.SaveBugErrNum += 1
							logger.Error("save error oracle bug error: ", err2)
						}
					}
				}
				continue
			}

			mutationName := mutateUnit.Name
			isUpper := mutateUnit.IsUpper
			mutatedSql := mutateUnit.Sql
			mutatedResult := mutateUnit.ExecResult

			//   2.3 use oracle to detect logical bugs
			// For implication mutations, use Check (result containment relationship)
			// Implication Oracle: check containment relationship
			check, oracleErr := oracle.Check(originalResult, mutatedResult, isUpper)
			if oracleErr != nil {
				logger.Warn("oracle check error for sqlId=", dmlSql.Id, " mutation=", mutationName, ": ", oracleErr)
				continue
			}
			if !check {
				// logical bug!!!
				bugId := taskResult.ImpoBugsNum
				taskResult.ImpoBugsNum += 1

				logger.Info("logical bug!!! bugId = ", bugId, " sqlId = ", dmlSql.Id, " mutationName = ", mutationName)
				if publicLogger != nil {
					publicLogger.Info("task-", strconv.Itoa(config.TaskId), " detected a logical bug!!! bugId = ",
						bugId, " sqlId = ", dmlSql.Id, " mutationName = ", mutationName)
				}

				// False positive detection (v0.5.0 ported to GaussDB-M)
				fpAnalysis, fpErr := fpDetector.AnalyzePotentialFalsePositive(
					originalSql, mutatedSql, isUpper, mutationName)
				if fpErr != nil {
					logger.Warn("False positive analysis failed for bug-", bugId, ": ", fpErr)
				} else if fpAnalysis.IsPotentialFalsePositive {
					taskResult.PotentialFalsePositivesNum += 1
					fpRecord := FalsePositiveRecord{
						BugId:           bugId,
						SqlId:           dmlSql.Id,
						MutationName:    mutationName,
						OriginalSql:     originalSql,
						MutatedSql:      mutatedSql,
						OriginalRows:    fpAnalysis.OriginalRows,
						MutatedRows:     fpAnalysis.MutatedRows,
						IsUpper:         isUpper,
						ReExecutions:    fpAnalysis.TotalExecutions,
						ConsistentCount: fpAnalysis.ConsistentCount,
						SuspicionReason: fpAnalysis.SuspicionReason,
						Timestamp:       time.Now().Format(time.RFC3339),
					}
					taskResult.FalsePositiveDetails = append(taskResult.FalsePositiveDetails, fpRecord)
					logger.Warn("Potential false positive detected! bugId=", bugId,
						" reason=", fpAnalysis.SuspicionReason,
						" consistency=", fpAnalysis.ConsistentCount, "/", fpAnalysis.TotalExecutions)
				}

				bugReport := &BugReport{
					ReportTime:     time.Now().String(),
					BugId:          bugId,
					SqlId:          dmlSql.Id,
					MutationName:   mutationName,
					IsUpper:        isUpper,
					OriginalSql:    originalSql,
					OriginalResult: originalResult,
					MutatedSql:     mutatedSql,
					MutatedResult:  mutatedResult,
				}
				err := bugReport.SaveBugReport(config.GetTaskBugsPath())
				if err != nil {
					taskResult.SaveBugErrNum += 1
					logger.Error("[Save Bug Error] ", err)
					continue
				}
			}
		}
	}
	// 2.4 save task result
	taskResult.EndTime = time.Now().String()
	err = taskResult.SaveTaskResult(config.GetTaskPath())
	if err != nil {
		logger.Error("[Save Result Error] ", err)
	}
	// **************************************************
	logger.Info("Finished **************************************************")

	// Close connection
	conn.Close()

	return taskResult, nil
}

// RunTaskUniversal: Universal task runner supporting multiple database types
func RunTaskUniversal(config *TaskConfig, publicLogger *logrus.Logger) (*TaskResult, error) {
	// Determine database type and call appropriate function
	switch config.DBMS {
	case "mysql", "mariadb", "tidb", "oceanbase":
		return RunTask(config, nil, publicLogger)
	case "postgresql":
		return RunTaskPostgreSQL(config, nil, publicLogger)
	case "opengauss_m", "gaussdb_m":
		return RunTaskGaussDB(config, publicLogger)
	case "opengauss_a", "gaussdb_a":
		return RunTaskGaussDBA(config, publicLogger)
	default:
		return nil, errors.New("unsupported database type: " + config.DBMS)
	}
}