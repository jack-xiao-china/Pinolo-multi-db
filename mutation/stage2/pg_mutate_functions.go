package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"math/rand"
	"strings"
)

// PostgreSQL mutation implementation functions
// These functions perform actual mutations on the pg_query AST.

// ------------------------------------------------
// WHERE clause mutations
// ------------------------------------------------

// doFixMWhere1U_Pg: WHERE xxx -> WHERE TRUE
func doFixMWhere1U_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMWhere1U_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find the SelectStmt with WHERE clause
	found := false
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		if sel.WhereClause != nil {
			found = true
			// Save old WHERE clause
			oldWhere := sel.WhereClause
			// Mutate: WHERE xxx -> WHERE TRUE
			sel.WhereClause = pgquery.MakeAConstIntNode(1, 0)
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMWhere1U_Pg]deparse error")
			}
			// Recover
			sel.WhereClause = oldWhere
			return sql, nil
		}
	}

	if !found {
		return "", errors.New("[doFixMWhere1U_Pg]no WHERE clause found")
	}
	return "", errors.New("[doFixMWhere1U_Pg]unexpected error")
}

// doFixMWhere0L_Pg: WHERE xxx -> WHERE FALSE
func doFixMWhere0L_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMWhere0L_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find the SelectStmt with WHERE clause
	found := false
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		if sel.WhereClause != nil {
			found = true
			// Save old WHERE clause
			oldWhere := sel.WhereClause
			// Mutate: WHERE xxx -> WHERE FALSE
			sel.WhereClause = pgquery.MakeAConstIntNode(0, 0)
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMWhere0L_Pg]deparse error")
			}
			// Recover
			sel.WhereClause = oldWhere
			return sql, nil
		}
	}

	if !found {
		return "", errors.New("[doFixMWhere0L_Pg]no WHERE clause found")
	}
	return "", errors.New("[doFixMWhere0L_Pg]unexpected error")
}

// ------------------------------------------------
// HAVING clause mutations
// ------------------------------------------------

// doFixMHaving1U_Pg: HAVING xxx -> HAVING TRUE
func doFixMHaving1U_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMHaving1U_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find the SelectStmt with HAVING clause
	found := false
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		if sel.HavingClause != nil {
			found = true
			// Save old HAVING clause
			oldHaving := sel.HavingClause
			// Mutate: HAVING xxx -> HAVING TRUE
			sel.HavingClause = pgquery.MakeAConstIntNode(1, 0)
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMHaving1U_Pg]deparse error")
			}
			// Recover
			sel.HavingClause = oldHaving
			return sql, nil
		}
	}

	if !found {
		return "", errors.New("[doFixMHaving1U_Pg]no HAVING clause found")
	}
	return "", errors.New("[doFixMHaving1U_Pg]unexpected error")
}

// doFixMHaving0L_Pg: HAVING xxx -> HAVING FALSE
func doFixMHaving0L_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMHaving0L_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find the SelectStmt with HAVING clause
	found := false
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		if sel.HavingClause != nil {
			found = true
			// Save old HAVING clause
			oldHaving := sel.HavingClause
			// Mutate: HAVING xxx -> HAVING FALSE
			sel.HavingClause = pgquery.MakeAConstIntNode(0, 0)
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMHaving0L_Pg]deparse error")
			}
			// Recover
			sel.HavingClause = oldHaving
			return sql, nil
		}
	}

	if !found {
		return "", errors.New("[doFixMHaving0L_Pg]no HAVING clause found")
	}
	return "", errors.New("[doFixMHaving0L_Pg]unexpected error")
}

// ------------------------------------------------
// JOIN ON clause mutations
// ------------------------------------------------

// doFixMOn1U_Pg: ON xxx -> ON TRUE
func doFixMOn1U_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMOn1U_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find JoinExpr with ON condition and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel != nil {
			for _, fromNode := range sel.FromClause {
				sql, found := mutateJoinOnToTrue(fromNode, rootNode)
				if found {
					return sql, nil
				}
			}
		}
	}

	return "", errors.New("[doFixMOn1U_Pg]no JOIN ON clause found")
}

// mutateJoinOnToTrue: find JoinExpr with quals and mutate to TRUE
func mutateJoinOnToTrue(fromNode *pgquery.Node, rootNode *pgquery.ParseResult) (string, bool) {
	if fromNode == nil {
		return "", false
	}

	join := fromNode.GetJoinExpr()
	if join != nil {
		// Skip LEFT/RIGHT JOIN
		if join.Jointype == pgquery.JoinType_JOIN_LEFT || join.Jointype == pgquery.JoinType_JOIN_RIGHT {
			// Still check nested joins
			sql, found := mutateJoinOnToTrue(join.Larg, rootNode)
			if found {
				return sql, true
			}
			sql, found = mutateJoinOnToTrue(join.Rarg, rootNode)
			if found {
				return sql, true
			}
			return "", false
		}
		// Mutate ON quals
		if join.Quals != nil {
			oldQuals := join.Quals
			join.Quals = pgquery.MakeAConstIntNode(1, 0)
			sql, err := pgquery.Deparse(rootNode)
			join.Quals = oldQuals
			if err == nil {
				return sql, true
			}
		}
		// Recursively check nested joins
		sql, found := mutateJoinOnToTrue(join.Larg, rootNode)
		if found {
			return sql, true
		}
		sql, found = mutateJoinOnToTrue(join.Rarg, rootNode)
		if found {
			return sql, true
		}
	}

	return "", false
}

// doFixMOn0L_Pg: ON xxx -> ON FALSE
func doFixMOn0L_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMOn0L_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find JoinExpr with ON condition and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel != nil {
			for _, fromNode := range sel.FromClause {
				sql, found := mutateJoinOnToFalse(fromNode, rootNode)
				if found {
					return sql, nil
				}
			}
		}
	}

	return "", errors.New("[doFixMOn0L_Pg]no JOIN ON clause found")
}

// mutateJoinOnToFalse: find JoinExpr with quals and mutate to FALSE
func mutateJoinOnToFalse(fromNode *pgquery.Node, rootNode *pgquery.ParseResult) (string, bool) {
	if fromNode == nil {
		return "", false
	}

	join := fromNode.GetJoinExpr()
	if join != nil {
		// Skip LEFT/RIGHT JOIN
		if join.Jointype == pgquery.JoinType_JOIN_LEFT || join.Jointype == pgquery.JoinType_JOIN_RIGHT {
			// Still check nested joins
			sql, found := mutateJoinOnToFalse(join.Larg, rootNode)
			if found {
				return sql, true
			}
			sql, found = mutateJoinOnToFalse(join.Rarg, rootNode)
			if found {
				return sql, true
			}
			return "", false
		}
		// Mutate ON quals
		if join.Quals != nil {
			oldQuals := join.Quals
			join.Quals = pgquery.MakeAConstIntNode(0, 0)
			sql, err := pgquery.Deparse(rootNode)
			join.Quals = oldQuals
			if err == nil {
				return sql, true
			}
		}
		// Recursively check nested joins
		sql, found := mutateJoinOnToFalse(join.Larg, rootNode)
		if found {
			return sql, true
		}
		sql, found = mutateJoinOnToFalse(join.Rarg, rootNode)
		if found {
			return sql, true
		}
	}

	return "", false
}

// ------------------------------------------------
// DISTINCT mutations
// ------------------------------------------------

// doFixMDistinctU_Pg: DISTINCT -> remove DISTINCT
func doFixMDistinctU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMDistinctU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		if len(sel.DistinctClause) > 0 {
			// Save old DistinctClause
			oldDistinct := sel.DistinctClause
			// Mutate: remove DISTINCT
			sel.DistinctClause = nil
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMDistinctU_Pg]deparse error")
			}
			// Recover
			sel.DistinctClause = oldDistinct
			return sql, nil
		}
	}

	return "", errors.New("[doFixMDistinctU_Pg]no DISTINCT clause found")
}

// doFixMDistinctL_Pg: add DISTINCT
func doFixMDistinctL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMDistinctL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		// Check conditions: no DISTINCT, no ORDER BY, no WITH, no UNION
		if len(sel.DistinctClause) == 0 && len(sel.SortClause) == 0 &&
			sel.WithClause == nil && sel.Op == pgquery.SetOperation_SET_OPERATION_UNDEFINED {
			// Mutate: add DISTINCT
			// DISTINCT clause in PostgreSQL is a list of expressions
			// Empty list means DISTINCT all columns
			sel.DistinctClause = []*pgquery.Node{}
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMDistinctL_Pg]deparse error")
			}
			// Recover
			sel.DistinctClause = nil
			return sql, nil
		}
	}

	return "", errors.New("[doFixMDistinctL_Pg]conditions not met")
}

// ------------------------------------------------
// UNION mutations
// ------------------------------------------------

// doFixMUnionAllU_Pg: UNION -> UNION ALL
func doFixMUnionAllU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMUnionAllU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		// Find UNION (not UNION ALL)
		if sel.Op == pgquery.SetOperation_SETOP_UNION && !sel.All {
			// Mutate: UNION -> UNION ALL
			sel.All = true
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMUnionAllU_Pg]deparse error")
			}
			// Recover
			sel.All = false
			return sql, nil
		}
	}

	return "", errors.New("[doFixMUnionAllU_Pg]no UNION found")
}

// doFixMUnionAllL_Pg: UNION ALL -> UNION
func doFixMUnionAllL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMUnionAllL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil {
			continue
		}
		// Find UNION ALL
		if sel.Op == pgquery.SetOperation_SETOP_UNION && sel.All {
			// Mutate: UNION ALL -> UNION
			sel.All = false
			// Deparse
			sql, err := pgquery.Deparse(rootNode)
			if err != nil {
				return "", errors.Wrap(err, "[doFixMUnionAllL_Pg]deparse error")
			}
			// Recover
			sel.All = true
			return sql, nil
		}
	}

	return "", errors.New("[doFixMUnionAllL_Pg]no UNION ALL found")
}

// ------------------------------------------------
// Comparison operator mutations
// ------------------------------------------------

// doFixMCmpOpU_Pg: >|<|= -> >=|<=|>=
func doFixMCmpOpU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCmpOpU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find A_Expr with comparison operator and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateCmpOpUpper(rootNode, rawStmt.Stmt)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doFixMCmpOpU_Pg]no comparison operator found for upper mutation")
}

func mutateCmpOpUpper(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node) (string, error) {
	// Check SelectStmt
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateCmpOpUpperInExpr(rootNode, sel.WhereClause)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateCmpOpUpperInExpr(rootNode, sel.HavingClause)
			if found {
				return sql, nil
			}
		}
		// Check FROM clause for JOIN quals
		for _, fromNode := range sel.FromClause {
			sql, found := mutateCmpOpUpperInFrom(rootNode, fromNode)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateCmpOpUpperInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	aExpr := exprNode.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_OP {
		opName := getAExprOperatorName(aExpr)
		switch opName {
		case ">":
			// > -> >=
			newName := []*pgquery.Node{pgquery.MakeStrNode(">=")}
			oldName := aExpr.Name
			aExpr.Name = newName
			sql, err := pgquery.Deparse(rootNode)
			aExpr.Name = oldName
			if err == nil {
				return sql, true
			}
		case "<":
			// < -> <=
			newName := []*pgquery.Node{pgquery.MakeStrNode("<=")}
			oldName := aExpr.Name
			aExpr.Name = newName
			sql, err := pgquery.Deparse(rootNode)
			aExpr.Name = oldName
			if err == nil {
				return sql, true
			}
		case "=":
			// = -> >=
			newName := []*pgquery.Node{pgquery.MakeStrNode(">=")}
			oldName := aExpr.Name
			aExpr.Name = newName
			sql, err := pgquery.Deparse(rootNode)
			aExpr.Name = oldName
			if err == nil {
				return sql, true
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateCmpOpUpperInExpr(rootNode, arg)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

func mutateCmpOpUpperInFrom(rootNode *pgquery.ParseResult, fromNode *pgquery.Node) (string, bool) {
	if fromNode == nil {
		return "", false
	}

	join := fromNode.GetJoinExpr()
	if join != nil {
		// Skip LEFT/RIGHT JOIN
		if join.Jointype == pgquery.JoinType_JOIN_LEFT || join.Jointype == pgquery.JoinType_JOIN_RIGHT {
			// Still check nested joins
			sql, found := mutateCmpOpUpperInFrom(rootNode, join.Larg)
			if found {
				return sql, true
			}
			sql, found = mutateCmpOpUpperInFrom(rootNode, join.Rarg)
			if found {
				return sql, true
			}
			return "", false
		}
		// Check ON quals
		if join.Quals != nil {
			sql, found := mutateCmpOpUpperInExpr(rootNode, join.Quals)
			if found {
				return sql, true
			}
		}
		// Recursively check nested joins
		sql, found := mutateCmpOpUpperInFrom(rootNode, join.Larg)
		if found {
			return sql, true
		}
		sql, found = mutateCmpOpUpperInFrom(rootNode, join.Rarg)
		if found {
			return sql, true
		}
	}

	return "", false
}

// doFixMCmpOpL_Pg: >=|<= -> >|<
// NOTE: != and <> are NOT included because != -> < has no valid containment relationship.
func doFixMCmpOpL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMCmpOpL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find A_Expr with comparison operator and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateCmpOpLower(rootNode, rawStmt.Stmt)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doFixMCmpOpL_Pg]no comparison operator found for lower mutation")
}

func mutateCmpOpLower(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateCmpOpLowerInExpr(rootNode, sel.WhereClause)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateCmpOpLowerInExpr(rootNode, sel.HavingClause)
			if found {
				return sql, nil
			}
		}
		// Check FROM clause for JOIN quals
		for _, fromNode := range sel.FromClause {
			sql, found := mutateCmpOpLowerInFrom(rootNode, fromNode)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateCmpOpLowerInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	aExpr := exprNode.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_OP {
		opName := getAExprOperatorName(aExpr)
		switch opName {
		case ">=":
			// >= -> >
			newName := []*pgquery.Node{pgquery.MakeStrNode(">")}
			oldName := aExpr.Name
			aExpr.Name = newName
			sql, err := pgquery.Deparse(rootNode)
			aExpr.Name = oldName
			if err == nil {
				return sql, true
			}
		case "<=":
			// <= -> <
			newName := []*pgquery.Node{pgquery.MakeStrNode("<")}
			oldName := aExpr.Name
			aExpr.Name = newName
			sql, err := pgquery.Deparse(rootNode)
			aExpr.Name = oldName
			if err == nil {
				return sql, true
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateCmpOpLowerInExpr(rootNode, arg)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

func mutateCmpOpLowerInFrom(rootNode *pgquery.ParseResult, fromNode *pgquery.Node) (string, bool) {
	if fromNode == nil {
		return "", false
	}

	join := fromNode.GetJoinExpr()
	if join != nil {
		// Skip LEFT/RIGHT JOIN
		if join.Jointype == pgquery.JoinType_JOIN_LEFT || join.Jointype == pgquery.JoinType_JOIN_RIGHT {
			sql, found := mutateCmpOpLowerInFrom(rootNode, join.Larg)
			if found {
				return sql, true
			}
			sql, found = mutateCmpOpLowerInFrom(rootNode, join.Rarg)
			if found {
				return sql, true
			}
			return "", false
		}
		// Check ON quals
		if join.Quals != nil {
			sql, found := mutateCmpOpLowerInExpr(rootNode, join.Quals)
			if found {
				return sql, true
			}
		}
		// Recursively check nested joins
		sql, found := mutateCmpOpLowerInFrom(rootNode, join.Larg)
		if found {
			return sql, true
		}
		sql, found = mutateCmpOpLowerInFrom(rootNode, join.Rarg)
		if found {
			return sql, true
		}
	}

	return "", false
}

// ------------------------------------------------
// IN clause mutation
// ------------------------------------------------

// doFixMInNullU_Pg: IN(x,x,x) -> IN(x,x,x,NULL)
func doFixMInNullU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMInNullU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	// Find A_Expr with IN kind and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateInClause(rootNode, rawStmt.Stmt)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doFixMInNullU_Pg]no IN clause found")
}

func mutateInClause(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateInClauseInExpr(rootNode, sel.WhereClause)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateInClauseInExpr(rootNode, sel.HavingClause)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateInClauseInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	aExpr := exprNode.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_IN {
		// Check if Rexpr is a List
		if aExpr.Rexpr != nil {
			list := aExpr.Rexpr.GetList()
			if list != nil && len(list.Items) > 0 {
				// Add NULL to the list
				nullNode := &pgquery.Node{
					Node: &pgquery.Node_AConst{
						AConst: &pgquery.A_Const{
							Isnull: true,
						},
					},
				}
				oldItems := list.Items
				newItems := append(oldItems, nullNode)
				list.Items = newItems
				sql, err := pgquery.Deparse(rootNode)
				list.Items = oldItems
				if err == nil {
					return sql, true
				}
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateInClauseInExpr(rootNode, arg)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

// ------------------------------------------------
// LIKE mutations (PostgreSQL uses ~~ internally)
// ------------------------------------------------

// doRdMLikePgU: LIKE pattern expansion - normal char -> '%'|'_', '_' -> '%'
func doRdMLikePgU(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doRdMLikePgU]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	// Find A_Expr with LIKE kind and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateLikePatternUpper(rootNode, rawStmt.Stmt, rander)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doRdMLikePgU]no LIKE clause found")
}

func mutateLikePatternUpper(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node, rander *rand.Rand) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateLikePatternUpperInExpr(rootNode, sel.WhereClause, rander)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateLikePatternUpperInExpr(rootNode, sel.HavingClause, rander)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateLikePatternUpperInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node, rander *rand.Rand) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	aExpr := exprNode.GetAExpr()
	if aExpr != nil && (aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_LIKE || aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_ILIKE) {
		// Check if Rexpr (pattern) is a constant string
		if aExpr.Rexpr != nil {
			aConst := aExpr.Rexpr.GetAConst()
			if aConst != nil {
				pattern := getAConstString(aConst)
				if pattern != "" {
					// Expand pattern: non-'%' chars -> '%'
					newPattern := []byte(pattern)
					for i, c := range newPattern {
						if c != '%' && rander.Intn(2) == 0 {
							newPattern[i] = '%'
						}
					}
					// Create new pattern node
					oldArg := aExpr.Rexpr
					aExpr.Rexpr = pgquery.MakeAConstStrNode(string(newPattern), 0)
					sql, err := pgquery.Deparse(rootNode)
					aExpr.Rexpr = oldArg
					if err == nil {
						return sql, true
					}
				}
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateLikePatternUpperInExpr(rootNode, arg, rander)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

// doRdMLikePgL: LIKE pattern contraction - '%' -> '_'
func doRdMLikePgL(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doRdMLikePgL]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	// Find A_Expr with LIKE kind and mutate
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateLikePatternLower(rootNode, rawStmt.Stmt, rander)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doRdMLikePgL]no LIKE clause found")
}

func mutateLikePatternLower(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node, rander *rand.Rand) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateLikePatternLowerInExpr(rootNode, sel.WhereClause, rander)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateLikePatternLowerInExpr(rootNode, sel.HavingClause, rander)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateLikePatternLowerInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node, rander *rand.Rand) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	aExpr := exprNode.GetAExpr()
	if aExpr != nil && (aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_LIKE || aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_ILIKE) {
		// Check if Rexpr (pattern) is a constant string
		if aExpr.Rexpr != nil {
			aConst := aExpr.Rexpr.GetAConst()
			if aConst != nil {
				pattern := getAConstString(aConst)
				if pattern != "" && strings.Contains(pattern, "%") {
					// Contract pattern: '%' -> '_'
					newPattern := []byte(pattern)
					for i, c := range newPattern {
						if c == '%' && rander.Intn(2) == 0 {
							newPattern[i] = '_'
						}
					}
					// Create new pattern node
					oldArg := aExpr.Rexpr
					aExpr.Rexpr = pgquery.MakeAConstStrNode(string(newPattern), 0)
					sql, err := pgquery.Deparse(rootNode)
					aExpr.Rexpr = oldArg
					if err == nil {
						return sql, true
					}
				}
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateLikePatternLowerInExpr(rootNode, arg, rander)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

// ------------------------------------------------
// REGEXP mutations (PostgreSQL uses ~ operator)
// ------------------------------------------------

// doRdMRegExpPgU: REGEXP pattern expansion - '^'|'$' -> '', '+'|'?' -> '*'
func doRdMRegExpPgU(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doRdMRegExpPgU]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	// In PostgreSQL, REGEXP is handled via ~ operator (OpExpr) or SIMILAR TO (A_Expr)
	// We'll check both
	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateRegExpPatternUpper(rootNode, rawStmt.Stmt, rander)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doRdMRegExpPgU]no REGEXP clause found")
}

func mutateRegExpPatternUpper(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node, rander *rand.Rand) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateRegExpPatternUpperInExpr(rootNode, sel.WhereClause, rander)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateRegExpPatternUpperInExpr(rootNode, sel.HavingClause, rander)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateRegExpPatternUpperInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node, rander *rand.Rand) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	// Check for OpExpr (~ operator for regex)
	opExpr := exprNode.GetOpExpr()
	if opExpr != nil {
		// Check if operator name is "~" (regex match)
		// The operator name is typically stored differently in OpExpr
		// For now, we'll check if this is a regex pattern by looking at the right expression
		if opExpr.Args[1] != nil {
			aConst := opExpr.Args[1].GetAConst()
			if aConst != nil {
				pattern := getAConstString(aConst)
				if pattern != "" {
					// Expand regex pattern
					newPattern := []byte(pattern)
					// Remove ^ prefix
					if strings.HasPrefix(pattern, "^") && rander.Intn(2) == 0 {
						newPattern = newPattern[1:]
					}
					// Remove $ suffix
					if strings.HasSuffix(string(newPattern), "$") && rander.Intn(2) == 0 {
						newPattern = newPattern[:len(newPattern)-1]
					}
					// + or ? -> *
					for i, c := range newPattern {
						if (c == '+' || c == '?') && rander.Intn(2) == 0 {
							newPattern[i] = '*'
						}
					}
					// Apply mutation
					oldArg := opExpr.Args[1]
					opExpr.Args[1] = pgquery.MakeAConstStrNode(string(newPattern), 0)
					sql, err := pgquery.Deparse(rootNode)
					opExpr.Args[1] = oldArg
					if err == nil {
						return sql, true
					}
				}
			}
		}
	}

	// Check for SIMILAR TO
	aExpr := exprNode.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_SIMILAR {
		if aExpr.Rexpr != nil {
			aConst := aExpr.Rexpr.GetAConst()
			if aConst != nil {
				pattern := getAConstString(aConst)
				if pattern != "" {
					newPattern := []byte(pattern)
					if strings.HasPrefix(pattern, "^") && rander.Intn(2) == 0 {
						newPattern = newPattern[1:]
					}
					if strings.HasSuffix(string(newPattern), "$") && rander.Intn(2) == 0 {
						newPattern = newPattern[:len(newPattern)-1]
					}
					for i, c := range newPattern {
						if (c == '+' || c == '?') && rander.Intn(2) == 0 {
							newPattern[i] = '*'
						}
					}
					oldArg := aExpr.Rexpr
					aExpr.Rexpr = pgquery.MakeAConstStrNode(string(newPattern), 0)
					sql, err := pgquery.Deparse(rootNode)
					aExpr.Rexpr = oldArg
					if err == nil {
						return sql, true
					}
				}
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateRegExpPatternUpperInExpr(rootNode, arg, rander)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

// doRdMRegExpPgL: REGEXP pattern contraction - '*' -> '+'|'?'
func doRdMRegExpPgL(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doRdMRegExpPgL]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	rander := rand.New(rand.NewSource(seed))

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sql, err := mutateRegExpPatternLower(rootNode, rawStmt.Stmt, rander)
		if err == nil && sql != "" {
			return sql, nil
		}
	}

	return "", errors.New("[doRdMRegExpPgL]no REGEXP clause found")
}

func mutateRegExpPatternLower(rootNode *pgquery.ParseResult, stmtNode *pgquery.Node, rander *rand.Rand) (string, error) {
	sel := stmtNode.GetSelectStmt()
	if sel != nil {
		// Check WHERE clause
		if sel.WhereClause != nil {
			sql, found := mutateRegExpPatternLowerInExpr(rootNode, sel.WhereClause, rander)
			if found {
				return sql, nil
			}
		}
		// Check HAVING clause
		if sel.HavingClause != nil {
			sql, found := mutateRegExpPatternLowerInExpr(rootNode, sel.HavingClause, rander)
			if found {
				return sql, nil
			}
		}
	}
	return "", errors.New("not found")
}

func mutateRegExpPatternLowerInExpr(rootNode *pgquery.ParseResult, exprNode *pgquery.Node, rander *rand.Rand) (string, bool) {
	if exprNode == nil {
		return "", false
	}

	// Check for OpExpr (~ operator for regex)
	opExpr := exprNode.GetOpExpr()
	if opExpr != nil && opExpr.Args[1] != nil {
		aConst := opExpr.Args[1].GetAConst()
		if aConst != nil {
			pattern := getAConstString(aConst)
			if pattern != "" && strings.Contains(pattern, "*") {
				// Contract regex pattern: * -> + or ?
				newPattern := []byte(pattern)
				for i, c := range newPattern {
					if c == '*' && rander.Intn(2) == 0 {
						if rander.Intn(2) == 0 {
							newPattern[i] = '+'
						} else {
							newPattern[i] = '?'
						}
					}
				}
				oldArg := opExpr.Args[1]
				opExpr.Args[1] = pgquery.MakeAConstStrNode(string(newPattern), 0)
				sql, err := pgquery.Deparse(rootNode)
				opExpr.Args[1] = oldArg
				if err == nil {
					return sql, true
				}
			}
		}
	}

	// Check for SIMILAR TO
	aExpr := exprNode.GetAExpr()
	if aExpr != nil && aExpr.Kind == pgquery.A_Expr_Kind_AEXPR_SIMILAR && aExpr.Rexpr != nil {
		aConst := aExpr.Rexpr.GetAConst()
		if aConst != nil {
			pattern := getAConstString(aConst)
			if pattern != "" && strings.Contains(pattern, "*") {
				newPattern := []byte(pattern)
				for i, c := range newPattern {
					if c == '*' && rander.Intn(2) == 0 {
						if rander.Intn(2) == 0 {
							newPattern[i] = '+'
						} else {
							newPattern[i] = '?'
						}
					}
				}
				oldArg := aExpr.Rexpr
				aExpr.Rexpr = pgquery.MakeAConstStrNode(string(newPattern), 0)
				sql, err := pgquery.Deparse(rootNode)
				aExpr.Rexpr = oldArg
				if err == nil {
					return sql, true
				}
			}
		}
	}

	// Check BoolExpr for nested expressions
	boolExpr := exprNode.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			sql, found := mutateRegExpPatternLowerInExpr(rootNode, arg, rander)
			if found {
				return sql, true
			}
		}
	}

	return "", false
}

// ------------------------------------------------
// Helper functions
// ------------------------------------------------

// getAExprOperatorName: get the operator name from an A_Expr
func getAExprOperatorName(expr *pgquery.A_Expr) string {
	if expr == nil || len(expr.Name) == 0 {
		return ""
	}
	for _, nameNode := range expr.Name {
		if nameNode != nil {
			str := nameNode.GetString_()
			if str != nil && str.Sval != "" {
				return str.Sval
			}
		}
	}
	return ""
}

// getAConstString: get the string value from an A_Const node
func getAConstString(aConst *pgquery.A_Const) string {
	if aConst == nil {
		return ""
	}
	switch aConst.Val.(type) {
	case *pgquery.A_Const_Sval:
		sval := aConst.GetSval()
		if sval != nil {
			return sval.Sval
		}
	case *pgquery.A_Const_Bsval:
		bsval := aConst.GetBsval()
		if bsval != nil {
			return bsval.Bsval
		}
	}
	return ""
}