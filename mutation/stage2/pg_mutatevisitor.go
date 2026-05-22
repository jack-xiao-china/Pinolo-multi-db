package stage2

import (
	"log"
	"strings"

	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL mutation names
const (
	// *pgquery.SelectStmt: WHERE xxx -> WHERE TRUE
	FixMWhere1U_Pg = "FixMWhere1U_Pg"
	// *pgquery.SelectStmt: WHERE xxx -> WHERE FALSE
	FixMWhere0L_Pg = "FixMWhere0L_Pg"

	// *pgquery.SelectStmt: HAVING xxx -> HAVING TRUE
	FixMHaving1U_Pg = "FixMHaving1U_Pg"
	// *pgquery.SelectStmt: HAVING xxx -> HAVING FALSE
	FixMHaving0L_Pg = "FixMHaving0L_Pg"

	// *pgquery.JoinExpr: ON xxx -> ON TRUE
	FixMOn1U_Pg = "FixMOn1U_Pg"
	// *pgquery.JoinExpr: ON xxx -> ON FALSE
	FixMOn0L_Pg = "FixMOn0L_Pg"

	// *pgquery.SelectStmt: DistinctClause non-empty -> empty
	FixMDistinctU_Pg = "FixMDistinctU_Pg"
	// *pgquery.SelectStmt: DistinctClause empty -> non-empty (add DISTINCT)
	FixMDistinctL_Pg = "FixMDistinctL_Pg"

	// *pgquery.SelectStmt (UNION): Op UNION & All false -> All true (UNION -> UNION ALL)
	FixMUnionAllU_Pg = "FixMUnionAllU_Pg"
	// *pgquery.SelectStmt (UNION): Op UNION & All true -> All false (UNION ALL -> UNION)
	FixMUnionAllL_Pg = "FixMUnionAllL_Pg"

	// *pgquery.A_Expr (comparison): >|<|= -> >=|<=|>=
	FixMCmpOpU_Pg = "FixMCmpOpU_Pg"
	// *pgquery.A_Expr (comparison): >=|<=|!= -> >|<|<
	FixMCmpOpL_Pg = "FixMCmpOpL_Pg"

	// *pgquery.A_Expr (IN): IN(x,x,x) -> IN(x,x,x,NULL)
	FixMInNullU_Pg = "FixMInNullU_Pg"

	// *pgquery.A_Expr (LIKE/~~): normal char -> '_'|'%', '_' -> '%'
	RdMLikePgU = "RdMLikePgU"
	// *pgquery.A_Expr (LIKE/~~): '%' -> '_'
	RdMLikePgL = "RdMLikePgL"

	// *pgquery.A_Expr (REGEXP/~): '^'|'$' -> '', '+'|'?' -> '*'
	RdMRegExpPgU = "RdMRegExpPgU"
	// *pgquery.A_Expr (REGEXP/~): '*' -> '+'|'?'
	RdMRegExpPgL = "RdMRegExpPgL"
)

// PgCandidate: mutation candidate for PostgreSQL AST
// Similar to Candidate but for pg_query AST nodes.
type PgCandidate struct {
	MutationName string      // mutation name
	U            int         // 1: upper mutation, 0: lower mutation
	Node         *pgquery.Node // candidate node (pointer to Node in AST)
	Flag         int         // 1: positive, 0: negative
}

// PgMutateVisitor: PostgreSQL AST mutation visitor
// Traverses pg_query ParseResult AST to find mutation candidates.
type PgMutateVisitor struct {
	Root      *pgquery.ParseResult
	Candidates map[string][]*PgCandidate // mutation name : slice of *PgCandidate
}

// FindCandidates: traverse the AST and find all mutation candidates
func (v *PgMutateVisitor) FindCandidates(result *pgquery.ParseResult, flag int) {
	if result == nil || len(result.Stmts) == 0 {
		return
	}

	// Process each statement in the ParseResult
	for _, rawStmt := range result.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		v.visitNode(rawStmt.Stmt, flag)
	}
}

// visitNode: visit a pg_query Node and find candidates
func (v *PgMutateVisitor) visitNode(node *pgquery.Node, flag int) {
	if node == nil {
		return
	}

	switch node.Node.(type) {
	case *pgquery.Node_SelectStmt:
		v.visitSelectStmt(node.GetSelectStmt(), flag)
	case *pgquery.Node_JoinExpr:
		v.visitJoinExpr(node.GetJoinExpr(), flag)
	case *pgquery.Node_AExpr:
		v.visitAExpr(node.GetAExpr(), flag)
	case *pgquery.Node_BoolExpr:
		v.visitBoolExpr(node.GetBoolExpr(), flag)
		// Handle parenthesized expressions if present
	case *pgquery.Node_SubLink:
		v.visitSubLink(node.GetSubLink(), flag)
	default:
		// For other node types, recursively visit child nodes
		v.visitChildren(node, flag)
	}
}

// visitSelectStmt: visit a SELECT statement and find candidates
func (v *PgMutateVisitor) visitSelectStmt(sel *pgquery.SelectStmt, flag int) {
	if sel == nil {
		return
	}

	// Check if this is a UNION query (Op != SET_OPERATION_UNDEFINED)
	if sel.Op != pgquery.SetOperation_SET_OPERATION_UNDEFINED {
		// This is a UNION/INTERSECT/EXCEPT query
		v.miningUnionSelectStmt(sel, flag)
		// Also visit left and right arguments
		v.visitSelectStmt(sel.Larg, flag)
		v.visitSelectStmt(sel.Rarg, flag)
		return
	}

	// Regular SELECT statement

	// Visit FROM clause (joins)
	for _, fromNode := range sel.FromClause {
		v.visitNode(fromNode, flag)
	}

	// Visit WHERE clause
	if sel.WhereClause != nil {
		v.visitNode(sel.WhereClause, flag)
		v.miningWhereClause(sel, flag)
	}

	// Visit HAVING clause
	if sel.HavingClause != nil {
		v.visitNode(sel.HavingClause, flag)
		v.miningHavingClause(sel, flag)
	}

	// Visit GROUP clause (may contain expressions)
	for _, groupNode := range sel.GroupClause {
		v.visitNode(groupNode, flag)
	}

	// Visit WINDOW clause
	for _, windowNode := range sel.WindowClause {
		v.visitNode(windowNode, flag)
	}

	// Visit WITH clause
	if sel.WithClause != nil {
		v.visitWithClause(sel.WithClause, flag)
	}

	// Mining mutations for this SELECT
	v.miningSelectStmt(sel, flag)
}

// visitJoinExpr: visit a JOIN expression and find candidates
func (v *PgMutateVisitor) visitJoinExpr(join *pgquery.JoinExpr, flag int) {
	if join == nil {
		return
	}

	// Skip LEFT/RIGHT JOIN - these have different semantics
	if join.Jointype == pgquery.JoinType_JOIN_LEFT || join.Jointype == pgquery.JoinType_JOIN_RIGHT {
		return
	}

	// Visit left and right arguments
	v.visitNode(join.Larg, flag)
	v.visitNode(join.Rarg, flag)

	// Visit ON condition (quals)
	if join.Quals != nil {
		v.visitNode(join.Quals, flag)
		v.miningJoinQuals(join, flag)
	}
}

// visitAExpr: visit an A_Expr (expression) and find candidates
func (v *PgMutateVisitor) visitAExpr(expr *pgquery.A_Expr, flag int) {
	if expr == nil {
		return
	}

	// Check for NOT prefix (if name contains "!")
	// Note: In pg_query, NOT is handled differently

	// Visit left expression
	if expr.Lexpr != nil {
		v.visitNode(expr.Lexpr, flag)
	}

	// Visit right expression
	if expr.Rexpr != nil {
		v.visitNode(expr.Rexpr, flag)
	}

	// Mining mutations based on expression kind
	switch expr.Kind {
	case pgquery.A_Expr_Kind_AEXPR_OP:
		// Comparison operator: >, <, =, >=, <=, !=
		v.miningCmpOp(expr, flag)
	case pgquery.A_Expr_Kind_AEXPR_IN:
		// IN clause
		if !v.isNotAExpr(expr) {
			v.miningInClause(expr, flag)
		} else {
			v.miningInClause(expr, flag^1)
		}
	case pgquery.A_Expr_Kind_AEXPR_LIKE:
		// LIKE (~~ operator in PostgreSQL internal representation)
		if !v.isNotAExpr(expr) {
			v.miningLikeExpr(expr, flag)
		} else {
			v.miningLikeExpr(expr, flag^1)
		}
	case pgquery.A_Expr_Kind_AEXPR_ILIKE:
		// ILIKE (~~* operator)
		if !v.isNotAExpr(expr) {
			v.miningLikeExpr(expr, flag)
		} else {
			v.miningLikeExpr(expr, flag^1)
		}
	case pgquery.A_Expr_Kind_AEXPR_SIMILAR:
		// SIMILAR TO (regex-like pattern)
		if !v.isNotAExpr(expr) {
			v.miningRegExpExpr(expr, flag)
		} else {
			v.miningRegExpExpr(expr, flag^1)
		}
	}
}

// visitBoolExpr: visit a boolean expression (AND/OR/NOT)
func (v *PgMutateVisitor) visitBoolExpr(expr *pgquery.BoolExpr, flag int) {
	if expr == nil {
		return
	}

	switch expr.Boolop {
	case pgquery.BoolExprType_AND_EXPR:
		// AND: visit all arguments with same flag
		for _, arg := range expr.Args {
			v.visitNode(arg, flag)
		}
	case pgquery.BoolExprType_OR_EXPR:
		// OR: visit all arguments with same flag
		for _, arg := range expr.Args {
			v.visitNode(arg, flag)
		}
	case pgquery.BoolExprType_NOT_EXPR:
		// NOT: invert flag and visit the single argument
		if len(expr.Args) > 0 {
			v.visitNode(expr.Args[0], flag^1)
		}
	}
}

// visitSubLink: visit a subquery link
func (v *PgMutateVisitor) visitSubLink(sublink *pgquery.SubLink, flag int) {
	if sublink == nil {
		return
	}

	// Visit the subquery
	if sublink.Subselect != nil {
		v.visitNode(sublink.Subselect, flag)
	}

	// Visit test expression if present
	if sublink.Testexpr != nil {
		v.visitNode(sublink.Testexpr, flag)
	}
}

// visitWithClause: visit a WITH clause (CTEs)
func (v *PgMutateVisitor) visitWithClause(withClause *pgquery.WithClause, flag int) {
	if withClause == nil {
		return
	}

	// Skip recursive WITH clauses
	if withClause.Recursive {
		return
	}

	// Visit each CTE
	for _, cte := range withClause.Ctes {
		if cte == nil {
			continue
		}
		commonTableExpr := cte.GetCommonTableExpr()
		if commonTableExpr != nil && commonTableExpr.Ctequery != nil {
			v.visitNode(commonTableExpr.Ctequery, flag)
		}
	}
}

// visitChildren: visit child nodes of a generic node
func (v *PgMutateVisitor) visitChildren(node *pgquery.Node, flag int) {
	if node == nil {
		return
	}

	// Handle list nodes
	if list := node.GetList(); list != nil {
		for _, item := range list.Items {
			v.visitNode(item, flag)
		}
	}

	// Handle other node types that may have nested nodes
	// This is a recursive traversal for safety
}

// Mining functions - add candidates for various mutation types

// miningSelectStmt: add mutation candidates for SELECT statement
func (v *PgMutateVisitor) miningSelectStmt(sel *pgquery.SelectStmt, flag int) {
	// FixMDistinctU_Pg: DISTINCT -> remove DISTINCT
	if len(sel.DistinctClause) > 0 && sel.Op == pgquery.SetOperation_SET_OPERATION_UNDEFINED {
		v.addPgCandidate(FixMDistinctU_Pg, 1, sel.DistinctClause[0], flag)
	}

	// FixMDistinctL_Pg: add DISTINCT (when no DISTINCT and no ORDER BY, no WITH)
	if len(sel.DistinctClause) == 0 && len(sel.SortClause) == 0 && sel.WithClause == nil &&
		sel.Op == pgquery.SetOperation_SET_OPERATION_UNDEFINED {
		v.addPgCandidate(FixMDistinctL_Pg, 0, nil, flag) // Node is nil, will use SelectStmt
	}
}

// miningWhereClause: add mutation candidates for WHERE clause
func (v *PgMutateVisitor) miningWhereClause(sel *pgquery.SelectStmt, flag int) {
	if sel.WhereClause != nil {
		// FixMWhere1U_Pg: WHERE expr -> WHERE TRUE
		v.addPgCandidate(FixMWhere1U_Pg, 1, sel.WhereClause, flag)
		// FixMWhere0L_Pg: WHERE expr -> WHERE FALSE
		v.addPgCandidate(FixMWhere0L_Pg, 0, sel.WhereClause, flag)
	}
}

// miningHavingClause: add mutation candidates for HAVING clause
func (v *PgMutateVisitor) miningHavingClause(sel *pgquery.SelectStmt, flag int) {
	if sel.HavingClause != nil {
		// FixMHaving1U_Pg: HAVING expr -> HAVING TRUE
		v.addPgCandidate(FixMHaving1U_Pg, 1, sel.HavingClause, flag)
		// FixMHaving0L_Pg: HAVING expr -> HAVING FALSE
		v.addPgCandidate(FixMHaving0L_Pg, 0, sel.HavingClause, flag)
	}
}

// miningJoinQuals: add mutation candidates for JOIN ON condition
func (v *PgMutateVisitor) miningJoinQuals(join *pgquery.JoinExpr, flag int) {
	if join.Quals != nil {
		// FixMOn1U_Pg: ON expr -> ON TRUE
		v.addPgCandidate(FixMOn1U_Pg, 1, join.Quals, flag)
		// FixMOn0L_Pg: ON expr -> ON FALSE
		v.addPgCandidate(FixMOn0L_Pg, 0, join.Quals, flag)
	}
}

// miningUnionSelectStmt: add mutation candidates for UNION operations
func (v *PgMutateVisitor) miningUnionSelectStmt(sel *pgquery.SelectStmt, flag int) {
	// UNION operations are represented with Op field
	if sel.Op == pgquery.SetOperation_SETOP_UNION {
		if !sel.All {
			// UNION -> UNION ALL (expanding mutation)
			v.addPgCandidate(FixMUnionAllU_Pg, 1, nil, flag)
		} else {
			// UNION ALL -> UNION (shrinking mutation)
			v.addPgCandidate(FixMUnionAllL_Pg, 0, nil, flag)
		}
	}
}

// miningCmpOp: add mutation candidates for comparison operators
func (v *PgMutateVisitor) miningCmpOp(expr *pgquery.A_Expr, flag int) {
	if expr == nil || len(expr.Name) == 0 {
		return
	}

	// Get the operator name
	opName := v.getAExprOpName(expr)
	if opName == "" {
		return
	}

	// Upper mutations: > -> >=, < -> <=, = -> >=
	switch opName {
	case ">":
		v.addPgCandidate(FixMCmpOpU_Pg, 1, nil, flag) // Node is the A_Expr itself
	case "<":
		v.addPgCandidate(FixMCmpOpU_Pg, 1, nil, flag)
	case "=":
		v.addPgCandidate(FixMCmpOpU_Pg, 1, nil, flag)
	}

	// Lower mutations: >= -> >, <= -> <, != -> <
	switch opName {
	case ">=":
		v.addPgCandidate(FixMCmpOpL_Pg, 0, nil, flag)
	case "<=":
		v.addPgCandidate(FixMCmpOpL_Pg, 0, nil, flag)
	case "<>":
		v.addPgCandidate(FixMCmpOpL_Pg, 0, nil, flag)
	case "!=":
		v.addPgCandidate(FixMCmpOpL_Pg, 0, nil, flag)
	}
}

// miningInClause: add mutation candidates for IN clause
func (v *PgMutateVisitor) miningInClause(expr *pgquery.A_Expr, flag int) {
	if expr == nil || expr.Kind != pgquery.A_Expr_Kind_AEXPR_IN {
		return
	}

	// FixMInNullU_Pg: IN(x,x,x) -> IN(x,x,x,NULL)
	// Note: In pg_query, IN list is represented as a List in Rexpr
	if expr.Rexpr != nil {
		list := expr.Rexpr.GetList()
		if list != nil && len(list.Items) > 0 {
			v.addPgCandidate(FixMInNullU_Pg, 1, nil, flag)
		}
	}
}

// miningLikeExpr: add mutation candidates for LIKE expression
func (v *PgMutateVisitor) miningLikeExpr(expr *pgquery.A_Expr, flag int) {
	if expr == nil {
		return
	}

	// Check if pattern is a constant string
	if expr.Rexpr != nil {
		aConst := expr.Rexpr.GetAConst()
		if aConst != nil {
			// Get the pattern string
			pattern := v.getAConstStringValue(aConst)
			if pattern != "" {
				// Upper mutation: expand pattern
				// Check pattern has non-'%' characters for upper mutation
				hasNonPercent := false
				for _, c := range pattern {
					if c != '%' {
						hasNonPercent = true
						break
					}
				}
				if hasNonPercent {
					v.addPgCandidate(RdMLikePgU, 1, nil, flag)
				}

				// Lower mutation: shrink pattern
				// Check pattern has '%' characters for lower mutation
				hasPercent := strings.Contains(pattern, "%")
				if hasPercent {
					v.addPgCandidate(RdMLikePgL, 0, nil, flag)
				}
			}
		}
	}
}

// miningRegExpExpr: add mutation candidates for REGEXP (~ operator)
func (v *PgMutateVisitor) miningRegExpExpr(expr *pgquery.A_Expr, flag int) {
	if expr == nil {
		return
	}

	// Check if pattern is a constant string
	if expr.Rexpr != nil {
		aConst := expr.Rexpr.GetAConst()
		if aConst != nil {
			pattern := v.getAConstStringValue(aConst)
			if pattern != "" {
				// Upper mutation: expand regex pattern
				// Conditions for upper mutation
				if strings.HasPrefix(pattern, "^") || strings.HasSuffix(pattern, "$") ||
					strings.ContainsAny(pattern, "+?") {
					v.addPgCandidate(RdMRegExpPgU, 1, nil, flag)
				}

				// Lower mutation: shrink regex pattern
				// Conditions for lower mutation
				if strings.Contains(pattern, "*") {
					v.addPgCandidate(RdMRegExpPgL, 0, nil, flag)
				}
			}
		}
	}
}

// Helper functions

// addPgCandidate: add a mutation candidate to the candidates map
func (v *PgMutateVisitor) addPgCandidate(mutationName string, u int, node *pgquery.Node, flag int) {
	if strings.HasSuffix(mutationName, "U") && u == 0 {
		log.Fatal("strings.HasSuffix(mutationName, \"U\") && u == 0")
	}
	if strings.HasSuffix(mutationName, "L") && u != 0 {
		log.Fatal("strings.HasSuffix(mutationName, \"L\") && u != 0")
	}

	var ls []*PgCandidate
	var ok bool
	if ls, ok = v.Candidates[mutationName]; !ok {
		ls = make([]*PgCandidate, 0)
	}
	ls = append(ls, &PgCandidate{
		MutationName: mutationName,
		U:            u,
		Node:         node,
		Flag:         flag,
	})
	v.Candidates[mutationName] = ls
}

// isNotAExpr: check if an A_Expr has NOT prefix
// In pg_query, NOT LIKE is represented with the same A_Expr but operator name includes "!"
func (v *PgMutateVisitor) isNotAExpr(expr *pgquery.A_Expr) bool {
	if expr == nil || len(expr.Name) == 0 {
		return false
	}
	opName := v.getAExprOpName(expr)
	return strings.HasPrefix(opName, "!") || strings.HasPrefix(opName, "NOT")
}

// getAExprOpName: get the operator name from an A_Expr
func (v *PgMutateVisitor) getAExprOpName(expr *pgquery.A_Expr) string {
	if expr == nil || len(expr.Name) == 0 {
		return ""
	}
	// The operator name is stored as a String node in the Name field
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

// getAConstStringValue: get the string value from an A_Const node
func (v *PgMutateVisitor) getAConstStringValue(aConst *pgquery.A_Const) string {
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