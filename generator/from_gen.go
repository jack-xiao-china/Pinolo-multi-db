package generator

import (
	"fmt"
)

// FROM clause and JOIN generation
// Inspired by EET's from_clause/table_ref/joined_table productions and SQLancer's MySQLJoin

// generateFromClause: generate a FROM clause with optional JOINs
// Returns (fromSQL, populated scope with available columns)
func (g *QueryGenerator) generateFromClause(scope *Scope) (string, *Scope) {
	if len(scope.Schema.Tables) == 0 {
		return "", scope
	}

	// Pick 1-3 random tables for the FROM clause
	numTables := g.randInt(1, min(3, len(scope.Schema.Tables)))
	selectedTables := pickRandomN(g.Rand, scope.Schema.Tables, numTables)

	// Create new scope with selected tables
	newScope := NewScope(scope.Schema, scope.Level)

	// First table goes in FROM directly
	firstTable := selectedTables[0]
	firstAlias := g.nextTableAlias()
	newScope.AddTable(firstTable.Name, firstAlias)

	fromParts := []string{fmt.Sprintf("%s AS %s", firstTable.Name, firstAlias)}

	// Remaining tables as JOINs (if enabled and tables available)
	if g.Config.EnableJoin && len(selectedTables) > 1 {
		for i := 1; i < len(selectedTables); i++ {
			table := selectedTables[i]
			alias := g.nextTableAlias()
			newScope.AddTable(table.Name, alias)

			joinType := g.pickJoinType()
			onClause := g.generateJoinOnClause(newScope)
			fromParts = append(fromParts, fmt.Sprintf("%s JOIN %s AS %s ON %s",
				joinType, table.Name, alias, onClause))
		}
	} else if len(selectedTables) > 1 {
		// No JOINs - add remaining tables as comma-separated FROM entries
		for i := 1; i < len(selectedTables); i++ {
			table := selectedTables[i]
			alias := g.nextTableAlias()
			newScope.AddTable(table.Name, alias)
			fromParts = append(fromParts, fmt.Sprintf("%s AS %s", table.Name, alias))
		}
	}

	return joinStrings(fromParts, " "), newScope
}

// pickJoinType: randomly choose a JOIN type
func (g *QueryGenerator) pickJoinType() string {
	choice := g.d6()
	if choice <= 3 {
		return "INNER" // Most common
	}
	if choice <= 5 {
		return "LEFT" // LEFT JOIN (Stage1 converts to INNER later)
	}
	return "RIGHT" // RIGHT JOIN (Stage1 converts to INNER later)
}

// generateJoinOnClause: generate an ON condition for a JOIN
// Uses the columns available in the current scope for type-safe references
func (g *QueryGenerator) generateJoinOnClause(scope *Scope) string {
	// Try to generate a simple equality condition on key columns first
	keyCols := scope.ColumnsOfType("any")
	for i, col1 := range keyCols {
		if col1.IsKey {
			for j, col2 := range keyCols {
				if j != i && col2.IsKey && typeMatch(col2.ColumnType, col1.ColumnType) {
					return fmt.Sprintf("%s.%s = %s.%s", col1.TableAlias, col1.ColumnName, col2.TableAlias, col2.ColumnName)
				}
			}
		}
	}

	// No matching key columns found - generate a boolean expression
	return g.generateBoolExpr(scope, 0)
}

// min: return minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}