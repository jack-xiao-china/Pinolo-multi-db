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
// Both MySQL and PostgreSQL modes use type-safe column-to-column or column-to-constant comparison
// This dramatically reduces execution errors from cross-type comparisons (e.g., int > varchar)
func (g *QueryGenerator) generateComparisonPredicate(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 {
		// Pick a column and compare with a type-compatible value
		col := pickRandom(g.Rand, scope.Columns)
		left := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		right := g.generateTypeCompatibleValue(scope, col.ColumnType, depth)
		op := g.pickComparisonOp()
		return fmt.Sprintf("(%s %s %s)", left, op, right)
	}
	// Fallback: no columns available, use constant comparison
	left := g.generateConstantExpr("int")
	right := g.generateConstantExpr("int")
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
// Always use a column reference directly for type safety
func (g *QueryGenerator) generateIsNullPredicate(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 {
		col := pickRandom(g.Rand, scope.Columns)
		expr := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		if g.randBool() {
			return fmt.Sprintf("%s IS NULL", expr)
		}
		return fmt.Sprintf("%s IS NOT NULL", expr)
	}
	// Fallback: no columns
	if g.randBool() {
		return "(1) IS NULL"
	}
	return "(1) IS NOT NULL"
}

// generateBetweenPredicate: generate BETWEEN predicate
// Both MySQL and PostgreSQL modes use type-safe column BETWEEN with compatible bounds
func (g *QueryGenerator) generateBetweenPredicate(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 {
		// Pick a column and use type-compatible bounds
		col := pickRandom(g.Rand, scope.Columns)
		expr := fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
		low := g.generateTypeCompatibleBound(col.ColumnType)
		high := g.generateTypeCompatibleBound(col.ColumnType)
		not := ""
		if g.randBool() {
			not = "NOT "
		}
		return fmt.Sprintf("(%s) %sBETWEEN %s AND %s", expr, not, low, high)
	}
	// Fallback: no columns, use constants
	expr := g.generateConstantExpr("int")
	low := g.generateConstantExpr("int")
	high := g.generateConstantExpr("int")
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
	case "varchar", "char", "text", "bpchar", "name", "longtext", "mediumtext", "tinytext", "enum":
		return fmt.Sprintf("'str_%d'", g.randInt(0, 50))
	case "date":
		year := 2000 + g.Rand.Intn(25)
		month := 1 + g.Rand.Intn(12)
		day := 1 + g.Rand.Intn(28)
		return fmt.Sprintf("'%04d-%02d-%02d'", year, month, day)
	case "timestamp", "timestamptz", "datetime":
		year := 2000 + g.Rand.Intn(25)
		month := 1 + g.Rand.Intn(12)
		day := 1 + g.Rand.Intn(28)
		hour := g.Rand.Intn(24)
		min := g.Rand.Intn(60)
		sec := g.Rand.Intn(60)
		return fmt.Sprintf("'%04d-%02d-%02d %02d:%02d:%02d'", year, month, day, hour, min, sec)
	case "time":
		hour := g.Rand.Intn(24)
		min := g.Rand.Intn(60)
		sec := g.Rand.Intn(60)
		return fmt.Sprintf("'%02d:%02d:%02d'", hour, min, sec)
	case "year":
		return fmt.Sprintf("%d", 2000+g.Rand.Intn(25))
	case "bit":
		return fmt.Sprintf("b'%d'", g.Rand.Intn(2))
	default:
		return fmt.Sprintf("%d", g.randInt(0, 100))
	}
}

// generateLikePredicate: generate LIKE predicate
// Always use a string column on the left for type safety
func (g *QueryGenerator) generateLikePredicate(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 {
		// Prefer string columns for LIKE
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
		// No string columns, use any column with CAST
		col := pickRandom(g.Rand, scope.Columns)
		expr := fmt.Sprintf("CAST(%s.%s AS CHAR)", col.TableAlias, col.ColumnName)
		pattern := g.generateLikePattern()
		not := ""
		if g.randBool() {
			not = "NOT "
		}
		return fmt.Sprintf("%s %sLIKE %s", expr, not, pattern)
	}
	// Fallback: no columns
	return g.generateBoolLeaf(scope)
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