package stage1

import (
	"bytes"
	"github.com/pkg/errors"
	tidbparser "github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/qaqcatz/impomysql/connector"
	"github.com/qaqcatz/impomysql/parser"
)

// AModeInitResult: result for A mode initialization
type AModeInitResult struct {
	InitSql    string
	Err        error
	ExecResult *connector.Result
	Skipped    bool    // indicates if SQL was skipped due to unsupported syntax
	SkipReason string  // reason for skipping
}

// InitForAMode: preprocesses SQL for GaussDB A mode (Oracle compatibility)
// Steps:
// 1. Filter out CONNECT BY hierarchical queries
// 2. Filter out PL/SQL blocks
// 3. Normalize Oracle syntax (+) outer join, NVL, etc.
// 4. Apply standard Stage1 transformations (remove aggregates, window functions, etc.)
func InitForAMode(sql string) *AModeInitResult {
	initResult := &AModeInitResult{
		InitSql:    "",
		Err:        nil,
		ExecResult: nil,
		Skipped:    false,
		SkipReason: "",
	}

	// 1. Check for CONNECT BY - skip if present
	processedSQL, skip := RmConnectBy(sql)
	if skip {
		initResult.Skipped = true
		initResult.SkipReason = "CONNECT BY hierarchical query not supported"
		return initResult
	}

	// 2. Check for PL/SQL blocks - skip if present
	processedSQL, skip = RmPLSQL(processedSQL)
	if skip {
		initResult.Skipped = true
		initResult.SkipReason = "PL/SQL block not supported"
		return initResult
	}

	// 3. Normalize Oracle-specific syntax
	oraclePreprocessor := parser.NewOraclePreprocessor()
	processedSQL = oraclePreprocessor.Normalize(processedSQL)

	// 4. Use TiDB parser for standard Stage1 transformations
	// Note: After normalization, the SQL should be MySQL/PostgreSQL compatible
	// and TiDB parser can handle it for basic transformations
	p := tidbparser.New()
	stmtNodes, _, err := p.Parse(processedSQL, "", "")
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForAMode]parse error")
		return initResult
	}
	if stmtNodes == nil || len(stmtNodes) == 0 {
		initResult.Err = errors.New("[InitForAMode]stmtNodes is empty")
		return initResult
	}
	rootNode := &stmtNodes[0]

	// Check statement type
	switch (*rootNode).(type) {
	case *ast.SelectStmt:
	case *ast.SetOprStmt:
	default:
		initResult.Err = errors.New("[InitForAMode]statement is not SELECT or UNION")
		return initResult
	}

	// Apply standard Stage1 visitor (remove aggregates, window functions, etc.)
	v := &InitVisitor{}
	(*rootNode).Accept(v)

	// Restore SQL
	initSql, err := restoreSQL(rootNode)
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForAMode]restore error")
		return initResult
	}

	initResult.InitSql = initSql
	return initResult
}

// InitAndExecForAMode: InitForAMode + execute
func InitAndExecForAMode(sql string, conn connector.SQLExecutor) *AModeInitResult {
	initResult := InitForAMode(sql)
	if initResult.Err != nil {
		return initResult
	}
	if initResult.Skipped {
		return initResult
	}

	result := conn.ExecSQL(initResult.InitSql)
	initResult.ExecResult = result
	return initResult
}

// restoreSQL: helper to restore AST to SQL string
func restoreSQL(rootNode *ast.StmtNode) (string, error) {
	buf := new(bytes.Buffer)
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags|format.RestoreStringWithoutCharset, buf)
	err := (*rootNode).Restore(ctx)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}