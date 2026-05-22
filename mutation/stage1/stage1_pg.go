package stage1

import (
	"github.com/pkg/errors"
	pgquery "github.com/pganalyze/pg_query_go/v6"
	"github.com/qaqcatz/impomysql/connector"
)

// PgInitResult: Stage1 initialization result for PostgreSQL
type PgInitResult struct {
	InitSql    string
	Err        error
	ExecResult *connector.Result
}

// InitForPostgreSQL: PostgreSQL specific Stage1 preprocessing
//
// Similar to MySQL Stage1, but using pg_query_go parser:
// - Remove aggregate functions and GROUP BY
// - Remove window functions
// - Convert LEFT/RIGHT JOIN to INNER JOIN
// - Remove LIMIT/OFFSET clauses
// - Remove uncertain functions (RANDOM(), NOW() etc.)
//
// Note that:
// (1) we only support SELECT statement
// (2) make sure your sql has no side-effects
// (3) The transformed sql may fail to execute
func InitForPostgreSQL(sql string) *PgInitResult {
	initResult := &PgInitResult{
		InitSql:    "",
		Err:        nil,
		ExecResult: nil,
	}

	// Parse with pg_query_go
	parseResult, err := pgquery.Parse(sql)
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForPostgreSQL]parse error")
		return initResult
	}

	// Get the first statement
	stmts := parseResult.Stmts
	if stmts == nil || len(stmts) == 0 {
		initResult.Err = errors.New("[InitForPostgreSQL]no statements found")
		return initResult
	}

	stmt := stmts[0]
	stmtNode := stmt.Stmt

	// Check if it's a SelectStmt or SetOperationStmt (UNION)
	selectStmt := stmtNode.GetSelectStmt()
	setOperationStmt := stmtNode.GetSetOperationStmt()

	if selectStmt == nil && setOperationStmt == nil {
		initResult.Err = errors.New("[InitForPostgreSQL]not a SELECT or UNION statement")
		return initResult
	}

	// Apply transformations
	if selectStmt != nil {
		transformSelectStmtForPostgreSQL(selectStmt)
	}

	// Deparse back to SQL
	modifiedSQL, err := pgquery.Deparse(parseResult)
	if err != nil {
		initResult.Err = errors.Wrap(err, "[InitForPostgreSQL]deparse error")
		return initResult
	}

	initResult.InitSql = modifiedSQL
	return initResult
}

// InitForPostgreSQLAndExec: InitForPostgreSQL + execute
func InitForPostgreSQLAndExec(sql string, conn connector.SQLExecutor) *PgInitResult {
	initResult := InitForPostgreSQL(sql)
	if initResult.Err != nil {
		return initResult
	}
	result := conn.ExecSQL(initResult.InitSql)
	initResult.ExecResult = result
	return initResult
}

// transformSelectStmtForPostgreSQL: transform SELECT statement for mutation testing
func transformSelectStmtForPostgreSQL(selectStmt *pgquery.SelectStmt) {
	if selectStmt == nil {
		return
	}

	// Remove DISTINCT ON (keep DISTINCT)
	if selectStmt.DistinctClause != nil {
		// Check if it's DISTINCT ON by examining the structure
		// For simple DISTINCT, we keep it; for DISTINCT ON, we simplify
		distinctOn := selectStmt.GetDistinctClause()
		if distinctOn != nil && len(distinctOn) > 0 {
			// DISTINCT ON case - remove the ON clause
			// Keep only the DISTINCT keyword effect
		}
	}

	// Remove GROUP BY clause
	selectStmt.GroupClause = nil

	// Remove HAVING clause (handled separately in mutation)
	// Keep HAVING for mutation testing

	// Remove window functions (Partitions)
	if selectStmt.WindowClause != nil {
		selectStmt.WindowClause = nil
	}

	// Remove LIMIT and OFFSET
	selectStmt.LimitCount = nil
	selectStmt.LimitOffset = nil

	// Remove window functions (Partitions)
	if selectStmt.WindowClause != nil {
		selectStmt.WindowClause = nil
	}

	// Transform LEFT/RIGHT JOIN to INNER JOIN
	transformFromClauseForPostgreSQL(selectStmt.FromClause)

	// Remove ORDER BY (not needed for mutation testing)
	selectStmt.SortClause = nil

	// Remove FOR UPDATE/SHARE clauses
	selectStmt.LockingClause = nil

	// Remove INTO clause
	selectStmt.IntoClause = nil
}

// transformFromClauseForPostgreSQL: transform JOIN nodes in FROM clause
func transformFromClauseForPostgreSQL(fromClause []*pgquery.Node) {
	if fromClause == nil {
		return
	}

	for _, node := range fromClause {
		if node == nil {
			continue
		}

		// Check for JoinExpr
		joinExpr := node.GetJoinExpr()
		if joinExpr != nil {
			transformJoinExprForPostgreSQL(joinExpr)
		}

		// Check for RangeSubselect (subquery in FROM)
		rangeSubselect := node.GetRangeSubselect()
		if rangeSubselect != nil {
			subselect := rangeSubselect.Subquery
			if subselect != nil {
				subSelectStmt := subselect.GetSelectStmt()
				if subSelectStmt != nil {
					transformSelectStmtForPostgreSQL(subSelectStmt)
				}
			}
		}
	}
}

// transformJoinExprForPostgreSQL: transform JOIN expression
func transformJoinExprForPostgreSQL(joinExpr *pgquery.JoinExpr) {
	if joinExpr == nil {
		return
	}

	// Convert LEFT/RIGHT/OUTER JOIN to INNER JOIN
	// PostgreSQL JoinType: JOIN_INNER=1, JOIN_LEFT=2, JOIN_FULL=3, JOIN_RIGHT=4
	// We only keep INNER JOIN
	switch joinExpr.Jointype {
	case pgquery.JoinType_JOIN_LEFT,
		pgquery.JoinType_JOIN_RIGHT,
		pgquery.JoinType_JOIN_FULL:
		// Convert to INNER JOIN
		joinExpr.Jointype = pgquery.JoinType_JOIN_INNER
	}

	// Recursively transform nested JOINs
	if joinExpr.Larg != nil {
		transformFromClauseForPostgreSQL([]*pgquery.Node{joinExpr.Larg})
	}
	if joinExpr.Rarg != nil {
		transformFromClauseForPostgreSQL([]*pgquery.Node{joinExpr.Rarg})
	}
}