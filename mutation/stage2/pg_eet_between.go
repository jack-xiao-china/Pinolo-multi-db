package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL EET BETWEEN -> Comparison transformation mutation

// addFixMBetweenToCmp_Pg: FixMBetweenToCmp_Pg, A_Expr(BETWEEN): x BETWEEN a AND b -> (x >= a) AND (x <= b)
// Semantically equivalent. If result sets differ -> bug detected.
func (v *PgMutateVisitor) addFixMBetweenToCmp_Pg(expr *pgquery.A_Expr, flag int) {
	if expr != nil && (expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN_SYM ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM) {
		v.addPgCandidate(FixMBetweenToCmp_Pg, 1, nil, flag)
	}
}

// addFixMBetweenDropUpperU_Pg: FixMBetweenDropUpperU_Pg, x BETWEEN a AND b -> x >= a (upper: drop upper bound)
// Implication (upper): BETWEEN result ⊆ >= result
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x < a
func (v *PgMutateVisitor) addFixMBetweenDropUpperU_Pg(expr *pgquery.A_Expr, flag int) {
	if expr != nil && (expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN_SYM ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM) &&
		expr.Lexpr != nil && expr.Rexpr != nil {
		v.addPgCandidate(FixMBetweenDropUpperU_Pg, 1, nil, flag)
	}
}

// addFixMBetweenDropLowerU_Pg: FixMBetweenDropLowerU_Pg, x BETWEEN a AND b -> x <= b (upper: drop lower bound)
// Implication (upper): BETWEEN result ⊆ <= result
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x > b
func (v *PgMutateVisitor) addFixMBetweenDropLowerU_Pg(expr *pgquery.A_Expr, flag int) {
	if expr != nil && (expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN_SYM ||
		expr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM) &&
		expr.Lexpr != nil && expr.Rexpr != nil {
		v.addPgCandidate(FixMBetweenDropLowerU_Pg, 1, nil, flag)
	}
}

// doFixMBetweenToCmp_Pg: FixMBetweenToCmp_Pg, x BETWEEN a AND b -> (x >= a) AND (x <= b)
// For NOT BETWEEN: NOT((x >= a) AND (x <= b))
// For BETWEEN SYMMETRIC: same transformation (symmetric BETWEEN evaluates the same as BETWEEN for this purpose)
func doFixMBetweenToCmp_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMBetweenToCmp_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find a BETWEEN A_Expr in the WHERE clause
		betweenExpr := findBetweenExprInWhere(sel.WhereClause)
		if betweenExpr == nil {
			continue
		}

		// Save original values
		oldWhere := sel.WhereClause
		originalKind := betweenExpr.Kind
		originalLexpr := betweenExpr.Lexpr
		originalRexpr := betweenExpr.Rexpr
		originalName := betweenExpr.Name

		// Build (x >= a) AND (x <= b)
		// In pg_query BETWEEN A_Expr:
		//   Lexpr = x (the expression being tested)
		//   Rexpr = List containing [a, b] (the bounds)
		//   For BETWEEN SYMMETRIC: same structure but Kind = AEXPR_BETWEEN_SYM

		x := betweenExpr.Lexpr

		// Extract bounds from Rexpr (it's a List with 2 items)
		bounds := betweenExpr.Rexpr.GetList()
		if bounds == nil || len(bounds.Items) < 2 {
			return "", errors.New("[doFixMBetweenToCmp_Pg]expected 2 bounds in BETWEEN Rexpr")
		}
		a := bounds.Items[0]
		b := bounds.Items[1]

		// x >= a
		geExpr := pgquery.MakeAExprNode(
			pgquery.A_Expr_Kind_AEXPR_OP,
			[]*pgquery.Node{pgquery.MakeStrNode(">=")},
			x,
			a,
			0,
		)

		// x <= b
		leExpr := pgquery.MakeAExprNode(
			pgquery.A_Expr_Kind_AEXPR_OP,
			[]*pgquery.Node{pgquery.MakeStrNode("<=")},
			x,
			b,
			0,
		)

		// (x >= a) AND (x <= b)
		andExpr := pgquery.MakeBoolExprNode(pgquery.BoolExprType_AND_EXPR, []*pgquery.Node{
			geExpr,
			leExpr,
		}, 0)

		// For NOT BETWEEN, wrap with NOT
		resultExpr := andExpr
		isNotBetween := betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
			betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM
		if isNotBetween {
			resultExpr = pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{
				andExpr,
			}, 0)
		}

		sel.WhereClause = resultExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMBetweenToCmp_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		betweenExpr.Kind = originalKind
		betweenExpr.Lexpr = originalLexpr
		betweenExpr.Rexpr = originalRexpr
		betweenExpr.Name = originalName

		return sql, nil
	}

	return "", errors.New("[doFixMBetweenToCmp_Pg]no BETWEEN expression found")
}

// findBetweenExprInWhere: recursively search for a BETWEEN A_Expr in a WHERE clause node
func findBetweenExprInWhere(node *pgquery.Node) *pgquery.A_Expr {
	if node == nil {
		return nil
	}

	// Check if this node itself is a BETWEEN A_Expr
	aExpr := node.GetAExpr()
	if aExpr != nil && (aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN ||
		aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
		aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_BETWEEN_SYM ||
		aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM) {
		return aExpr
	}

	// Recursively search in BoolExpr args
	boolExpr := node.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			result := findBetweenExprInWhere(arg)
			if result != nil {
				return result
			}
		}
	}

	// Search in Parenthesized expressions (not directly available in pg_query,
	// but check for nested nodes)
	return nil
}

// doFixMBetweenDropUpperU_Pg: FixMBetweenDropUpperU_Pg, x BETWEEN a AND b -> x >= a (upper: drop upper bound)
// Containment: x BETWEEN a AND b ⊆ x >= a
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x < a
func doFixMBetweenDropUpperU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMBetweenDropUpperU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		betweenExpr := findBetweenExprInWhere(sel.WhereClause)
		if betweenExpr == nil {
			continue
		}

		oldWhere := sel.WhereClause
		originalKind := betweenExpr.Kind
		originalLexpr := betweenExpr.Lexpr
		originalRexpr := betweenExpr.Rexpr
		originalName := betweenExpr.Name

		x := betweenExpr.Lexpr
		bounds := betweenExpr.Rexpr.GetList()
		if bounds == nil || len(bounds.Items) < 1 {
			return "", errors.New("[doFixMBetweenDropUpperU_Pg]expected bounds in BETWEEN Rexpr")
		}
		a := bounds.Items[0] // lower bound

		isNotBetween := betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
			betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM

		var resultExpr *pgquery.Node
		if !isNotBetween {
			// BETWEEN -> x >= a (drop upper bound)
			resultExpr = pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode(">=")},
				x,
				a,
				0,
			)
		} else {
			// NOT BETWEEN -> x < a (NOT BETWEEN ⊆ < lower bound)
			resultExpr = pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("<")},
				x,
				a,
				0,
			)
		}

		sel.WhereClause = resultExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMBetweenDropUpperU_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		betweenExpr.Kind = originalKind
		betweenExpr.Lexpr = originalLexpr
		betweenExpr.Rexpr = originalRexpr
		betweenExpr.Name = originalName

		return sql, nil
	}

	return "", errors.New("[doFixMBetweenDropUpperU_Pg]no BETWEEN expression found")
}

// doFixMBetweenDropLowerU_Pg: FixMBetweenDropLowerU_Pg, x BETWEEN a AND b -> x <= b (upper: drop lower bound)
// Containment: x BETWEEN a AND b ⊆ x <= b
// For NOT BETWEEN: x NOT BETWEEN a AND b -> x > b
func doFixMBetweenDropLowerU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMBetweenDropLowerU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		betweenExpr := findBetweenExprInWhere(sel.WhereClause)
		if betweenExpr == nil {
			continue
		}

		oldWhere := sel.WhereClause
		originalKind := betweenExpr.Kind
		originalLexpr := betweenExpr.Lexpr
		originalRexpr := betweenExpr.Rexpr
		originalName := betweenExpr.Name

		x := betweenExpr.Lexpr
		bounds := betweenExpr.Rexpr.GetList()
		if bounds == nil || len(bounds.Items) < 2 {
			return "", errors.New("[doFixMBetweenDropLowerU_Pg]expected 2 bounds in BETWEEN Rexpr")
		}
		b := bounds.Items[1] // upper bound

		isNotBetween := betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN ||
			betweenExpr.Kind == pgquery.A_Expr_Kind_AEXPR_NOT_BETWEEN_SYM

		var resultExpr *pgquery.Node
		if !isNotBetween {
			// BETWEEN -> x <= b (drop lower bound)
			resultExpr = pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode("<=")},
				x,
				b,
				0,
			)
		} else {
			// NOT BETWEEN -> x > b (NOT BETWEEN ⊆ > upper bound)
			resultExpr = pgquery.MakeAExprNode(
				pgquery.A_Expr_Kind_AEXPR_OP,
				[]*pgquery.Node{pgquery.MakeStrNode(">")},
				x,
				b,
				0,
			)
		}

		sel.WhereClause = resultExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMBetweenDropLowerU_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		betweenExpr.Kind = originalKind
		betweenExpr.Lexpr = originalLexpr
		betweenExpr.Rexpr = originalRexpr
		betweenExpr.Name = originalName

		return sql, nil
	}

	return "", errors.New("[doFixMBetweenDropLowerU_Pg]no BETWEEN expression found")
}