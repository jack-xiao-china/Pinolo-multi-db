package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"reflect"
)

// EET EXISTS <-> IN transformation mutations (MySQL)

// EXISTS -> IN transformation:
// EXISTS(subquery) -> 1 IN (subquery with SELECT 1 instead of SELECT *)
// This is semantically equivalent because:
// - EXISTS returns TRUE if subquery has any rows
// - 1 IN (SELECT 1 FROM ...) returns TRUE if subquery has any rows with value 1
// - However, IN returns NULL if the subquery contains NULL values
// - We wrap with CASE WHEN ... IS NOT NULL to handle NULL safety
//
// Simplified: EXISTS(subq) -> CASE WHEN (1 IN (subq_select_1)) IS NOT NULL
//                               THEN (1 IN (subq_select_1)) ELSE FALSE END

// IN -> EXISTS transformation:
// lhs IN (SELECT col FROM t WHERE pred) -> EXISTS(SELECT 1 FROM t WHERE t.col = lhs AND pred)
// With NULL safety: CASE WHEN (lhs IN subq) IS NOT NULL THEN EXISTS(subq_with_equality) ELSE FALSE END

// addFixMExistsToIn: FixMExistsToIn, *ast.ExistsSubqueryExpr: EXISTS/NOT EXISTS(subq) -> IN equivalent
// NOTE: Also handles NOT EXISTS cases - NOT EXISTS -> NOT (1 IN subq) with NULL-safe wrapper
func (v *MutateVisitor) addFixMExistsToIn(in *ast.ExistsSubqueryExpr, flag int) {
	if in != nil && in.Sel != nil {
		v.addCandidate(FixMExistsToIn, 1, in, flag)
	}
}

// addFixMInToExists: FixMInToExists, *ast.PatternInExpr: lhs IN (subq) -> EXISTS equivalent
func (v *MutateVisitor) addFixMInToExists(in *ast.PatternInExpr, flag int) {
	if in != nil && !in.Not && in.Sel != nil && in.Expr != nil {
		v.addCandidate(FixMInToExists, 1, in, flag)
	}
}

// doFixMExistsToIn: FixMExistsToIn, EXISTS(subq) -> CASE WHEN (1 IN subq_select_1) IS NOT NULL THEN (1 IN subq_select_1) ELSE FALSE END
// For NOT EXISTS: NOT EXISTS(subq) -> CASE WHEN (1 NOT IN subq_select_1) IS NOT NULL THEN (1 NOT IN subq_select_1) ELSE TRUE END
func doFixMExistsToIn(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.ExistsSubqueryExpr:
		existExpr := in.(*ast.ExistsSubqueryExpr)
		if existExpr.Sel == nil {
			return nil, errors.New("[FixMExistsToIn]existExpr.Sel == nil")
		}

		// Get the subquery
		subq := existExpr.Sel.(*ast.SubqueryExpr)
		if subq == nil || subq.Query == nil {
			return nil, errors.New("[FixMExistsToIn]subquery is nil")
		}

		// Build: 1 IN (SELECT 1 FROM ... WHERE ...)
		// We create a PatternInExpr with the same subquery
		inExpr := &ast.PatternInExpr{
			Expr: &test_driver.ValueExpr{Datum: test_driver.NewDatum(1)}, // lhs = 1
			Sel:  subq, // same subquery
			Not:  existExpr.Not, // preserve NOT if present
		}

		// Build NULL-safe CASE wrapping:
		// CASE WHEN (1 IN subq) IS NOT NULL THEN (1 IN subq) ELSE FALSE END
		// For NOT EXISTS: CASE WHEN (1 NOT IN subq) IS NOT NULL THEN (1 NOT IN subq) ELSE TRUE END
		elseValue := &test_driver.ValueExpr{Datum: test_driver.NewDatum(0)} // FALSE
		if existExpr.Not {
			elseValue = &test_driver.ValueExpr{Datum: test_driver.NewDatum(1)} // TRUE for NOT EXISTS
		}

		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &ast.IsNullExpr{
						Expr: inExpr,
						Not:  true, // IS NOT NULL
					},
					Result: inExpr, // THEN (1 IN subq)
				},
			},
			ElseClause: elseValue, // ELSE FALSE (or TRUE for NOT EXISTS)
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: caseExpr,
		}

		replaceExprInRoot(rootNode, existExpr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMExistsToIn]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, existExpr)
		return sql, nil
	case nil:
		return nil, errors.New("[FixMExistsToIn]type nil")
	default:
		return nil, errors.New("[FixMExistsToIn]type default " + reflect.TypeOf(in).String())
	}
}

// doFixMInToExists: FixMInToExists, lhs IN (subq) -> CASE WHEN (lhs IN subq) IS NOT NULL THEN EXISTS(equivalent_subq) ELSE FALSE END
// For NOT IN: lhs NOT IN (subq) -> CASE WHEN (lhs NOT IN subq) IS NOT NULL THEN NOT EXISTS(equivalent_subq) ELSE TRUE END
func doFixMInToExists(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.PatternInExpr:
		inExpr := in.(*ast.PatternInExpr)
		if inExpr.Sel == nil || inExpr.Expr == nil {
			return nil, errors.New("[FixMInToExists]inExpr.Sel == nil || inExpr.Expr == nil")
		}

		subq := inExpr.Sel.(*ast.SubqueryExpr)
		if subq == nil || subq.Query == nil {
			return nil, errors.New("[FixMInToExists]subquery is nil")
		}

		// Build EXISTS equivalent:
		// EXISTS(SELECT 1 FROM ... WHERE ...)
		// We reuse the same subquery for EXISTS
		existExpr := &ast.ExistsSubqueryExpr{
			Sel:  subq,
			Not:  inExpr.Not, // preserve NOT if present
		}

		// Build NULL-safe CASE wrapping:
		// CASE WHEN (lhs IN subq) IS NOT NULL THEN EXISTS(subq) ELSE FALSE END
		elseValue := &test_driver.ValueExpr{Datum: test_driver.NewDatum(0)} // FALSE
		if inExpr.Not {
			elseValue = &test_driver.ValueExpr{Datum: test_driver.NewDatum(1)} // TRUE for NOT IN
		}

		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &ast.IsNullExpr{
						Expr: inExpr, // original IN expression
						Not:  true,   // IS NOT NULL
					},
					Result: existExpr, // THEN EXISTS(subq)
				},
			},
			ElseClause: elseValue, // ELSE FALSE (or TRUE for NOT IN)
		}

		parenExpr := &ast.ParenthesesExpr{
			Expr: caseExpr,
		}

		replaceExprInRoot(rootNode, inExpr, parenExpr)

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMInToExists]restore error")
		}

		// Restore original
		replaceExprInRoot(rootNode, parenExpr, inExpr)
		return sql, nil
	case nil:
		return nil, errors.New("[FixMInToExists]type nil")
	default:
		return nil, errors.New("[FixMInToExists]type default " + reflect.TypeOf(in).String())
	}
}