package task

import (
	"strconv"
	"strings"
	"time"

	"github.com/qaqcatz/impomysql/connector"
	"github.com/qaqcatz/impomysql/mutation/oracle"
	"github.com/qaqcatz/impomysql/mutation/stage2"
	"github.com/sirupsen/logrus"
)

// MutationUnitLike: common interface for MySQL MutateUnit and PostgreSQL PgMutateUnit.
// Both types have identical fields but are defined as separate structs.
type MutationUnitLike interface {
	GetName() string
	GetSql() string
	GetIsUpper() bool
	GetErr() error
	GetExecResult() *connector.Result
}

// MutateUnitAdapter: wraps *stage2.MutateUnit to implement MutationUnitLike
type MutateUnitAdapter struct {
	Name       string
	Sql        string
	IsUpper    bool
	Err        error
	ExecResult *connector.Result
}

func (a *MutateUnitAdapter) GetName() string                  { return a.Name }
func (a *MutateUnitAdapter) GetSql() string                   { return a.Sql }
func (a *MutateUnitAdapter) GetIsUpper() bool                 { return a.IsUpper }
func (a *MutateUnitAdapter) GetErr() error                    { return a.Err }
func (a *MutateUnitAdapter) GetExecResult() *connector.Result { return a.ExecResult }

// MutationLoopContext: shared context for the mutation testing loop body
type MutationLoopContext struct {
	Conn           connector.SQLExecutor
	Config         *TaskConfig
	Logger         *logrus.Logger
	PublicLogger   *logrus.Logger
	TaskResult     *TaskResult
	FpDetector     *oracle.FalsePositiveDetector
	AggregateMode  bool // use aggregate-aware Oracle
}

// processMutationUnit: common logic for processing a single mutation unit.
// This is the shared body of the inner loop across all 4 task runners.
// Returns nothing; updates taskResult in place.
func processMutationUnit(
	unit MutationUnitLike,
	dmlSqlId int,
	originalSql string,
	originalResult *connector.Result,
	ctx *MutationLoopContext,
) {
	taskResult := ctx.TaskResult
	conn := ctx.Conn
	logger := ctx.Logger
	publicLogger := ctx.PublicLogger
	config := ctx.Config
	fpDetector := ctx.FpDetector

	// handle stage2 unit error
	if unit.GetErr() != nil {
		taskResult.Stage2UnitErrNum += 1
		return
	}

	// handle stage2 unit exec error
	if unit.GetExecResult().Err != nil {
		taskResult.Stage2UnitExecErrNum += 1
		stage2.GlobalMutationStats.RecordResult(unit.GetName(), false, true)

		// Error Oracle (v0.6.0): if original succeeded but upper mutation errors
		// Filter out expected errors that are natural consequences of mutations,
		// not actual DBMS logical bugs.
		if unit.GetIsUpper() && originalResult.Err == nil && !isExpectedMutationError(unit.GetExecResult().Err) {
			reExecResult := conn.ExecSQL(unit.GetSql())
			if reExecResult.Err != nil && !isExpectedMutationError(reExecResult.Err) {
				bugId := taskResult.ImpoBugsNum
				taskResult.ImpoBugsNum += 1
				taskResult.ErrorOracleBugsNum += 1
				stage2.GlobalMutationStats.RecordResult(unit.GetName(), true, true)

				errMsg := unit.GetExecResult().Err.Error()
				logger.Info("ERROR ORACLE bug! bugId=", bugId, " sqlId=", dmlSqlId,
					" mutation=", unit.GetName(), " error=", errMsg)

				bugReport := &BugReport{
					ReportTime:     time.Now().String(),
					BugId:          bugId,
					SqlId:          dmlSqlId,
					MutationName:   unit.GetName(),
					IsUpper:        unit.GetIsUpper(),
					OriginalSql:    originalSql,
					OriginalResult: originalResult,
					MutatedSql:     unit.GetSql(),
					MutatedResult:  unit.GetExecResult(),
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
		return
	}

	mutationName := unit.GetName()
	isUpper := unit.GetIsUpper()
	mutatedSql := unit.GetSql()
	mutatedResult := unit.GetExecResult()

	// Implication Oracle: check containment relationship
	var check bool
	var oracleErr error
	if ctx.AggregateMode || isAggregateResult(originalResult) {
		check, oracleErr = oracle.CheckAggregate(originalResult, mutatedResult, isUpper)
	} else {
		check, oracleErr = oracle.Check(originalResult, mutatedResult, isUpper)
	}
	if oracleErr != nil {
		logger.Warn("oracle check error for sqlId=", dmlSqlId, " mutation=", mutationName, ": ", oracleErr)
		return
	}
	if !check {
		// logical bug!!!
		bugId := taskResult.ImpoBugsNum
		taskResult.ImpoBugsNum += 1
		stage2.GlobalMutationStats.RecordResult(mutationName, true, false)

		logger.Info("logical bug!!! bugId = ", bugId, " sqlId = ", dmlSqlId, " mutationName = ", mutationName)
		if publicLogger != nil {
			publicLogger.Info("task-", strconv.Itoa(config.TaskId), " detected a logical bug!!! bugId = ",
				bugId, " sqlId = ", dmlSqlId, " mutationName = ", mutationName)
		}

		// False positive detection
		if fpDetector != nil {
			fpAnalysis, fpErr := fpDetector.AnalyzePotentialFalsePositive(
				originalSql, mutatedSql, isUpper, mutationName)
			if fpErr != nil {
				logger.Warn("False positive analysis failed for bug-", bugId, ": ", fpErr)
			} else if fpAnalysis.IsPotentialFalsePositive {
				taskResult.PotentialFalsePositivesNum += 1
				fpRecord := FalsePositiveRecord{
					BugId:           bugId,
					SqlId:           dmlSqlId,
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
		}

		bugReport := &BugReport{
			ReportTime:     time.Now().String(),
			BugId:          bugId,
			SqlId:          dmlSqlId,
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
			logger.Error("save bug error: ", err)
		}
	} else {
		// No bug found: record normal execution for stats
		stage2.GlobalMutationStats.RecordResult(mutationName, false, false)
	}
}

// isExpectedMutationError: returns true if the error is an expected consequence
// of a mutation rather than a genuine DBMS logical bug.
// These errors occur because mutations can create semantically invalid SQL in certain
// contexts (e.g., relaxing = to >= in a scalar subquery makes it return multiple rows).
func isExpectedMutationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())

	// Expected SQL semantic errors caused by mutations:
	expectedPatterns := []string{
		// MySQL Error 1242: scalar subquery returns more than 1 row
		// Happens when = is relaxed to >= in subquery comparisons
		"subquery returns more than 1 row",
		// MySQL Error 1241: operand should contain 1 column(s)
		"operand should contain 1 column",
		// MySQL Error 1093: can't specify target table for update in FROM clause
		"can't specify target table",
		// PostgreSQL: more than one row returned by a subquery used as an expression
		"more than one row returned by a subquery",
		// Infrastructure / resource errors (not DBMS logical bugs):
		"no space left on device",
		"disk full",
		"out of memory",
		"out of sortmemory",
		"lock wait timeout exceeded",
		"deadlock found",
		// GaussDB/OpenGauss resource errors:
		"temporary file size exceeds",
		"could not write to file",
		"insufficient memory",
		// Go context timeout errors (queries that exceed DefaultQueryTimeout):
		"context deadline exceeded",
		"context canceled",
	}

	for _, pattern := range expectedPatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}
	return false
}

// isAggregateResult: detect if a query result looks like an aggregate query result.
// Returns true if the result has a single row with all-numeric columns.
// This auto-detects aggregate queries (COUNT, SUM, AVG, MIN, MAX) that need
// numeric comparison instead of set containment.
func isAggregateResult(result *connector.Result) bool {
	if result == nil || len(result.Rows) != 1 || len(result.Rows[0]) == 0 {
		return false
	}
	// Check if all columns are numeric
	for _, val := range result.Rows[0] {
		if val == connector.NullMarker {
			continue // NULL is acceptable
		}
		// Try to parse as float - if any column fails, it's not all-numeric
		_, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil {
			return false
		}
	}
	return true
}
