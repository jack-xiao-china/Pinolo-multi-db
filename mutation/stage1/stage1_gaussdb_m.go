package stage1

import (
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/qaqcatz/impomysql/connector"
)

// MModeInitResult: result for GaussDB-M mode Stage1 initialization
type MModeInitResult struct {
	InitSql    string
	Err        error
	ExecResult *connector.Result
	Skipped    bool   // true if SQL contains unsupported syntax
	SkipReason string // reason for skipping
}

// InitForMMode: Stage1 preprocessing for GaussDB-M (MySQL compatibility mode)
//
// Currently uses standard MySQL Stage1 transformations (remove aggregate, window,
// LEFT/RIGHT JOIN, LIMIT, uncertain functions). M-mode specific preprocessing
// (TOP clause removal, M-specific uncertain functions) relies on the standard
// rmUncertain which already handles SYSDATE, etc.
//
// Future enhancements may add:
// - TOP n clause removal (GaussDB-M specific syntax)
// - M-mode specific implicit type conversion risk detection
// - Behavioral difference-aware expression simplification
func InitForMMode(sql string) *MModeInitResult {
	initResult := &MModeInitResult{
		InitSql:    "",
		Err:        nil,
		ExecResult: nil,
		Skipped:    false,
		SkipReason: "",
	}

	// Delegate to standard MySQL Stage1 Init
	mysqlInitResult := Init(sql)
	if mysqlInitResult.Err != nil {
		initResult.Err = mysqlInitResult.Err
		return initResult
	}

	initResult.InitSql = mysqlInitResult.InitSql
	return initResult
}

// InitAndExecForMMode: InitForMMode + exec
func InitAndExecForMMode(sql string, conn connector.SQLExecutor) *MModeInitResult {
	initResult := InitForMMode(sql)
	if initResult.Err != nil || initResult.Skipped {
		return initResult
	}
	result := conn.ExecSQL(initResult.InitSql)
	initResult.ExecResult = result
	return initResult
}