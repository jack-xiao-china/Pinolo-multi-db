package stage2

import (
	"github.com/pkg/errors"
	"github.com/pingcap/tidb/parser/ast"
	"github.com/pingcap/tidb/parser/opcode"
	"github.com/pingcap/tidb/parser/test_driver"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"math/rand"
	"reflect"
)

// EET (Equivalent Expression Testing) transformation mutations
// Inspired by SQLancer's EET Oracle transformation rules

// addFixMAndTrueU: FixMAndTrueU, *ast.SelectStmt: WHERE E -> WHERE (p OR NOT p OR p IS NULL) AND E
// The left side is always TRUE (three-valued logic), so E's result set is contained in mutated result set.
func (v *MutateVisitor) addFixMAndTrueU(in *ast.SelectStmt, flag int) {
	if in.Where != nil {
		v.addCandidate(FixMAndTrueU, 1, in, flag)
	}
}

// doFixMAndTrueU: FixMAndTrueU, *ast.SelectStmt: WHERE E -> WHERE (p OR NOT p OR p IS NULL) AND E
func doFixMAndTrueU(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == nil {
			return nil, errors.New("[FixMAndTrueU]sel.Where == nil")
		}
		old := sel.Where

		// Build tautology: (p OR NOT p OR p IS NULL)
		// Use a simple column reference as p if available, otherwise use a constant
		pExpr := findPredicateForEET(sel, seed)
		tautology := buildTautologyExpr(pExpr)

		// WHERE E -> WHERE (tautology) AND E
		sel.Where = &ast.BinaryOperationExpr{
			Op: opcode.LogicAnd,
			L:  tautology,
			R:  old,
		}

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMAndTrueU]restore error")
		}
		sel.Where = old
		return sql, nil
	case nil:
		return nil, errors.New("[FixMAndTrueU]type nil")
	default:
		return nil, errors.New("[FixMAndTrueU]type default " + reflect.TypeOf(in).String())
	}
}

// addFixMOrFalseL: FixMOrFalseL, *ast.SelectStmt: WHERE E -> WHERE (p AND NOT p AND p IS NOT NULL) OR E
func (v *MutateVisitor) addFixMOrFalseL(in *ast.SelectStmt, flag int) {
	if in.Where != nil {
		v.addCandidate(FixMOrFalseL, 0, in, flag)
	}
}

// doFixMOrFalseL: FixMOrFalseL, *ast.SelectStmt: WHERE E -> WHERE (p AND NOT p AND p IS NOT NULL) OR E
func doFixMOrFalseL(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == nil {
			return nil, errors.New("[FixMOrFalseL]sel.Where == nil")
		}
		old := sel.Where

		// Build contradiction: (p AND NOT p AND p IS NOT NULL)
		pExpr := findPredicateForEET(sel, seed)
		contradiction := buildContradictionExpr(pExpr)

		// WHERE E -> WHERE (contradiction) OR E
		sel.Where = &ast.BinaryOperationExpr{
			Op: opcode.LogicOr,
			L:  contradiction,
			R:  old,
		}

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMOrFalseL]restore error")
		}
		sel.Where = old
		return sql, nil
	case nil:
		return nil, errors.New("[FixMOrFalseL]type nil")
	default:
		return nil, errors.New("[FixMOrFalseL]type default " + reflect.TypeOf(in).String())
	}
}

// addFixMCaseTrueU: FixMCaseTrueU, *ast.SelectStmt: WHERE E -> WHERE CASE WHEN TRUE THEN E ELSE rand END
func (v *MutateVisitor) addFixMCaseTrueU(in *ast.SelectStmt, flag int) {
	if in.Where != nil {
		v.addCandidate(FixMCaseTrueU, 1, in, flag)
	}
}

// doFixMCaseTrueU: FixMCaseTrueU, WHERE E -> CASE WHEN TRUE THEN E ELSE rand END
func doFixMCaseTrueU(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == nil {
			return nil, errors.New("[FixMCaseTrueU]sel.Where == nil")
		}
		old := sel.Where

		// Build CASE WHEN TRUE THEN E ELSE rand END
		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &test_driver.ValueExpr{Datum: test_driver.NewDatum(1)}, // TRUE
					Result: old,
				},
			},
			ElseClause: buildRandomBoolExpr(seed),
		}

		sel.Where = caseExpr

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMCaseTrueU]restore error")
		}
		sel.Where = old
		return sql, nil
	case nil:
		return nil, errors.New("[FixMCaseTrueU]type nil")
	default:
		return nil, errors.New("[FixMCaseTrueU]type default " + reflect.TypeOf(in).String())
	}
}

// addFixMCaseFalseL: FixMCaseFalseL, *ast.SelectStmt: WHERE E -> WHERE CASE WHEN FALSE THEN rand ELSE E END
func (v *MutateVisitor) addFixMCaseFalseL(in *ast.SelectStmt, flag int) {
	if in.Where != nil {
		v.addCandidate(FixMCaseFalseL, 0, in, flag)
	}
}

// doFixMCaseFalseL: FixMCaseFalseL, WHERE E -> CASE WHEN FALSE THEN rand ELSE E END
func doFixMCaseFalseL(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == nil {
			return nil, errors.New("[FixMCaseFalseL]sel.Where == nil")
		}
		old := sel.Where

		// Build CASE WHEN FALSE THEN rand ELSE E END
		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &test_driver.ValueExpr{Datum: test_driver.NewDatum(0)}, // FALSE
					Result: buildRandomBoolExpr(seed),
				},
			},
			ElseClause: old,
		}

		sel.Where = caseExpr

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMCaseFalseL]restore error")
		}
		sel.Where = old
		return sql, nil
	case nil:
		return nil, errors.New("[FixMCaseFalseL]type nil")
	default:
		return nil, errors.New("[FixMCaseFalseL]type default " + reflect.TypeOf(in).String())
	}
}

// addFixMCaseRandEq: FixMCaseRandEq, *ast.SelectStmt: WHERE E -> WHERE CASE WHEN rand THEN E ELSE E END
func (v *MutateVisitor) addFixMCaseRandEq(in *ast.SelectStmt, flag int) {
	if in.Where != nil {
		v.addCandidate(FixMCaseRandEq, 1, in, flag) // Using 1 as default (equivalence mutation)
	}
}

// doFixMCaseRandEq: FixMCaseRandEq, WHERE E -> CASE WHEN rand THEN E ELSE E END
func doFixMCaseRandEq(rootNode ast.Node, in ast.Node, seed int64) ([]byte, error) {
	switch in.(type) {
	case *ast.SelectStmt:
		sel := in.(*ast.SelectStmt)
		if sel.Where == nil {
			return nil, errors.New("[FixMCaseRandEq]sel.Where == nil")
		}
		old := sel.Where

		// Build CASE WHEN rand THEN E ELSE E END
		r := rand.New(rand.NewSource(seed))
		randBool := r.Intn(2) // 0 or 1

		caseExpr := &ast.CaseExpr{
			WhenClauses: []*ast.WhenClause{
				{
					Expr: &test_driver.ValueExpr{Datum: test_driver.NewDatum(randBool)},
					Result: old,
				},
			},
			ElseClause: old,
		}

		sel.Where = caseExpr

		sql, err := restore(rootNode)
		if err != nil {
			return nil, errors.Wrap(err, "[FixMCaseRandEq]restore error")
		}
		sel.Where = old
		return sql, nil
	case nil:
		return nil, errors.New("[FixMCaseRandEq]type nil")
	default:
		return nil, errors.New("[FixMCaseRandEq]type default " + reflect.TypeOf(in).String())
	}
}

// HAVING clause EET mutations

// addFixMAndTrueUHaving: FixMAndTrueU for HAVING clause
func (v *MutateVisitor) addFixMAndTrueUHaving(in *ast.SelectStmt, flag int) {
	if in.Having != nil && in.Having.Expr != nil {
		v.addCandidate(FixMAndTrueU + "Having", 1, in, flag)
	}
}

// addFixMOrFalseLHaving: FixMOrFalseL for HAVING clause
func (v *MutateVisitor) addFixMOrFalseLHaving(in *ast.SelectStmt, flag int) {
	if in.Having != nil && in.Having.Expr != nil {
		v.addCandidate(FixMOrFalseL + "Having", 0, in, flag)
	}
}

// Helper functions for EET transformations

// findPredicateForEET: find a suitable predicate p for tautology/contradiction construction
func findPredicateForEET(sel *ast.SelectStmt, seed int64) ast.ExprNode {
	// Try to find a column reference from FROM clause
	// For simplicity, use a constant predicate: 1 = 1
	r := rand.New(rand.NewSource(seed))
	val := r.Intn(100)

	// Use simple comparison: val > 0
	return &ast.BinaryOperationExpr{
		Op: opcode.GT,
		L:  &test_driver.ValueExpr{Datum: test_driver.NewDatum(val)},
		R:  &test_driver.ValueExpr{Datum: test_driver.NewDatum(0)},
	}
}

// buildTautologyExpr: build (p OR NOT p OR p IS NULL) - always TRUE in three-valued logic
func buildTautologyExpr(p ast.ExprNode) ast.ExprNode {
	// NOT p
	notP := &ast.UnaryOperationExpr{
		Op: opcode.Not,
		V:  p,
	}

	// p OR NOT p
	pOrNotP := &ast.BinaryOperationExpr{
		Op: opcode.LogicOr,
		L:  p,
		R:  notP,
	}

	// p IS NULL
	pIsNull := &ast.IsNullExpr{
		Expr: p,
		Not:  false,
	}

	// (p OR NOT p) OR p IS NULL
	return &ast.BinaryOperationExpr{
		Op: opcode.LogicOr,
		L:  pOrNotP,
		R:  pIsNull,
	}
}

// buildContradictionExpr: build (p AND NOT p AND p IS NOT NULL) - always FALSE
func buildContradictionExpr(p ast.ExprNode) ast.ExprNode {
	// NOT p
	notP := &ast.UnaryOperationExpr{
		Op: opcode.Not,
		V:  p,
	}

	// p AND NOT p
	pAndNotP := &ast.BinaryOperationExpr{
		Op: opcode.LogicAnd,
		L:  p,
		R:  notP,
	}

	// p IS NOT NULL
	pIsNotNull := &ast.IsNullExpr{
		Expr: p,
		Not:  true,
	}

	// (p AND NOT p) AND p IS NOT NULL
	return &ast.BinaryOperationExpr{
		Op: opcode.LogicAnd,
		L:  pAndNotP,
		R:  pIsNotNull,
	}
}

// buildRandomBoolExpr: build a random boolean expression (TRUE or FALSE)
func buildRandomBoolExpr(seed int64) ast.ExprNode {
	r := rand.New(rand.NewSource(seed))
	if r.Intn(2) == 1 {
		return &test_driver.ValueExpr{Datum: test_driver.NewDatum(1)}
	}
	return &test_driver.ValueExpr{Datum: test_driver.NewDatum(0)}
}

// getEETMutationNames: return all EET mutation names (including semantic rewrite rules)
func getEETMutationNames() []string {
	return []string{
		FixMAndTrueU,
		FixMOrFalseL,
		FixMCaseTrueU,
		FixMCaseFalseL,
		FixMCaseRandEq,
		FixMDeMorganAnd,
		FixMDeMorganOr,
		FixMBetweenToCmp,
		FixMCoalesceToCase,
		FixMNullifToCase,
			FixMExistsToIn,
			FixMInToExists,
	}
}