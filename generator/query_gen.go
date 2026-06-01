package generator

import (
	"fmt"
)

// SELECT query generation - the core of the random SQL generator
// Generates 4 query shapes (plain, UNION, CTE, derived table) like SQLancer EET

// generatePlainSelect: generate a plain SELECT query
// SELECT ... FROM ... [WHERE ...] [GROUP BY ...] [HAVING ...] [ORDER BY ...] [LIMIT ...]
func (g *QueryGenerator) generatePlainSelect() string {
	scope := NewScope(g.Schema, 0)

	// 1. Build FROM clause (populates scope with table/column references)
	fromClause, scope := g.generateFromClause(scope)
	if fromClause == "" {
		return "SELECT 1" // Fallback for empty schema
	}

	// 2. Build SELECT list (1-3 columns with aliases)
	selectList := g.generateSelectList(scope)

	// 3. Build WHERE clause (optional)
	whereClause := ""
	if g.randBool() && scope.NumColumns() > 0 {
		whereClause = g.generateBoolExpr(scope, 0)
	}

	// 4. Build GROUP BY + HAVING (optional)
	groupByClause := ""
	havingClause := ""
	if g.Config.EnableGroupBy && g.randBool() && scope.NumColumns() > 0 {
		groupByClause = g.generateGroupByClause(scope)
		if g.randBool() {
			havingClause = g.generateBoolExpr(scope, 0)
		}
	}

	// 5. Build ORDER BY (optional)
	orderByClause := ""
	if g.Config.EnableOrderBy && g.randBool() && scope.NumColumns() > 0 {
		orderByClause = g.generateOrderByClause(scope)
	}

	// 6. Build LIMIT (optional, requires ORDER BY)
	limitClause := ""
	if g.Config.EnableLimit && orderByClause != "" && g.randBool() {
		limitClause = fmt.Sprintf("LIMIT %d", g.randInt(1, 50))
	}

	// Assemble the query
	sql := "SELECT " + selectList
	sql += " FROM " + fromClause
	if whereClause != "" {
		sql += " WHERE " + whereClause
	}
	if groupByClause != "" {
		sql += " GROUP BY " + groupByClause
	}
	if havingClause != "" {
		sql += " HAVING " + havingClause
	}
	if orderByClause != "" {
		sql += " ORDER BY " + orderByClause
	}
	if limitClause != "" {
		sql += " " + limitClause
	}

	return sql
}

// generateSelectList: generate a SELECT list (1-3 columns with aliases)
func (g *QueryGenerator) generateSelectList(scope *Scope) string {
	numCols := g.randInt(1, min(3, scope.NumColumns()))
	if numCols == 0 {
		numCols = 1
	}
	selectItems := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		alias := g.nextColAlias()
		expr := g.generateExpression(scope, 0, "any")
		selectItems[i] = fmt.Sprintf("%s AS %s", expr, alias)
	}
	return joinStrings(selectItems, ", ")
}

// generateGroupByClause: generate a GROUP BY clause
func (g *QueryGenerator) generateGroupByClause(scope *Scope) string {
	numCols := g.randInt(1, min(3, scope.NumColumns()))
	if numCols == 0 {
		return ""
	}
	colRefs := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		col := pickRandom(g.Rand, scope.Columns)
		colRefs[i] = fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
	}
	return joinStrings(colRefs, ", ")
}

// generateOrderByClause: generate an ORDER BY clause
func (g *QueryGenerator) generateOrderByClause(scope *Scope) string {
	numCols := g.randInt(1, min(2, scope.NumColumns()))
	if numCols == 0 {
		return ""
	}
	orderItems := make([]string, numCols)
	for i := 0; i < numCols; i++ {
		col := pickRandom(g.Rand, scope.Columns)
		direction := ""
		if g.randBool() {
			direction = " ASC"
		} else {
			direction = " DESC"
		}
		orderItems[i] = fmt.Sprintf("%s.%s%s", col.TableAlias, col.ColumnName, direction)
	}
	return joinStrings(orderItems, ", ")
}

// generateUnionSelect: generate a UNION/UNION ALL query
// SELECT ... FROM ... UNION [ALL] SELECT ... FROM ...
func (g *QueryGenerator) generateUnionSelect() string {
	left := g.generatePlainSelect()
	right := g.generatePlainSelect()

	unionType := "UNION ALL"
	if g.randBool() {
		unionType = g.dialectUnionDistinct()
	}

	return fmt.Sprintf("(%s) %s (%s)", left, unionType, right)
}

// generateCTESelect: generate a WITH/CTE query
// WITH cte_name AS (SELECT ...) SELECT ... FROM cte_name ...
func (g *QueryGenerator) generateCTESelect() string {
	cteName := g.nextCTEName()

	// Generate CTE body - a plain SELECT
	cteBody := g.generatePlainSelect()

	// Generate main query using CTE
	scope := NewScope(g.Schema, 0)
	cteAlias := g.nextTableAlias()
	scope.AddTable(cteName, cteAlias)

	// Build SELECT list using CTE columns
	if scope.NumColumns() == 0 {
		// If CTE has no discoverable columns, use a simple reference
		return fmt.Sprintf("WITH %s AS (%s) SELECT * FROM %s", cteName, cteBody, cteName)
	}

	selectList := g.generateSelectList(scope)
	whereClause := ""
	if g.randBool() && scope.NumColumns() > 0 {
		whereClause = " WHERE " + g.generateBoolExpr(scope, 0)
	}

	return fmt.Sprintf("WITH %s AS (%s) SELECT %s FROM %s AS %s%s",
		cteName, cteBody, selectList, cteName, cteAlias, whereClause)
}

// generateDerivedSelect: generate a query with a derived table (subquery in FROM)
// SELECT ... FROM (SELECT ...) AS alias ...
func (g *QueryGenerator) generateDerivedSelect() string {
	// Generate the subquery
	subquery := g.generatePlainSelect()

	// Build main query using derived table
	scope := NewScope(g.Schema, 0)
	derivedAlias := g.nextTableAlias()

	// For derived tables, we can't easily discover column metadata
	// So we use SELECT * from the derived table
	selectList := "*"
	whereClause := ""
	if g.randBool() {
		whereClause = " WHERE " + g.generateBoolExpr(scope, 0)
	}

	sql := fmt.Sprintf("SELECT %s FROM (%s) AS %s", selectList, subquery, derivedAlias)
	if whereClause != "" {
		sql += whereClause
	}
	return sql
}