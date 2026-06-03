package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// GaussDB-M specific EET transformation mutations

// doFixMIfToCase: FixMIfToCase, IF(cond, a, b) -> CASE WHEN cond THEN a ELSE b END
func doFixMIfToCase(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.FuncCallExpr:
		funcCall := in.(*ast.FuncCallExpr)
		if funcCall.FnName.L != "if" || len(funcCall.Args) != 3 {
			return nil, errors.New("[FixMIfToCase]expected IF function with 3 args")
		}

		cond := funcCall.Args[0] // condition
		a := funcCall.Args[1]     // true branch
		b := funcCall.Args[2]     // false branch

		// Build CASE WHEN cond THEN a ELSE b END
		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr:   cond,
					Result: a,
				},
			},
			ElseClause: b,
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: caseExpr,
		}

		replaceExprInRoot(rootNode, funcCall, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMIfToCase]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, funcCall)
		return sql, nil
	case nil:
		return nil, errors.New("[FixMIfToCase]type nil")
	default:
		return nil, errors.New("[FixMIfToCase]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMConcatToPipe: FixMConcatToPipe, CONCAT(a, b) -> a || b
// Note: In GaussDB-M, || is string concatenation (same as MySQL's CONCAT in M mode)
// However, NULL handling may differ: CONCAT(NULL, b) = NULL, while NULL || b depends on M mode settings
func doFixMConcatToPipe(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.FuncCallExpr:
		funcCall := in.(*ast.FuncCallExpr)
		if funcCall.FnName.L != "concat" || len(funcCall.Args) != 2 {
			return nil, errors.New("[FixMConcatToPipe]expected CONCAT function with 2 args")
		}

		a := funcCall.Args[0]
		b := funcCall.Args[1]

		// Build a || b (string concatenation operator)
		// In TiDB parser, LogicOr represents || which in M mode is string concatenation
		pipeExpr := &ast.BinaryOperationExpr{
			Op: opcode.LogicOr, // || operator
			L:  a,
			R:  b,
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: pipeExpr,
		}

		replaceExprInRoot(rootNode, funcCall, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMConcatToPipe]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, funcCall)
		return sql, nil
	case nil:
		return nil, errors.New("[FixMConcatToPipe]type nil")
	default:
		return nil, errors.New("[FixMConcatToPipe]type default " + reflect.TypeOf(in).String())
	}
}