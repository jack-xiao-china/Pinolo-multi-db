package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"strings"
)

// PostgreSQL EET COALESCE -> CASE and NULLIF -> CASE transformation mutations

// addFixMCoalesceToCase_Pg: FixMCoalesceToCase_Pg, FuncCall(COALESCE): COALESCE(a, b) -> CASE WHEN a IS NOT NULL THEN a ELSE b END
func (v *PgMutateVisitor) addFixMCoalesceToCase_Pg(node *pgquery.Node, flag int) {
	if node == nil {
		return
	}
	funcCall := node.GetFuncCall()
	if funcCall != nil && isPgCoalesceFunc(funcCall) && len(funcCall.Args) >= 2 {
		v.addPgCandidate(FixMCoalesceToCase_Pg, 1, node, flag)
	}
}

// addFixMNullifToCase_Pg: FixMNullifToCase_Pg, FuncCall(NULLIF): NULLIF(a, b) -> CASE WHEN a = b THEN NULL ELSE a END
func (v *PgMutateVisitor) addFixMNullifToCase_Pg(node *pgquery.Node, flag int) {
	if node == nil {
		return
	}
	funcCall := node.GetFuncCall()
	if funcCall != nil && isPgNullifFunc(funcCall) && len(funcCall.Args) == 2 {
		v.addPgCandidate(FixMNullifToCase_Pg, 1, node, flag)
	}
}

// isPgCoalesceFunc: check if a FuncCall is COALESCE
func isPgCoalesceFunc(funcCall *pgquery.FuncCall) bool {
	if funcCall == nil || len(funcCall.Funcname) == 0 {
		return false
	}
	// Funcname is a list of String nodes (may include schema name)
	// The last element is the actual function name
	lastNameNode := funcCall.Funcname[len(funcCall.Funcname)-1]
	strNode := lastNameNode.GetString_()
	if strNode != nil {
		return strings.EqualFold(strNode.Sval, "coalesce")
	}
	return false
}

// isPgNullifFunc: check if a FuncCall is NULLIF
func isPgNullifFunc(funcCall *pgquery.FuncCall) bool {
	if funcCall == nil || len(funcCall.Funcname) == 0 {
		return false
	}
	lastNameNode := funcCall.Funcname[len(funcCall.Funcname)-1]
	strNode := lastNameNode.GetString_()
	if strNode != nil {
		return strings.EqualFold(strNode.Sval, "nullif")
	}
	return false
}

// doFixMCoalesceToCase_Pg: FixMCoalesceToCase_Pg, COALESCE(a, b) -> CASE WHEN a IS NOT NULL THEN a ELSE b END
// For multi-argument COALESCE, recursively nest CASE expressions.
func doFixMCoalesceToCase_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCoalesceToCase_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find a COALESCE FuncCall in the WHERE clause
		coalesceNode := findFuncCallInWhere(sel.WhereClause, isPgCoalesceFunc)
		if coalesceNode == nil {
			continue
		}

		funcCall := coalesceNode.GetFuncCall()
		if funcCall == nil || len(funcCall.Args) < 2 {
			continue
		}

		// Save original WHERE clause
		oldWhere := sel.WhereClause

		// Build CASE WHEN equivalent for COALESCE
		caseExpr := buildPgCoalesceCaseExpr(funcCall.Args)

		sel.WhereClause = caseExpr

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMCoalesceToCase_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMCoalesceToCase_Pg]no COALESCE function found")
}

// doFixMNullifToCase_Pg: FixMNullifToCase_Pg, NULLIF(a, b) -> CASE WHEN a = b THEN NULL ELSE a END
func doFixMNullifToCase_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMNullifToCase_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		nullifNode := findFuncCallInWhere(sel.WhereClause, isPgNullifFunc)
		if nullifNode == nil {
			continue
		}

		funcCall := nullifNode.GetFuncCall()
		if funcCall == nil || len(funcCall.Args) != 2 {
			continue
		}

		a := funcCall.Args[0]
		b := funcCall.Args[1]

		oldWhere := sel.WhereClause

		// CASE WHEN a = b THEN NULL ELSE a END
		// a = b (comparison)
		eqExpr := pgquery.MakeAExprNode(
			pgquery.A_Expr_Kind_AEXPR_OP,
			[]*pgquery.Node{pgquery.MakeStrNode("=")},
			a,
			b,
			0,
		)

		// THEN NULL
		nullResult := makePgNullConstNode(0)

		// CASE WHEN a = b THEN NULL ELSE a END
		whenClause := pgquery.MakeCaseWhenNode(eqExpr, nullResult, 0)
		caseExpr := &pgquery.CaseExpr{
			Args:      []*pgquery.Node{whenClause},
			Defresult: a,
			Location:  0,
		}
		caseNode := &pgquery.Node{
			Node: &pgquery.Node_CaseExpr{
				CaseExpr: caseExpr,
			},
		}

		sel.WhereClause = caseNode

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			return "", errors.Wrap(err, "[doFixMNullifToCase_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMNullifToCase_Pg]no NULLIF function found")
}

// buildPgCoalesceCaseExpr: recursively build CASE expression equivalent to COALESCE
// COALESCE(a1, a2, ..., an) -> CASE WHEN a1 IS NOT NULL THEN a1 ELSE (recursive) END
func buildPgCoalesceCaseExpr(args []*pgquery.Node) *pgquery.Node {
	if len(args) == 1 {
		// COALESCE(a) -> a (single argument)
		whenClause := pgquery.MakeCaseWhenNode(
			buildPgIsNotNullNode(args[0]),
			args[0],
			0,
		)
		caseExpr := &pgquery.CaseExpr{
			Args:      []*pgquery.Node{whenClause},
			Defresult: makePgNullConstNode(0),
			Location:  0,
		}
		return &pgquery.Node{
			Node: &pgquery.Node_CaseExpr{
				CaseExpr: caseExpr,
			},
		}
	}

	// COALESCE(a1, rest) -> CASE WHEN a1 IS NOT NULL THEN a1 ELSE COALESCE(rest) END
	whenClause := pgquery.MakeCaseWhenNode(
		buildPgIsNotNullNode(args[0]),
		args[0],
		0,
	)

	elseExpr := buildPgCoalesceCaseExpr(args[1:])

	caseExpr := &pgquery.CaseExpr{
		Args:      []*pgquery.Node{whenClause},
		Defresult: elseExpr,
		Location:  0,
	}

	return &pgquery.Node{
		Node: &pgquery.Node_CaseExpr{
			CaseExpr: caseExpr,
		},
	}
}

// buildPgIsNotNullNode: build a NullTest node for IS NOT NULL
func buildPgIsNotNullNode(expr *pgquery.Node) *pgquery.Node {
	return &pgquery.Node{
		Node: &pgquery.Node_NullTest{
			NullTest: &pgquery.NullTest{
				Arg:          expr,
				Nulltesttype: pgquery.NullTestType_IS_NOT_NULL,
				Location:     0,
			},
		},
	}
}

// findFuncCallInWhere: recursively search for a specific FuncCall in WHERE clause
func findFuncCallInWhere(node *pgquery.Node, matchFunc func(*pgquery.FuncCall) bool) *pgquery.Node {
	if node == nil {
		return nil
	}

	// Check if this node is a matching FuncCall
	funcCall := node.GetFuncCall()
	if funcCall != nil && matchFunc(funcCall) {
		return node
	}

	// Search in BoolExpr args
	boolExpr := node.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			result := findFuncCallInWhere(arg, matchFunc)
			if result != nil {
				return result
			}
		}
	}

	// Search in A_Expr sub-expressions
	aExpr := node.GetAExpr()
	if aExpr != nil {
		if aExpr.Lexpr != nil {
			result := findFuncCallInWhere(aExpr.Lexpr, matchFunc)
			if result != nil {
				return result
			}
		}
		if aExpr.Rexpr != nil {
			result := findFuncCallInWhere(aExpr.Rexpr, matchFunc)
			if result != nil {
				return result
			}
		}
	}

	// Search in CaseExpr
	caseExpr := node.GetCaseExpr()
	if caseExpr != nil {
		for _, arg := range caseExpr.Args {
			whenClause := arg.GetCaseWhen()
			if whenClause != nil {
				result := findFuncCallInWhere(whenClause.Expr, matchFunc)
				if result != nil {
					return result
				}
				result = findFuncCallInWhere(whenClause.Result, matchFunc)
				if result != nil {
					return result
				}
			}
		}
		if caseExpr.Defresult != nil {
			result := findFuncCallInWhere(caseExpr.Defresult, matchFunc)
			if result != nil {
				return result
			}
		}
	}

	return nil
}

// makePgNullConstNode: create a NULL constant node for PostgreSQL AST
func makePgNullConstNode(location int32) *pgquery.Node {
	return &pgquery.Node{
		Node: &pgquery.Node_AConst{
			AConst: &pgquery.A_Const{
				Isnull:   true,
				Location: location,
			},
		},
	}
}
