package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
	"strings"
)

// EET COALESCE→CASE and NULLIF→CASE transformation mutations (MySQL)

// addFixMCoalesceToCase: FixMCoalesceToCase, *ast.FuncCallExpr: COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
// Semantically equivalent. If result sets differ → bug detected.
func (v *MutateVisitor) addFixMCoalesceToCase(in *ast.FuncCallExpr, flag int) {
	if in != nil && isCoalesceFunc(in) && len(in.Args) >= 2 {
		v.addCandidate(FixMCoalesceToCase, 1, in, flag)
	}
}

// addFixMNullifToCase: FixMNullifToCase, *ast.FuncCallExpr: NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
// Semantically equivalent. If result sets differ → bug detected.
func (v *MutateVisitor) addFixMNullifToCase(in *ast.FuncCallExpr, flag int) {
	if in != nil && isNullifFunc(in) && len(in.Args) == 2 {
		v.addCandidate(FixMNullifToCase, 1, in, flag)
	}
}

// isCoalesceFunc: check if a FuncCallExpr is COALESCE
func isCoalesceFunc(in *ast.FuncCallExpr) bool {
	if in == nil {
		return false
	}
	return strings.EqualFold(in.FnName.L, "coalesce")
}

// isNullifFunc: check if a FuncCallExpr is NULLIF
func isNullifFunc(in *ast.FuncCallExpr) bool {
	if in == nil {
		return false
	}
	return strings.EqualFold(in.FnName.L, "nullif")
}

// doFixMCoalesceToCase: FixMCoalesceToCase, COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
// For multi-argument COALESCE: COALESCE(a, b, c) → CASE WHEN a IS NOT NULL THEN a
//   ELSE CASE WHEN b IS NOT NULL THEN b ELSE c END END
func doFixMCoalesceToCase(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.FuncCallExpr:
		expr := in.(*ast.FuncCallExpr)
		if !isCoalesceFunc(expr) {
			return nil, errors.New("[FixMCoalesceToCase]expected COALESCE function")
		}

		oldArgs := expr.Args

		// Build CASE WHEN equivalent for COALESCE
		// COALESCE(a, b) → CASE WHEN a IS NOT NULL THEN a ELSE b END
		// For 3+ args, recursively nest CASE expressions
		caseExpr := buildCoalesceCaseExpr(oldArgs)

		parenExpr := &ast.ParenthesesExpr{
			Expr: caseExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMCoalesceToCase]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)
		expr.Args = oldArgs

		return sql, nil
	case nil:
		return nil, errors.New("[FixMCoalesceToCase]type nil")
	default:
		return nil, errors.New("[FixMCoalesceToCase]type default " + reflect.TypeOf(in).String())
	}
}

// buildCoalesceCaseExpr: recursively build CASE expression equivalent to COALESCE
// COALESCE(a1, a2, ..., an) → CASE WHEN a1 IS NOT NULL THEN a1
//   ELSE COALESCE(a2, ..., an) END
//   → CASE WHEN a1 IS NOT NULL THEN a1
//     ELSE CASE WHEN a2 IS NOT NULL THEN a2 ELSE ... END END
func buildCoalesceCaseExpr(args []ast.ExprNode) *ast.CaseExpr {
	if len(args) == 0 {
		// Should not happen, but return NULL as fallback
		return &ast.CaseExpr{
			ElseClause: &test_driver.ValueExpr{Datum: test_driver.NewDatum(nil)},
		}
	}

	if len(args) == 1 {
		// COALESCE(a) → a (single argument, just return it)
		// But we still wrap in CASE for consistency
		return &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr:   &ast.IsNullExpr{Expr: args[0], Not: true}, // a IS NOT NULL
					Result: args[0],
				},
			},
			ElseClause: &test_driver.ValueExpr{Datum: test_driver.NewDatum(nil)},
		}
	}

	// COALESCE(a1, a2, ..., an) → CASE WHEN a1 IS NOT NULL THEN a1 ELSE (recursive) END
	whenClause := &ast.WhenClause{
		Expr:   &ast.IsNullExpr{Expr: args[0], Not: true}, // a1 IS NOT NULL
		Result: args[0],                                    // THEN a1
	}

	// ELSE: COALESCE(a2, ..., an) → nested CASE
	elseExpr := buildCoalesceCaseExpr(args[1:])

	return &ast.CaseExpr{
		WhenClauses: []*ast.WhenClause{whenClause},
		ElseClause:  elseExpr,
	}
}

// doFixMNullifToCase: FixMNullifToCase, NULLIF(a, b) → CASE WHEN a = b THEN NULL ELSE a END
func doFixMNullifToCase(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.FuncCallExpr:
		expr := in.(*ast.FuncCallExpr)
		if !isNullifFunc(expr) {
			return nil, errors.New("[FixMNullifToCase]expected NULLIF function")
		}
		if len(expr.Args) != 2 {
			return nil, errors.New("[FixMNullifToCase]expected 2 arguments")
		}

		oldArgs := expr.Args
		a := oldArgs[0]
		b := oldArgs[1]

		// CASE WHEN a = b THEN NULL ELSE a END
		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &ast.BinaryOperationExpr{
						Op: opcode.EQ,
						L:  a,
						R:  b,
					},
					Result: &test_driver.ValueExpr{Datum: test_driver.NewDatum(nil)}, // THEN NULL
				},
			},
			ElseClause: a, // ELSE a
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: caseExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMNullifToCase]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)
		expr.Args = oldArgs

		return sql, nil
	case nil:
		return nil, errors.New("[FixMNullifToCase]type nil")
	default:
		return nil, errors.New("[FixMNullifToCase]type default " + reflect.TypeOf(in).String())
	}
}