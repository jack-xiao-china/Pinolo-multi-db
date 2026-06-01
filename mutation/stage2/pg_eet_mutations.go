package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"math/rand"
)

// PostgreSQL EET (Equivalent Expression Testing) transformation mutations
// Inspired by SQLancer's EET Oracle transformation rules

// ------------------------------------------------
// EET Rule 1: Tautology wrapping
// E → (p OR NOT p OR p IS NULL) AND E
// ------------------------------------------------

// doFixMAndTrueU_Pg: WHERE E → WHERE (p OR NOT p OR p IS NULL) AND E
func doFixMAndTrueU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMAndTrueU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Build tautology: (p OR NOT p OR p IS NULL)
		pExpr := buildPgPredicate(seed)
		tautology := buildPgTautology(pExpr, rander)

		// WHERE E → WHERE tautology AND E
		oldWhere := sel.WhereClause
		sel.WhereClause = pgquery.MakeBoolExprNode(pgquery.BoolExprType_AND_EXPR, []*pgquery.Node{
			tautology,
			oldWhere,
		}, 0)

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMAndTrueU_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMAndTrueU_Pg]no WHERE clause found")
}

// ------------------------------------------------
// EET Rule 2: Contradiction wrapping
// E → (p AND NOT p AND p IS NOT NULL) OR E
// ------------------------------------------------

// doFixMOrFalseL_Pg: WHERE E → WHERE (p AND NOT p AND p IS NOT NULL) OR E
func doFixMOrFalseL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMOrFalseL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Build contradiction: (p AND NOT p AND p IS NOT NULL)
		pExpr := buildPgPredicate(seed)
		contradiction := buildPgContradiction(pExpr, rander)

		// WHERE E → WHERE contradiction OR E
		oldWhere := sel.WhereClause
		sel.WhereClause = pgquery.MakeBoolExprNode(pgquery.BoolExprType_OR_EXPR, []*pgquery.Node{
			contradiction,
			oldWhere,
		}, 0)

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMOrFalseL_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMOrFalseL_Pg]no WHERE clause found")
}

// ------------------------------------------------
// EET Rule 3: CASE WHEN TRUE wrapping
// E → CASE WHEN TRUE THEN E ELSE rand END
// ------------------------------------------------

// doFixMCaseTrueU_Pg: WHERE E → WHERE CASE WHEN TRUE THEN E ELSE rand END
func doFixMCaseTrueU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCaseTrueU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Build CASE WHEN TRUE THEN E ELSE rand END
		oldWhere := sel.WhereClause
		caseExpr := buildPgCaseExpr(
			pgquery.MakeAConstIntNode(1, 0),  // WHEN TRUE (integer 1 in PostgreSQL)
			oldWhere,                         // THEN E
			buildPgRandomBoolExpr(seed),      // ELSE rand
		)

		sel.WhereClause = caseExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMCaseTrueU_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMCaseTrueU_Pg]no WHERE clause found")
}

// ------------------------------------------------
// EET Rule 4: CASE WHEN FALSE wrapping
// E → CASE WHEN FALSE THEN rand ELSE E END
// ------------------------------------------------

// doFixMCaseFalseL_Pg: WHERE E → WHERE CASE WHEN FALSE THEN rand ELSE E END
func doFixMCaseFalseL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCaseFalseL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Build CASE WHEN FALSE THEN rand ELSE E END
		oldWhere := sel.WhereClause
		caseExpr := buildPgCaseExpr(
			pgquery.MakeAConstIntNode(0, 0),  // WHEN FALSE (integer 0 in PostgreSQL)
			buildPgRandomBoolExpr(seed),      // THEN rand
			oldWhere,                         // ELSE E
		)

		sel.WhereClause = caseExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMCaseFalseL_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMCaseFalseL_Pg]no WHERE clause found")
}

// ------------------------------------------------
// EET Rule 5: CASE WHEN rand wrapping
// E → CASE WHEN rand THEN E ELSE E END
// ------------------------------------------------

// doFixMCaseRandEq_Pg: WHERE E → WHERE CASE WHEN rand THEN E ELSE E END
func doFixMCaseRandEq_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCaseRandEq_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Build CASE WHEN rand THEN E ELSE E END
		oldWhere := sel.WhereClause
		randBool := pgquery.MakeAConstIntNode(int64(rander.Intn(2)), 0)
		caseExpr := buildPgCaseExpr(
			randBool,  // WHEN rand
			oldWhere,  // THEN E
			oldWhere,  // ELSE E
		)

		sel.WhereClause = caseExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMCaseRandEq_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMCaseRandEq_Pg]no WHERE clause found")
}

// ------------------------------------------------
// Helper functions for PostgreSQL EET mutations
// ------------------------------------------------

// buildPgPredicate: build a simple predicate p for tautology/contradiction construction
func buildPgPredicate(seed int64) *pgquery.Node {
	rander := rand.New(rand.NewSource(seed))
	val := rander.Intn(100)

	// Use simple comparison: val > 0
	return pgquery.MakeAExprNode(
		pgquery.A_Expr_Kind_AEXPR_OP,
		[]*pgquery.Node{pgquery.MakeStrNode(">")},
		pgquery.MakeAConstIntNode(int64(val), 0),
		pgquery.MakeAConstIntNode(0, 0),
		0,
	)
}

// buildPgTautology: build (p OR NOT p OR p IS NULL) - always TRUE in three-valued logic
func buildPgTautology(p *pgquery.Node, rander *rand.Rand) *pgquery.Node {
	// NOT p
	notP := pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{p}, 0)

	// p OR NOT p
	pOrNotP := pgquery.MakeBoolExprNode(pgquery.BoolExprType_OR_EXPR, []*pgquery.Node{
		p,
		notP,
	}, 0)

	// p IS NULL (construct NullTest manually)
	pIsNull := &pgquery.Node{
		Node: &pgquery.Node_NullTest{
			NullTest: &pgquery.NullTest{
				Arg:       p,
				Nulltesttype: pgquery.NullTestType_IS_NULL,
			},
		},
	}

	// (p OR NOT p) OR p IS NULL
	return pgquery.MakeBoolExprNode(pgquery.BoolExprType_OR_EXPR, []*pgquery.Node{
		pOrNotP,
		pIsNull,
	}, 0)
}

// buildPgContradiction: build (p AND NOT p AND p IS NOT NULL) - always FALSE
func buildPgContradiction(p *pgquery.Node, rander *rand.Rand) *pgquery.Node {
	// NOT p
	notP := pgquery.MakeBoolExprNode(pgquery.BoolExprType_NOT_EXPR, []*pgquery.Node{p}, 0)

	// p AND NOT p
	pAndNotP := pgquery.MakeBoolExprNode(pgquery.BoolExprType_AND_EXPR, []*pgquery.Node{
		p,
		notP,
	}, 0)

	// p IS NOT NULL (construct NullTest manually)
	pIsNotNull := &pgquery.Node{
		Node: &pgquery.Node_NullTest{
			NullTest: &pgquery.NullTest{
				Arg:       p,
				Nulltesttype: pgquery.NullTestType_IS_NOT_NULL,
			},
		},
	}

	// (p AND NOT p) AND p IS NOT NULL
	return pgquery.MakeBoolExprNode(pgquery.BoolExprType_AND_EXPR, []*pgquery.Node{
		pAndNotP,
		pIsNotNull,
	}, 0)
}

// buildPgCaseExpr: build CASE WHEN expr THEN result ELSE elseClause END
func buildPgCaseExpr(expr *pgquery.Node, result *pgquery.Node, elseClause *pgquery.Node) *pgquery.Node {
	// Use MakeCaseWhenNode for WHEN clause
	whenClause := pgquery.MakeCaseWhenNode(expr, result, 0)

	// Manually construct CaseExpr to include defresult (ELSE clause)
	caseExpr := &pgquery.CaseExpr{
		Arg:       nil, // standard CASE WHEN form (no implicit equality argument)
		Args:      []*pgquery.Node{whenClause},
		Defresult: elseClause,
		Location:  0,
	}
	return &pgquery.Node{
		Node: &pgquery.Node_CaseExpr{
			CaseExpr: caseExpr,
		},
	}
}

// buildPgRandomBoolExpr: build a random boolean expression (TRUE or FALSE)
func buildPgRandomBoolExpr(seed int64) *pgquery.Node {
	rander := rand.New(rand.NewSource(seed))
	if rander.Intn(2) == 1 {
		return pgquery.MakeAConstIntNode(1, 0) // TRUE
	}
	return pgquery.MakeAConstIntNode(0, 0) // FALSE
}

// doFixMAndTrueU_Pg_Having: HAVING E → HAVING (p OR NOT p OR p IS NULL) AND E
func doFixMAndTrueU_Pg_Having(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMAndTrueU_Pg_Having]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.HavingClause == nil {
			continue
		}

		pExpr := buildPgPredicate(seed)
		tautology := buildPgTautology(pExpr, rander)

		oldHaving := sel.HavingClause
		sel.HavingClause = pgquery.MakeBoolExprNode(pgquery.BoolExprType_AND_EXPR, []*pgquery.Node{
			tautology,
			oldHaving,
		}, 0)

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.HavingClause = oldHaving
			return "", errors.Wrap(err, "[doFixMAndTrueU_Pg_Having]deparse error")
		}

		sel.HavingClause = oldHaving
		return sql, nil
	}

	return "", errors.New("[doFixMAndTrueU_Pg_Having]no HAVING clause found")
}

// doFixMOrFalseL_Pg_Having: HAVING E → HAVING (p AND NOT p AND p IS NOT NULL) OR E
func doFixMOrFalseL_Pg_Having(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMOrFalseL_Pg_Having]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.HavingClause == nil {
			continue
		}

		pExpr := buildPgPredicate(seed)
		contradiction := buildPgContradiction(pExpr, rander)

		oldHaving := sel.HavingClause
		sel.HavingClause = pgquery.MakeBoolExprNode(pgquery.BoolExprType_OR_EXPR, []*pgquery.Node{
			contradiction,
			oldHaving,
		}, 0)

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.HavingClause = oldHaving
			return "", errors.Wrap(err, "[doFixMOrFalseL_Pg_Having]deparse error")
		}

		sel.HavingClause = oldHaving
		return sql, nil
	}

	return "", errors.New("[doFixMOrFalseL_Pg_Having]no HAVING clause found")
}