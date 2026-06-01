package task

import (
	"github.com/pkg/errors"
	"github.com/qaqcatz/impomysql/connector"
	"github.com/qaqcatz/impomysql/generator"
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

// RunTaskPostgreSQL: Run task for native PostgreSQL database
// Uses pgx/v5 driver and pg_query_go parser
// If publicConn is provided, use it instead of creating a new one
func RunTaskPostgreSQL(config *TaskConfig, publicConn *connector.PostgreSQLConnector, publicLogger *logrus.Logger) (*TaskResult, error) {
	// 1 init
	startTime := time.Now().String()

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
		return nil, errors.Wrap(err, "[RunTaskPostgreSQL]create logger error")
	}
	defer file.Close()
	writers := []io.Writer{file}
	multiWriter := io.MultiWriter(writers...)
	logger.SetOutput(multiWriter)
	logger.SetLevel(logrus.InfoLevel)

	// 1.3 create PostgreSQL connector
	var conn *connector.PostgreSQLConnector = nil
	if publicConn == nil {
		logger.Info("create PostgreSQL connector")
		conn, err = connector.NewPostgreSQLConnector(config.Host, config.Port, config.Username, config.Password, config.DbName)
		if err != nil {
			logger.Error("create connector error: " + err.Error())
			return nil, err
		}
	} else {
		logger.Info("use public PostgreSQL connector")
		conn = publicConn
	}

	// 1.4b random SQL generation mode
	if config.GenMode != "" {
		logger.Info("random SQL generation mode: " + config.GenMode)
		schemaInfo, err := conn.DiscoverSchema()
		if err != nil {
			logger.Error("discover schema error: " + err.Error())
			return nil, errors.Wrap(err, "[RunTaskPostgreSQL]discover schema error")
		}
		if len(schemaInfo.Tables) == 0 {
			logger.Error("no tables found in database")
			return nil, errors.New("[RunTaskPostgreSQL]no tables found in database for random generation")
		}

		genConfig := &generator.GeneratorConfig{
			Seed:           config.GenSeed,
			MaxDepth:       config.GenDepth,
			QueriesNum:     config.GenQueries,
				Dialect:        config.DBMS,
			EnableJoin:     config.GenJoin,
			EnableSubquery: config.GenSubquery,
			EnableUnion:    config.GenUnion,
			EnableCTE:      config.GenCTE,
			EnableGroupBy:  config.GenGroupBy,
			EnableOrderBy:  config.GenOrderBy,
			EnableLimit:    config.GenLimit,
		}
		if !genConfig.EnableJoin && !genConfig.EnableSubquery && !genConfig.EnableUnion &&
			!genConfig.EnableCTE && !genConfig.EnableGroupBy {
			genConfig.EnableJoin = true
			genConfig.EnableSubquery = true
			genConfig.EnableUnion = true
			genConfig.EnableCTE = true
			genConfig.EnableGroupBy = true
			genConfig.EnableOrderBy = true
			genConfig.EnableLimit = true
		}

		gen := generator.NewQueryGenerator(genConfig, schemaInfo)

		ddlSqls := gen.GenerateDDL()
		ddlContent := ""
		for _, sql := range ddlSqls {
			ddlContent += sql + ";\n"
		}
		ddlPath := path.Join(config.GetTaskPath(), "gen_ddl.sql")
		err = ioutil.WriteFile(ddlPath, []byte(ddlContent), 0777)
		if err != nil {
			logger.Error("write ddl error: " + err.Error())
			return nil, errors.Wrap(err, "[RunTaskPostgreSQL]write generated ddl error")
		}
		config.DDLPath = ddlPath

		dmlSqls := gen.GenerateSelects(config.GenQueries)
		dmlContent := ""
		for _, sql := range dmlSqls {
			dmlContent += sql + ";\n"
		}
		dmlPath := path.Join(config.GetTaskPath(), "gen_dml.sql")
		err = ioutil.WriteFile(dmlPath, []byte(dmlContent), 0777)
		if err != nil {
			logger.Error("write dml error: " + err.Error())
			return nil, errors.Wrap(err, "[RunTaskPostgreSQL]write generated dml error")
		}
		config.DMLPath = dmlPath

		logger.Info("generated DDL: ", len(ddlSqls), " statements, DML: ", len(dmlSqls), " queries")
	}

	// 1.5 read ddl, execute ddl
	var ddlSqls []*connector.EachSql
	logger.Info("init ddl")
	if config.GenMode != "" {
		ddlData, err := ioutil.ReadFile(config.DDLPath)
		if err == nil {
			ddlSqls = connector.ExtractSQL(string(ddlData))
		}
		logger.Info("genMode: skip DDL execution (tables already exist), DDL count: ", len(ddlSqls))
	} else {
		ddlData, err := ioutil.ReadFile(config.DDLPath)
		if err != nil {
			logger.Error("read ddl error: " + err.Error())
			return nil, errors.Wrap(err, "[RunTaskPostgreSQL]read ddl error")
		}
		ddlSqls = connector.ExtractSQL(string(ddlData))

	// Execute DDL (database should already exist)
	for _, ddlSql := range ddlSqls {
		result := conn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			logger.Error("ddl exec error: " + result.Err.Error() + " sql: " + ddlSql.Sql)
			// Continue for ignorable errors (e.g., DROP TABLE IF EXISTS)
		}
	}

		}
	// 1.6 read dml
	logger.Info("init dml")
	dmlData, err := ioutil.ReadFile(config.DMLPath)
	if err != nil {
		logger.Error("read dml error: " + err.Error())
		return nil, errors.Wrap(err, "[RunTaskPostgreSQL]read dml error")
	}
	dmlSqls := connector.ExtractSQL(string(dmlData))

	endInitTime := time.Now().String()

	// 2. run
	logger.Info("Running **************************************************")
	taskResult := &TaskResult{
		StartTime:              startTime,
		DDLSqlsNum:             len(ddlSqls),
		DMLSqlsNum:             len(dmlSqls),
		EndInitTime:            endInitTime,
		Stage1ErrNum:           0,
		Stage1ExecErrNum:       0,
		Stage1SkippedNum:       0,
		Stage2ErrNum:           0,
		Stage2UnitNum:          0,
		Stage2UnitErrNum:       0,
		Stage2UnitExecErrNum:   0,
		ImpoBugsNum:            0,
		SaveBugErrNum:          0,
		EndTime:                "",
	}

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

		// 2.1 stage1.InitForPostgreSQLAndExec
		stage1Result := stage1.InitForPostgreSQLAndExec(dmlSql.Sql, conn)
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

		// 2.2 stage2.MutateAllAndExecForPostgreSQL
		stage2Result := stage2.MutateAllAndExecForPostgreSQL(originalSql, config.Seed+int64(i), conn)
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
				continue
			}

			mutationName := mutateUnit.Name
			isUpper := mutateUnit.IsUpper
			mutatedSql := mutateUnit.Sql
			mutatedResult := mutateUnit.ExecResult

			// 2.3 use oracle.Check to detect logical bugs
			check, err := oracle.Check(originalResult, mutatedResult, isUpper)
			if err != nil {
				return nil, err
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

	logger.Info("Finished **************************************************")

	// Close connection only if we created it
	if publicConn == nil {
		conn.Close()
	}

	return taskResult, nil
}