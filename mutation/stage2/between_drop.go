package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// BETWEEN drop-bound implication mutations (MySQL)
// These are Implication Oracle mutations: dropping a bound expands the result set.

// addFixMBetweenDropUpperU: FixMBetweenDropUpperU, *ast.BetweenExpr: x BETWEEN a AND b → x >= a
// Implication (upper mutation): BETWEEN result ⊆ >= result (satisfying both bounds ⊆ satisfying lower bound)
// For NOT BETWEEN: x NOT BETWEEN a AND b → x < a (satisfying NOT BETWEEN ⊆ satisfying < lower)
func (v *MutateVisitor) addFixMBetweenDropUpperU(in *ast.BetweenExpr, flag int) {
	if in != nil && in.Expr != nil && in.Left != nil {
		v.addCandidate(FixMBetweenDropUpperU, 1, in, flag)
	}
}

// addFixMBetweenDropLowerU: FixMBetweenDropLowerU, *ast.BetweenExpr: x BETWEEN a AND b → x <= b
// Implication (upper mutation): BETWEEN result ⊆ <= result (satisfying both bounds ⊆ satisfying upper bound)
// For NOT BETWEEN: x NOT BETWEEN a AND b → x > b (satisfying NOT BETWEEN ⊆ satisfying > upper)
func (v *MutateVisitor) addFixMBetweenDropLowerU(in *ast.BetweenExpr, flag int) {
	if in != nil && in.Expr != nil && in.Right != nil {
		v.addCandidate(FixMBetweenDropLowerU, 1, in, flag)
	}
}

// doFixMBetweenDropUpperU: FixMBetweenDropUpperU, x BETWEEN a AND b -> x >= a (upper: drop upper bound)
// Containment: x BETWEEN a AND b ⊆ x >= a
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x < a
func doFixMBetweenDropUpperU(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.BetweenExpr:
		expr := in.(*ast.BetweenExpr)

		oldExpr := expr.Expr
		oldLeft := expr.Left
		oldNot := expr.Not

		var resultExpr ast.ExprNode

		if !oldNot {
			// BETWEEN -> x >= a (drop upper bound)
			resultExpr = &ast.BinaryOperationExpr{
				Op: opcode.GE,
				L:  oldExpr,
				R:  oldLeft,
			}
		} else {
			// NOT BETWEEN -> x < a (NOT BETWEEN ⊆ < lower bound)
			resultExpr = &ast.BinaryOperationExpr{
				Op: opcode.LT,
				L:  oldExpr,
				R:  oldLeft,
			}
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: resultExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMBetweenDropUpperU]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)
		expr.Expr = oldExpr
		expr.Left = oldLeft
		expr.Not = oldNot

		return sql, nil
	case nil:
		return nil, errors.New("[FixMBetweenDropUpperU]type nil")
	default:
		return nil, errors.New("[FixMBetweenDropUpperU]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMBetweenDropLowerU: FixMBetweenDropLowerU, x BETWEEN a AND b -> x <= b (upper: drop lower bound)
// Containment: x BETWEEN a AND b ⊆ x <= b
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x > b
func doFixMBetweenDropLowerU(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.BetweenExpr:
		expr := in.(*ast.BetweenExpr)

		oldExpr := expr.Expr
		oldRight := expr.Right
		oldNot := expr.Not

		var resultExpr ast.ExprNode

		if !oldNot {
			// BETWEEN -> x <= b (drop lower bound)
			resultExpr = &ast.BinaryOperationExpr{
				Op: opcode.LE,
				L:  oldExpr,
				R:  oldRight,
			}
		} else {
			// NOT BETWEEN -> x > b (NOT BETWEEN ⊆ > upper bound)
			resultExpr = &ast.BinaryOperationExpr{
				Op: opcode.GT,
				L:  oldExpr,
				R:  oldRight,
			}
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: resultExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMBetweenDropLowerU]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)
		expr.Expr = oldExpr
		expr.Right = oldRight
		expr.Not = oldNot

		return sql, nil
	case nil:
		return nil, errors.New("[FixMBetweenDropLowerU]type nil")
	default:
		return nil, errors.New("[FixMBetweenDropLowerU]type default " + reflect.TypeOf(in).String())
	}
}
