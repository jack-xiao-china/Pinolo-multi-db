package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL EET EXISTS <-> IN transformation mutations

// EXISTS -> IN transformation:
// EXISTS(subquery) -> CASE WHEN (1 IN subquery) IS NOT NULL THEN (1 IN subquery) ELSE FALSE END
// In PG AST: SubLink with subLinkType=EXISTS_SUBLINK -> SubLink with subLinkType=ANY_SUBLINK (IN)

// IN -> EXISTS transformation:
// lhs IN (subquery) -> CASE WHEN (lhs IN subquery) IS NOT NULL THEN EXISTS(subquery) ELSE FALSE END
// In PG AST: SubLink with subLinkType=ANY_SUBLINK -> SubLink with subLinkType=EXISTS_SUBLINK

// addFixMExistsToIn_Pg: FixMExistsToIn_Pg, SubLink(EXISTS): EXISTS(subq) -> IN equivalent
func (v *PgMutateVisitor) addFixMExistsToIn_Pg(sublink *pgquery.SubLink, flag int) {
	if sublink != nil && sublink.SubLinkType == pgquery.SubLinkType_EXISTS_SUBLINK && sublink.Subselect != nil {
		v.addPgCandidate(FixMExistsToIn_Pg, 1, nil, flag)
	}
}

// addFixMInToExists_Pg: FixMInToExists_Pg, SubLink(ANY/IN): lhs IN (subq) -> EXISTS equivalent
func (v *PgMutateVisitor) addFixMInToExists_Pg(sublink *pgquery.SubLink, flag int) {
	if sublink != nil && sublink.SubLinkType == pgquery.SubLinkType_ANY_SUBLINK && sublink.Subselect != nil && sublink.Testexpr != nil {
		v.addPgCandidate(FixMInToExists_Pg, 1, nil, flag)
	}
}

// doFixMExistsToIn_Pg: FixMExistsToIn_Pg, EXISTS(subq) -> CASE WHEN (1 IN subq) IS NOT NULL THEN (1 IN subq) ELSE FALSE END
func doFixMExistsToIn_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMExistsToIn_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find EXISTS SubLink in WHERE clause
		sublinkNode := findSubLinkInWhere(sel.WhereClause, pgquery.SubLinkType_EXISTS_SUBLINK)
		if sublinkNode == nil {
			continue
		}

		sublink := sublinkNode.GetSubLink()
		if sublink == nil {
			continue
		}

		oldWhere := sel.WhereClause

		// Build: 1 IN (subquery) as ANY_SUBLINK
		inSublink := &pgquery.SubLink{
			SubLinkType: pgquery.SubLinkType_ANY_SUBLINK,
			Subselect:   sublink.Subselect,
			Testexpr:    pgquery.MakeAConstIntNode(1, 0), // lhs = 1
			Location:    0,
		}
		inSublinkNode := &pgquery.Node{
			Node: &pgquery.Node_SubLink{
				SubLink: inSublink,
			},
		}

		// Build CASE WHEN (1 IN subq) IS NOT NULL THEN (1 IN subq) ELSE FALSE END
		isNotNull := &pgquery.Node{
			Node: &pgquery.Node_NullTest{
				NullTest: &pgquery.NullTest{
					Arg:          inSublinkNode,
					Nulltesttype: pgquery.NullTestType_IS_NOT_NULL,
					Location:     0,
				},
			},
		}

		whenClause := pgquery.MakeCaseWhenNode(isNotNull, inSublinkNode, 0)

		caseExpr := &pgquery.CaseExpr{
			Args:      []*pgquery.Node{whenClause},
			Defresult: pgquery.MakeAConstIntNode(0, 0), // ELSE FALSE
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
			return "", errors.Wrap(err, "[doFixMExistsToIn_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMExistsToIn_Pg]no EXISTS SubLink found")
}

// doFixMInToExists_Pg: FixMInToExists_Pg, lhs IN (subq) -> CASE WHEN (lhs IN subq) IS NOT NULL THEN EXISTS(subq) ELSE FALSE END
func doFixMInToExists_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMInToExists_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find ANY_SUBLINK (IN) in WHERE clause
		sublinkNode := findSubLinkInWhere(sel.WhereClause, pgquery.SubLinkType_ANY_SUBLINK)
		if sublinkNode == nil {
			continue
		}

		sublink := sublinkNode.GetSubLink()
		if sublink == nil {
			continue
		}

		oldWhere := sel.WhereClause
		originalSubLinkType := sublink.SubLinkType
		originalTestexpr := sublink.Testexpr

		// Build EXISTS equivalent: EXISTS(subquery)
		existSublink := &pgquery.SubLink{
			SubLinkType: pgquery.SubLinkType_EXISTS_SUBLINK,
			Subselect:   sublink.Subselect,
			Location:    0,
		}
		existSublinkNode := &pgquery.Node{
			Node: &pgquery.Node_SubLink{
				SubLink: existSublink,
			},
		}

		// Build CASE WHEN (lhs IN subq) IS NOT NULL THEN EXISTS(subq) ELSE FALSE END
		// Restore IN expression first
		sublink.SubLinkType = originalSubLinkType
		sublink.Testexpr = originalTestexpr

		isNotNull := &pgquery.Node{
			Node: &pgquery.Node_NullTest{
				NullTest: &pgquery.NullTest{
					Arg:          sublinkNode, // original IN expression
					Nulltesttype: pgquery.NullTestType_IS_NOT_NULL,
					Location:     0,
				},
			},
		}

		whenClause := pgquery.MakeCaseWhenNode(isNotNull, existSublinkNode, 0)

		caseExpr := &pgquery.CaseExpr{
			Args:      []*pgquery.Node{whenClause},
			Defresult: pgquery.MakeAConstIntNode(0, 0), // ELSE FALSE
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
			return "", errors.Wrap(err, "[doFixMInToExists_Pg]deparse error")
		}

		sel.WhereClause = oldWhere
		return sql, nil
	}

	return "", errors.New("[doFixMInToExists_Pg]no IN SubLink found")
}

// findSubLinkInWhere: recursively search for a SubLink with specific type in WHERE clause
func findSubLinkInWhere(node *pgquery.Node, targetType pgquery.SubLinkType) *pgquery.Node {
	if node == nil {
		return nil
	}

	// Check if this node is a SubLink with the target type
	sublink := node.GetSubLink()
	if sublink != nil && sublink.SubLinkType == targetType {
		return node
	}

	// Search in BoolExpr args
	boolExpr := node.GetBoolExpr()
	if boolExpr != nil {
		for _, arg := range boolExpr.Args {
			result := findSubLinkInWhere(arg, targetType)
			if result != nil {
				return result
			}
		}
	}

	// Search in A_Expr
	aExpr := node.GetAExpr()
	if aExpr != nil {
		if aExpr.Lexpr != nil {
			result := findSubLinkInWhere(aExpr.Lexpr, targetType)
			if result != nil {
				return result
			}
		}
		if aExpr.Rexpr != nil {
			result := findSubLinkInWhere(aExpr.Rexpr, targetType)
			if result != nil {
				return result
			}
		}
	}

	return nil
}