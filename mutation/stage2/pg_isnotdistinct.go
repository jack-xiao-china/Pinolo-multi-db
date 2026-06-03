package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL IS NOT DISTINCT FROM -> = implication mutation

// a IS NOT DISTINCT FROM b -> a = b
// Implication (lower): = result ⊆ IS NOT DISTINCT FROM result
// a = b TRUE ⊆ a IS NOT DISTINCT FROM b TRUE
// (= returns NULL for NULL inputs, IS NOT DISTINCT FROM returns TRUE for NULL=NULL)
// Lower mutation: expected =result ⊆ IS NOT DISTINCT FROM result
// If containment violated -> bug detected.

// addFixMIsNotDistinctFromToLowerL_Pg: FixMIsNotDistinctFromToLowerL_Pg, A_Expr(IS NOT DISTINCT FROM): a IS NOT DISTINCT FROM b -> a = b
func (v *PgMutateVisitor) addFixMIsNotDistinctFromToLowerL_Pg(expr *pgquery.A_Expr, flag int) {
	if expr != nil && expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_DISTINCT &&
		expr.Lexpr != nil && expr.Rexpr != nil {
		v.addPgCandidate(FixMIsNotDistinctFromToLowerL_Pg, 1, nil, flag)
	}
}

// doFixMIsNotDistinctFromToLowerL_Pg: FixMIsNotDistinctFromToLowerL_Pg, a IS NOT DISTINCT FROM b -> a = b
// Change A_Expr Kind from AEXPR_NOT_DISTINCT to AEXPR_OP, keep operator "="
func doFixMIsNotDistinctFromToLowerL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMIsNotDistinctFromToLowerL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find AEXPR_NOT_DISTINCT A_Expr in WHERE clause
		notDistinctExpr := findNotDistinctExprInWhere(sel.WhereClause)
		if notDistinctExpr == nil {
			continue
		}

		oldWhere := sel.WhereClause
		originalKind := notDistinctExpr.Kind
		originalLexpr := notDistinctExpr.Lexpr
		originalRexpr := notDistinctExpr.Rexpr
		originalName := notDistinctExpr.Name

		x := notDistinctExpr.Lexpr
		b := notDistinctExpr.Rexpr

		// Build: a = b (regular equality comparison)
		eqExpr := pgquery.MakeAExprNode(
			pgquery.A_Expr_Kind_AEXPR_OP,
			[]*pgquery.Node{pgquery.MakeStrNode("=")},
			x,
			b,
			0,
		)

		sel.WhereClause = eqExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMIsNotDistinctFromToLowerL_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		notDistinctExpr.Kind = originalKind
		notDistinctExpr.Lexpr = originalLexpr
		notDistinctExpr.Rexpr = originalRexpr
		notDistinctExpr.Name = originalName

		return sql, nil
	}

	return "", errors.New("[doFixMIsNotDistinctFromToLowerL_Pg]no IS NOT DISTINCT FROM expression found")
}

// findNotDistinctExprInWhere: recursively search for an AEXPR_NOT_DISTINCT A_Expr in WHERE clause
func findNotDistinctExprInWhere(node *pgquery.Node) *pgquery.A_Expr {
	if node == nil {
		return nil
	}

	// Check if this node itself is a NOT DISTINCT A_Expr
	aExpr := node.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_DISTINCT {
		return aExpr
	}

	// Search in BoolExpr args
	boolExpr := node.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			result := findNotDistinctExprInWhere(arg)
			if result != nil {
				return result
			}
		}
	}

	return nil
}