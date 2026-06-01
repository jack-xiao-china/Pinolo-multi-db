package generator

import (
	"fmt"
)

// Boolean expression generation for WHERE/HAVING/ON conditions
// Inspired by EET's bool_expr::factory() and SQLancer's generateBooleanExpression()
// PostgreSQL strict type checking: comparisons must be type-compatible

// generateBoolExpr: generate a random boolean expression
func (g *QueryGenerator) generateBoolExpr(scope *Scope, depth int) string {
	if depth >= g.Config.MaxDepth || scope.NumColumns() == 0 {
		return g.generateBoolLeaf(scope)
	}

	choice := g.d12()
	if choice <= 3 {
		return g.generateComparisonPredicate(scope, depth)
	}
	if choice <= 6 {
		return g.generateBoolBinaryOp(scope, depth)
	}
	if choice <= 8 {
		return g.generateIsNullPredicate(scope, depth)
	}
	if choice <= 10 {
		return g.generateBetweenPredicate(scope, depth)
	}
	if choice <= 11 {
		return g.generateLikePredicate(scope, depth)
	}
	return g.generateBoolLeaf(scope)
}

// generateBoolLeaf: generate a simple boolean leaf (comparison or constant)
func (g *QueryGenerator) generateBoolLeaf(scope *Scope) string {
	choice := g.d6()
	if choice <= 4 && scope.NumColumns() > 0 {
		return g.generateComparisonPredicate(scope, g.Config.MaxDepth)
	}
	// Constant boolean
	if g.isMySQLDialect() {
		if g.randBool() {
			return "1"
		}
		return "0"
	}
	// PostgreSQL uses TRUE/FALSE literals
	if g.randBool() {
		return "TRUE"
	}
	return "FALSE"
}

// generateComparisonPredicate: generate a comparison predicate
// PostgreSQL mode uses type-safe column-to-column or column-to-constant comparison
func (g *QueryGenerator) generateComparisonPredicate(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 && g.isPostgreSQLDialect() {
		// PostgreSQL strict type checking: pick a column and compare with compatible value
		col := pickRandom(g.Rand, scope.Columns)
		left := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		right := g.generateTypeCompatibleValue(scope, col.ColumnType, depth)
		op := g.pickComparisonOp()
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	}
	// MySQL mode: relaxed type coercion allows mixed types
	left := g.generateExpression(scope, depth+1, "any")
	right := g.generateExpression(scope, depth+1, "any")
	op := g.pickComparisonOp()
	return fmt.Sprintf("(%s %s %s)", left, op, right)
}

// generateBoolBinaryOp: generate AND/OR boolean expression
func (g *QueryGenerator) generateBoolBinaryOp(scope *Scope, depth int) string {
	left := g.generateBoolExpr(scope, depth+1)
	right := g.generateBoolExpr(scope, depth+1)
	op := g.pickBoolBinaryOp()
	return fmt.Sprintf("(%s %s %s)", left, op, right)
}

// pickBoolBinaryOp: randomly choose AND or OR
func (g *QueryGenerator) pickBoolBinaryOp() string {
	if g.randBool() {
		return "AND"
	}
	return "OR"
}

// generateIsNullPredicate: generate IS NULL / IS NOT NULL predicate
// Type-safe: use a column reference directly
func (g *QueryGenerator) generateIsNullPredicate(scope *Scope, depth int) string {
	if g.isPostgreSQLDialect() && scope.NumColumns() > 0 {
		// Use column reference directly for type safety
		col := pickRandom(g.Rand, scope.Columns)
		expr := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		if g.randBool() {
			return fmt.Sprintf("%s IS NULL", expr)
		}
		return fmt.Sprintf("%s IS NOT NULL", expr)
	}
	expr := g.generateExpression(scope, depth+1, "any")
	if g.randBool() {
		return fmt.Sprintf("(%s) IS NULL", expr)
	}
	return fmt.Sprintf("(%s) IS NOT NULL", expr)
}

// generateBetweenPredicate: generate BETWEEN predicate
// PostgreSQL mode uses type-safe column BETWEEN with compatible bounds
func (g *QueryGenerator) generateBetweenPredicate(scope *Scope, depth int) string {
	if g.isPostgreSQLDialect() && scope.NumColumns() > 0 {
		// PostgreSQL strict: column BETWEEN compatible_constant AND compatible_constant
		col := pickRandom(g.Rand, scope.Columns)
		expr := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		low := g.generateTypeCompatibleBound(col.ColumnType)
		high := g.generateTypeCompatibleBound(col.ColumnType)
		not := ""
		if g.randBool() {
			not = "NOT "
		}
		return fmt.Sprintf("%s %sBETWEEN %s AND %s", expr, not, low, high)
	}
	// MySQL mode: relaxed type coercion
	expr := g.generateExpression(scope, depth+1, "any")
	low := g.generateLeaf(scope, "any")
	high := g.generateLeaf(scope, "any")
	not := ""
	if g.randBool() {
		not = "NOT "
	}
	return fmt.Sprintf("(%s) %sBETWEEN %s AND %s", expr, not, low, high)
}

// generateTypeCompatibleBound: generate a BETWEEN bound value compatible with a column type
func (g *QueryGenerator) generateTypeCompatibleBound(colType string) string {
	normalizedType := normalizeType(colType)
	switch normalizedType {
	case "int", "bigint", "smallint", "tinyint", "mediumint", "int4", "int8", "int2":
		return fmt.Sprintf("%d", g.randInt(0, 100))
	case "float", "double", "decimal", "numeric", "float4", "float8", "double precision":
		return fmt.Sprintf("%.2f", float64(g.randInt(0, 100))/10.0)
	case "varchar", "char", "text", "bpchar", "name":
		return fmt.Sprintf("'str_%d'", g.randInt(0, 50))
	case "date":
		return "'2020-01-01'"
	case "timestamp", "timestamptz", "datetime":
		return "'2020-01-01 12:00:00'"
	default:
		return fmt.Sprintf("%d", g.randInt(0, 100))
	}
}

// generateLikePredicate: generate LIKE predicate
// Always use a string column on the left for type safety
func (g *QueryGenerator) generateLikePredicate(scope *Scope, depth int) string {
	if g.isPostgreSQLDialect() && scope.NumColumns() > 0 {
		// Use a string column directly for LIKE (PostgreSQL requires string type)
		strCols := scope.ColumnsOfType("string")
		if len(strCols) > 0 {
			col := pickRandom(g.Rand, strCols)
			expr := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
			pattern := g.generateLikePattern()
			not := ""
			if g.randBool() {
				not = "NOT "
			}
			return fmt.Sprintf("%s %sLIKE %s", expr, not, pattern)
		}
		// No string columns, skip LIKE
		return g.generateBoolLeaf(scope)
	}
	expr := g.generateExpression(scope, depth+1, "string")
	pattern := g.generateLikePattern()
	not := ""
	if g.randBool() {
		not = "NOT "
	}
	return fmt.Sprintf("(%s) %sLIKE %s", expr, not, pattern)
}

// generateLikePattern: generate a random LIKE pattern string
func (g *QueryGenerator) generateLikePattern() string {
	patterns := []string{
		"'%%'",
		"'str_%'",
		"'%_str'",
		"'s__r%'",
	}
	return pickRandom(g.Rand, patterns)
}