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
	"time"
)

// RunTaskGaussDBA: Run task for GaussDB/openGauss A mode (Oracle compatibility)
// Uses PostgreSQL protocol with pgx driver
// Supports Oracle-specific syntax: ROWNUM, (+) outer join, NVL, DECODE, etc.
func RunTaskGaussDBA(config *TaskConfig, publicLogger *logrus.Logger) (*TaskResult, error) {
	// 1. Initialize
	startTime := time.Now().String()

	// 1.1 Create task directory
	_ = os.RemoveAll(config.GetTaskPath())
	_ = os.MkdirAll(config.GetTaskPath(), 0777)

	// 1.2 Create logger
	loggerPath := path.Join(config.GetTaskPath(), "task.log")
	logger := logrus.New()
	logger.SetFormatter(&logrus.TextFormatter{
		DisableColors: true,
		FullTimestamp: true,
	})
	file, err := os.OpenFile(loggerPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0777)
	if err != nil {
		return nil, errors.Wrap(err, "[RunTaskGaussDBA]create logger error")
	}
	defer file.Close()
	writers := []io.Writer{file}
	multiWriter := io.MultiWriter(writers...)
	logger.SetOutput(multiWriter)
	logger.SetLevel(logrus.InfoLevel)

	// 1.3 Create connector for GaussDB-A
	logger.Info("create GaussDB-A connector (dbms: ", config.DBMS, ")")
	conn, err := connector.NewGaussDBAConnector(config.Host, config.Port, config.Username, config.Password, config.DbName)
	if err != nil {
		logger.Error("create connector error: " + err.Error())
		return nil, err
	}

	// 1.4 Read and execute DDL (no InitDB for GaussDB A mode)
	logger.Info("init ddl")
	ddlData, err := ioutil.ReadFile(config.DDLPath)
	if err != nil {
		logger.Error("read ddl error: " + err.Error())
		return nil, errors.Wrap(err, "[RunTaskGaussDBA]read ddl error")
	}
	ddlSqls := connector.ExtractSQL(string(ddlData))
	// Execute DDL directly (database already exists in A mode)
	for _, ddlSql := range ddlSqls {
		result := conn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			logger.Error("ddl exec error: " + result.Err.Error() + " sql: " + ddlSql.Sql)
			// Continue for some errors
		}
	}

	// 1.5 Read DML
	logger.Info("init dml")
	dmlData, err := ioutil.ReadFile(config.DMLPath)
	if err != nil {
		logger.Error("read dml error: " + err.Error())
		return nil, errors.Wrap(err, "[RunTaskGaussDBA]read dml error")
	}
	dmlSqls := connector.ExtractSQL(string(dmlData))

	endInitTime := time.Now().String()

	// 2. Run mutation testing
	logger.Info("Running **************************************************")
	taskResult := &TaskResult{
		StartTime:              startTime,
		DDLSqlsNum:             len(ddlSqls),
		DMLSqlsNum:             len(dmlSqls),
		EndInitTime:            endInitTime,
		Stage1ErrNum:           0,
		Stage1ExecErrNum:       0,
		Stage1SkippedNum:       0, // A mode: track skipped SQLs
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
	// Create false positive detector (v0.5.0 ported to GaussDB-A)
	fpDetector := oracle.NewFalsePositiveDetector(conn, 3, 0.67, 5)

	// For each DML SQL, do mutation testing
	total := len(dmlSqls)
	cur := 0
	for i, dmlSql := range dmlSqls {

		// Progress rate
		if cur > total/20 {
			cur = 0
			logger.Info("[Rate]", i, "/", total)
		} else {
			cur += 1
		}

		// 2.1 Stage1.InitAndExecForAMode
		stage1Result := stage1.InitAndExecForAMode(dmlSql.Sql, conn)

		// Handle skipped SQL (CONNECT BY, PL/SQL, etc.)
		if stage1Result.Skipped {
			taskResult.Stage1SkippedNum += 1
			logger.Info("[Skipped] sqlId = ", dmlSql.Id, " reason: ", stage1Result.SkipReason)
			continue
		}

		// Handle stage1 error
		if stage1Result.Err != nil {
			taskResult.Stage1ErrNum += 1
			continue
		}

		// Handle stage1 execute error
		if stage1Result.ExecResult.Err != nil {
			taskResult.Stage1ExecErrNum += 1
			continue
		}

		originalSql := stage1Result.InitSql
		originalResult := stage1Result.ExecResult

		// 2.2 Stage2.MutateAllAndExec
		// Note: For A mode, we use standard Stage2 mutations
		// Additional A-mode specific mutations (ROWNUM) are handled separately
		stage2Result := stage2.MutateAllAndExec(originalSql, config.Seed+int64(i), conn)

		// Handle stage2 error
		if stage2Result.Err != nil {
			taskResult.Stage2ErrNum += 1
			continue
		}

		// For each mutation unit (prioritized by historical hit rate)
		taskResult.Stage2UnitNum += len(stage2Result.MutateUnits)
		stage2.GlobalMutationStats.PrioritizeUnits(stage2Result.MutateUnits)
		mCtx := &MutationLoopContext{
			Conn: conn, Config: config, Logger: logger, PublicLogger: publicLogger,
			TaskResult: taskResult, FpDetector: fpDetector,
		}
		for _, mu := range stage2Result.MutateUnits {
			adapter := &MutateUnitAdapter{
				Name: mu.Name, Sql: mu.Sql, IsUpper: mu.IsUpper,
				Err: mu.Err, ExecResult: mu.ExecResult,
			}
			processMutationUnit(adapter, dmlSql.Id, originalSql, originalResult, mCtx)
		}
	}

	// 2.4 Save task result
	taskResult.EndTime = time.Now().String()
	err = taskResult.SaveTaskResult(config.GetTaskPath())
	if err != nil {
		logger.Error("[Save Result Error] ", err)
	}

	logger.Info("Finished **************************************************")

	// Close connection
	conn.Close()

	return taskResult, nil
}