package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL EET De Morgan's Law transformation mutations

// addFixMDeMorganAnd_Pg: FixMDeMorganAnd_Pg, BoolExpr(AND): (A AND B) -> NOT(NOT(A) OR NOT(B))
// Semantically equivalent. If result sets differ -> bug detected.
func (v *PgMutateVisitor) addFixMDeMorganAnd_Pg(expr *pgquery.BoolExpr, flag int) {
	if expr != nil && expr.Boolop == pgquery.BoolExprType_AND_EXPR && len(expr.Args) == 2 {
		v.addPgCandidate(FixMDeMorganAnd_Pg, 1, nil, flag)
	}
}

// addFixMDeMorganOr_Pg: FixMDeMorganOr_Pg, BoolExpr(OR): (A OR B) -> NOT(NOT(A) AND NOT(B))
// Semantically equivalent. If result sets differ -> bug detected.
func (v *PgMutateVisitor) addFixMDeMorganOr_Pg(expr *pgquery.BoolExpr, flag int) {
	if expr != nil && expr.Boolop == pgquery.BoolExprType_OR_EXPR && len(expr.Args) == 2 {
		v.addPgCandidate(FixMDeMorganOr_Pg, 1, nil, flag)
	}
}

// doFixMDeMorganAnd_Pg: FixMDeMorganAnd_Pg, (A AND B) -> NOT(NOT(A) OR NOT(B))
// Traverses the AST, finds the first BoolExpr(AND) with exactly 2 args, and applies De Morgan transformation.
func doFixMDeMorganAnd_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMDeMorganAnd_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	found, oldWhere, originalBoolop, originalArgs := findAndReplaceBoolExpr(rootNode, pgquery.BoolExprType_AND_EXPR, pgquery.BoolExprType_OR_EXPR)
	if !found {
		return "", errors.New("[doFixMDeMorganAnd_Pg]no AND BoolExpr with 2 args found")
	}

	sql, err := pgquery.Deparse(rootNode)
	if err != nil {
		// Restore
		restoreBoolExpr(rootNode, oldWhere, originalBoolop, originalArgs)
		return "", errors.Wrap(err, "[doFixMDeMorganAnd_Pg]deparse error")
	}

	// Restore original AST
	restoreBoolExpr(rootNode, oldWhere, originalBoolop, originalArgs)
	return sql, nil
}

// doFixMDeMorganOr_Pg: FixMDeMorganOr_Pg, (A OR B) -> NOT(NOT(A) AND NOT(B))
func doFixMDeMorganOr_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMDeMorganOr_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	found, oldWhere, originalBoolop, originalArgs := findAndReplaceBoolExpr(rootNode, pgquery.BoolExprType_OR_EXPR, pgquery.BoolExprType_AND_EXPR)
	if !found {
		return "", errors.New("[doFixMDeMorganOr_Pg]no OR BoolExpr with 2 args found")
	}

	sql, err := pgquery.Deparse(rootNode)
	if err != nil {
		restoreBoolExpr(rootNode, oldWhere, originalBoolop, originalArgs)
		return "", errors.Wrap(err, "[doFixMDeMorganOr_Pg]deparse error")
	}

	restoreBoolExpr(rootNode, oldWhere, originalBoolop, originalArgs)
	return sql, nil
}

// findAndReplaceBoolExpr: find the first BoolExpr with targetBoolop and exactly 2 args,
// then apply De Morgan transformation:
//   AND → OR: (A AND B) → NOT(NOT(A) OR NOT(B))
//   OR → AND: (A OR B) → NOT(NOT(A) AND NOT(B))
// Returns (found, oldWhereClause, originalBoolop, originalArgs) for restoration.
func findAndReplaceBoolExpr(rootNode *pgquery.ParseResult, targetBoolop pgquery.BoolExprType, resultBoolop pgquery.BoolExprType) (bool, *pgquery.Node, pgquery.BoolExprType, []*pgquery.Node) {
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Search for a BoolExpr in the WHERE clause
		boolExpr := sel.WhereClause.GetBoolExpr()
		if boolExpr != nil && boolExpr.Boolop == targetBoolop && len(boolExpr.Args) == 2 {
			// Save original values for restoration
			oldWhere := sel.WhereClause
			originalBoolop := boolExpr.Boolop
			originalArgs := boolExpr.Args

			// Apply De Morgan transformation:
			// (A AND B) → NOT(NOT(A) OR NOT(B))
			// (A OR B) → NOT(NOT(A) AND NOT(B))
			a := boolExpr.Args[0]
			b := boolExpr.Args[1]

			// NOT(A)
			notA := pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{a}, 0)
			// NOT(B)
			notB := pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{b}, 0)

			// NOT(A) <resultBoolop> NOT(B)
			innerExpr := pgquery.MakeBoolExprNode(resultBoolop, []*pgquery.Node{notA, notB}, 0)

			// NOT(NOT(A) <resultBoolop> NOT(B))
			demorganResult := pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{innerExpr}, 0)

			sel.WhereClause = demorganResult
			return true, oldWhere, originalBoolop, originalArgs
		}
	}
	return false, nil, 0, nil
}

// restoreBoolExpr: restore the original BoolExpr after mutation
func restoreBoolExpr(rootNode *pgquery.ParseResult, oldWhere *pgquery.Node, originalBoolop pgquery.BoolExprType, originalArgs []*pgquery.Node) {
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		// Check if we modified this SelectStmt's WhereClause
		// (the demorgan result is a NOT BoolExpr wrapping another BoolExpr)
		currentWhere := sel.WhereClause
		notExpr := currentWhere.GetBoolExpr()
		if notExpr != nil && notExpr.Boolop == pgquery.BoolExprType_NOT_EXPR && len(notExpr.Args) == 1 {
			// This looks like our De Morgan result
			// Restore: replace with the original saved BoolExpr
			boolExpr := oldWhere.GetBoolExpr()
			if boolExpr != nil {
				boolExpr.Boolop = originalBoolop
				boolExpr.Args = originalArgs
				sel.WhereClause = oldWhere
			}
			return
		}
	}
}