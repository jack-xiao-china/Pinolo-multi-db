package stage1

import (
	"bytes"

	"github.com/pingcap/tidb/parser"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/format"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/pkg/errors"
	"github.com/qaqcatz/impomysql/connector"
)

// AggregateInitVisitor: Stage1 visitor that preserves aggregate functions and GROUP BY.
// Applies all Stage1 transformations EXCEPT rmAgg, allowing aggregate queries to be
// tested with the aggregate-aware Oracle (CheckAggregate).
type AggregateInitVisitor struct{}

func (v *AggregateInitVisitor) Enter(in ast.Node) (ast.Node, bool) {
	// Skip rmAgg — preserve aggregate functions and GROUP BY
	rmWindow(in)
	rmLRJoin(in)
	rmLimit(in)
	rmUncertain(in)
	return in, false
}

func (v *AggregateInitVisitor) Leave(in ast.Node) (ast.Node, bool) {
	return in, true
}

// InitForAggregate: Stage1 preprocessing that preserves aggregate functions.
// Same as Init() but does NOT remove aggregate functions or GROUP BY.
// This allows testing aggregate queries with the aggregate-aware Oracle.
func InitForAggregate(sql string) *InitResult {
	initResult := &InitResult{
		InitSql:    "",
		Err:        nil,
		ExecResult: nil,
	}
	p := parser.New()
	stmtNodes, _, err := p.Parse(sql, "", "")
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForAggregate]parse error")
		return initResult
	}
	if stmtNodes == nil || len(stmtNodes) == 0 {
		initResult.Err = errors.New("[InitForAggregate]stmtNodes == nil || len(stmtNodes) == 0")
		return initResult
	}
	rootNode := &stmtNodes[0]

	switch (*rootNode).(type) {
	case *ast.SelectStmt:
	case *ast.SetOprStmt:
	default:
		initResult.Err = errors.New("[InitForAggregate]*rootNode is not *ast.SelectStmt or *ast.SetOprStmt")
		return initResult
	}

	v := &AggregateInitVisitor{}
	(*rootNode).Accept(v)

	buf := new(bytes.Buffer)
	ctx := format.NewRestoreCtx(format.DefaultRestoreFlags, buf)
	err = (*rootNode).Restore(ctx)
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForAggregate]restore error")
		return initResult
	}
	initResult.InitSql = buf.String()
	return initResult
}

// InitForAggregateAndExec: InitForAggregate + exec
func InitForAggregateAndExec(sql string, conn connector.SQLExecutor) *InitResult {
	initResult := InitForAggregate(sql)
	if initResult.Err != nil {
		return initResult
	}
	result := conn.ExecSQL(initResult.InitSql)
	initResult.ExecResult = result
	return initResult
}
