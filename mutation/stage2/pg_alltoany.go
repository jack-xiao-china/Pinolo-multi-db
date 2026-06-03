package stage2

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
)

// PostgreSQL EET ALL <-> ANY cross-quantifier implication mutations

// ALL -> ANY transformation:
// x > ALL(subq) -> x > ANY(subq) / x > SOME(subq)
// Implication (upper): ALL result ⊆ ANY result (satisfying ALL values ⊆ satisfying SOME value)
// Warning: NULL boundary may break containment. Accept false positive risk.

// ANY -> ALL transformation:
// x > ANY(subq) -> x > ALL(subq)
// Implication (lower): ANY result ⊇ ALL result (satisfying SOME value ⊇ satisfying ALL values)
// Warning: NULL boundary may break containment. Accept false positive risk.

// addFixMAllToAnyU_Pg: FixMAllToAnyU_Pg, SubLink(ALL): ALL(subq) -> ANY(subq) (upper mutation)
func (v *PgMutateVisitor) addFixMAllToAnyU_Pg(sublink *pgquery.SubLink, flag int) {
	if sublink != nil && sublink.SubLinkType == pgquery.SubLinkType_ALL_SUBLINK && sublink.Subselect != nil && sublink.Testexpr != nil {
		v.addPgCandidate(FixMAllToAnyU_Pg, 1, nil, flag)
	}
}

// addFixMAnyToAllL_Pg: FixMAnyToAllL_Pg, SubLink(ANY/SOME): ANY(subq) -> ALL(subq) (lower mutation)
func (v *PgMutateVisitor) addFixMAnyToAllL_Pg(sublink *pgquery.SubLink, flag int) {
	if sublink != nil && sublink.SubLinkType == pgquery.SubLinkType_ANY_SUBLINK && sublink.Subselect != nil && sublink.Testexpr != nil {
		v.addPgCandidate(FixMAnyToAllL_Pg, 1, nil, flag)
	}
}

// doFixMAllToAnyU_Pg: FixMAllToAnyU_Pg, x > ALL(subq) -> x > ANY(subq) (upper mutation)
// Simply change SubLinkType from ALL_SUBLINK to ANY_SUBLINK
func doFixMAllToAnyU_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMAllToAnyU_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find ALL SubLink in WHERE clause
		sublinkNode := findSubLinkInWhere(sel.WhereClause, pgquery.SubLinkType_ALL_SUBLINK)
		if sublinkNode == nil {
			continue
		}

		sublink := sublinkNode.GetSubLink()
		if sublink == nil {
			continue
		}

		oldWhere := sel.WhereClause
		originalSubLinkType := sublink.SubLinkType

		// Change ALL -> ANY
		sublink.SubLinkType = pgquery.SubLinkType_ANY_SUBLINK

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			sublink.SubLinkType = originalSubLinkType
			return "", errors.Wrap(err, "[doFixMAllToAnyU_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		sublink.SubLinkType = originalSubLinkType

		return sql, nil
	}

	return "", errors.New("[doFixMAllToAnyU_Pg]no ALL SubLink found")
}

// doFixMAnyToAllL_Pg: FixMAnyToAllL_Pg, x > ANY(subq) -> x > ALL(subq) (lower mutation)
// Simply change SubLinkType from ANY_SUBLINK to ALL_SUBLINK
func doFixMAnyToAllL_Pg(rootNode *pgquery.ParseResult, node *pgquery.Node, seed int64) (string, error) {
	if rootNode == nil || len(rootNode.Stmts) == 0 {
		return "", errors.New("[doFixMAnyToAllL_Pg]rootNode == nil || len(rootNode.Stmts) == 0")
	}

	for _, rawStmt := range rootNode.Stmts {
		if rawStmt == nil || rawStmt.Stmt == nil {
			continue
		}
		sel := rawStmt.Stmt.GetSelectStmt()
		if sel == nil || sel.WhereClause == nil {
			continue
		}

		// Find ANY SubLink in WHERE clause
		// Note: ANY_SUBLINK includes IN-style subqueries, but we only want operator-style ANY (x > ANY)
		// For IN-style (lhs IN subquery), the mining function addFixMInToExists_Pg handles that separately
		// We filter by checking that the SubLink has a Testexpr (lhs) which is the comparison operand
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

		// Change ANY -> ALL
		sublink.SubLinkType = pgquery.SubLinkType_ALL_SUBLINK

		sql, err := pgquery.Deparse(rootNode)
		if err != nil {
			sel.WhereClause = oldWhere
			sublink.SubLinkType = originalSubLinkType
			return "", errors.Wrap(err, "[doFixMAnyToAllL_Pg]deparse error")
		}

		// Restore original
		sel.WhereClause = oldWhere
		sublink.SubLinkType = originalSubLinkType

		return sql, nil
	}

	return "", errors.New("[doFixMAnyToAllL_Pg]no ANY SubLink found")
}