package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// IS NULL / IS NOT NULL implication mutations
//
// FixMIsNullToFalseL: x IS NULL → FALSE (lower: result shrinks)
//   {rows where x IS NULL} → {} (empty set) → mutated ⊆ original
//
// FixMIsNotNullToTrueU: x IS NOT NULL → TRUE (upper: result expands)
//   {rows where x IS NOT NULL} → {all rows} → original ⊆ mutated

// addFixMIsNullToFalseL: *ast.IsNullExpr (Not=false): x IS NULL → FALSE
func (v *MutateVisitor) addFixMIsNullToFalseL(in *ast.IsNullExpr, flag int) {
	if in != nil && in.Expr != nil && !in.Not {
		v.addCandidate(FixMIsNullToFalseL, 0, in, flag)
	}
}

// addFixMIsNotNullToTrueU: *ast.IsNullExpr (Not=true): x IS NOT NULL → TRUE
func (v *MutateVisitor) addFixMIsNotNullToTrueU(in *ast.IsNullExpr, flag int) {
	if in != nil && in.Expr != nil && in.Not {
		v.addCandidate(FixMIsNotNullToTrueU, 1, in, flag)
	}
}

// doFixMIsNullToFalseL: x IS NULL → FALSE
func doFixMIsNullToFalseL(rootNode ast.Node, in ast.Node) ([]byte, error) {
	switch in.(type) {
	case *ast.IsNullExpr:
		expr := in.(*ast.IsNullExpr)

		falseExpr := &test_driver.ValueExpr{
			Datum: test_driver.NewDatum(0),
		}
		parenExpr := &ast.ParenthesesExpr{
			Expr: falseExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMIsNullToFalseL]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)

		return sql, nil
	case nil:
		return nil, errors.New("[FixMIsNullToFalseL]type nil")
	default:
		return nil, errors.New("[FixMIsNullToFalseL]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMIsNotNullToTrueU: x IS NOT NULL → TRUE
func doFixMIsNotNullToTrueU(rootNode ast.Node, in ast.Node) ([]byte, error) {
	switch in.(type) {
	case *ast.IsNullExpr:
		expr := in.(*ast.IsNullExpr)

		trueExpr := &test_driver.ValueExpr{
			Datum: test_driver.NewDatum(1),
		}
		parenExpr := &ast.ParenthesesExpr{
			Expr: trueExpr,
		}

		replaceExprInRoot(rootNode, expr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMIsNotNullToTrueU]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, expr)

		return sql, nil
	case nil:
		return nil, errors.New("[FixMIsNotNullToTrueU]type nil")
	default:
		return nil, errors.New("[FixMIsNotNullToTrueU]type default " + reflect.TypeOf(in).String())
	}
}
