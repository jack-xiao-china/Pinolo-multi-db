package generator

import (
	"fmt"
)

// Expression generation inspired by SQLancer's ExpressionGenerator (Action enum + depth control)
// and EET's value_expr::factory() (dice-based random choice with type constraints)
// Supports MySQL and PostgreSQL dialects via dialect helper methods

// generateExpression: generate a random SQL expression with depth control
// typeConstraint: desired result type ("" or "any" = any type)
func (g *QueryGenerator) generateExpression(scope *Scope, depth int, typeConstraint string) string {
	// Leaf nodes when max depth reached or scope has no columns
	if depth >= g.Config.MaxDepth || scope.NumColumns() == 0 {
		return g.generateLeaf(scope, typeConstraint)
	}

	choice := g.d20()
	if choice <= 3 {
		return g.generateColumnRefExpr(scope, typeConstraint)
	}
	if choice <= 6 {
		return g.generateConstantExpr(typeConstraint)
	}
	if choice <= 10 {
		return g.generateComparisonExpr(scope, depth)
	}
	if choice <= 14 {
		return g.generateBinaryArithExpr(scope, depth, typeConstraint)
	}
	if choice <= 17 {
		return g.generateFunctionCallExpr(scope, depth, typeConstraint)
	}
	if choice <= 19 {
		return g.generateCaseExpr(scope, depth, typeConstraint)
	}
	return g.generateInExpr(scope, depth)
}

// generateLeaf: generate a leaf expression (column reference or constant)
func (g *QueryGenerator) generateLeaf(scope *Scope, typeConstraint string) string {
	if g.randBool() && scope.NumColumns() > 0 {
		return g.generateColumnRefExpr(scope, typeConstraint)
	}
	return g.generateConstantExpr(typeConstraint)
}

// generateColumnRefExpr: generate a column reference expression (e.g., "t0.col_int")
func (g *QueryGenerator) generateColumnRefExpr(scope *Scope, typeConstraint string) string {
	cols := scope.ColumnsOfType(typeConstraint)
	if len(cols) == 0 {
		cols = scope.Columns
	}
	if len(cols) == 0 {
		return g.generateConstantExpr(typeConstraint)
	}
	col := pickRandom(g.Rand, cols)
	return fmt.Sprintf("%s.%s", col.TableAlias, col.ColumnName)
}

// generateConstantExpr: generate a constant/literal expression
func (g *QueryGenerator) generateConstantExpr(typeConstraint string) string {
	switch typeConstraint {
	case "int", "bigint", "smallint", "tinyint", "mediumint", "int4", "int8", "int2":
		return fmt.Sprintf("%d", g.randInt(-100, 100))
	case "float", "double", "decimal", "numeric", "float4", "float8", "double precision":
		return fmt.Sprintf("%.2f", float64(g.randInt(-100, 100))/10.0)
	case "varchar", "char", "text", "bpchar", "varchar(20)", "varchar(50)", "name":
		return fmt.Sprintf("'str_%d'", g.randInt(0, 999))
	case "bool", "boolean":
		if g.isMySQLDialect() {
			if g.randBool() {
				return "1"
			}
			return "0"
		}
		if g.randBool() {
			return "TRUE"
		}
		return "FALSE"
	default:
		choice := g.d6()
		if choice <= 2 {
			return fmt.Sprintf("%d", g.randInt(-100, 100))
		}
		if choice <= 4 {
			return fmt.Sprintf("%.2f", float64(g.randInt(-100, 100))/10.0)
		}
		return fmt.Sprintf("'str_%d'", g.randInt(0, 999))
	}
}

// generateComparisonExpr: generate a type-safe comparison expression
// For PostgreSQL/GaussDB-A strict type checking, both sides must be type-compatible
func (g *QueryGenerator) generateComparisonExpr(scope *Scope, depth int) string {
	if scope.NumColumns() > 0 && g.isPostgreSQLDialect() {
		// PostgreSQL strict mode: pick a column and compare with a compatible expression
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

// generateTypeCompatibleValue: generate a value that is type-compatible with a given column type
// This ensures PostgreSQL doesn't reject comparisons due to type mismatches
func (g *QueryGenerator) generateTypeCompatibleValue(scope *Scope, colType string, depth int) string {
	normalizedType := normalizeType(colType)
	switch normalizedType {
	case "int", "bigint", "smallint", "tinyint", "mediumint", "int4", "int8", "int2":
		// Numeric column: compare with another numeric column or integer constant
		if g.randBool() {
			intCols := scope.ColumnsOfType("int")
			if len(intCols) > 0 {
				c := pickRandom(g.Rand, intCols)
				return fmt.Sprintf("%s.%s", c.TableAlias, c.ColumnName)
			}
		}
		return fmt.Sprintf("%d", g.randInt(-100, 100))
	case "float", "double", "decimal", "numeric", "float4", "float8", "double precision":
		if g.randBool() {
			floatCols := scope.ColumnsOfType("float")
			if len(floatCols) > 0 {
				c := pickRandom(g.Rand, floatCols)
				return fmt.Sprintf("%s.%s", c.TableAlias, c.ColumnName)
			}
		}
		return fmt.Sprintf("%.2f", float64(g.randInt(-100, 100))/10.0)
	case "varchar", "char", "text", "bpchar", "name":
		if g.randBool() {
			strCols := scope.ColumnsOfType("string")
			if len(strCols) > 0 {
				c := pickRandom(g.Rand, strCols)
				return fmt.Sprintf("%s.%s", c.TableAlias, c.ColumnName)
			}
		}
		return fmt.Sprintf("'str_%d'", g.randInt(0, 999))
	default:
		return g.generateExpression(scope, depth+1, "any")
	}
}

// pickComparisonOp: randomly choose a comparison operator
func (g *QueryGenerator) pickComparisonOp() string {
	ops := []string{"=", "<>", "<", "<=", ">", ">="}
	return pickRandom(g.Rand, ops)
}

// generateBinaryArithExpr: generate an arithmetic binary operation
func (g *QueryGenerator) generateBinaryArithExpr(scope *Scope, depth int, typeConstraint string) string {
	left := g.generateExpression(scope, depth+1, typeConstraint)
	right := g.generateExpression(scope, depth+1, typeConstraint)
	op := g.pickArithOp()
	return fmt.Sprintf("(%s %s %s)", left, op, right)
}

// pickArithOp: randomly choose an arithmetic operator
func (g *QueryGenerator) pickArithOp() string {
	ops := []string{"+", "-", "*", "/", "%"}
	return pickRandom(g.Rand, ops)
}

// generateFunctionCallExpr: generate a function call expression
// Uses dialect-specific function selection and syntax
func (g *QueryGenerator) generateFunctionCallExpr(scope *Scope, depth int, typeConstraint string) string {
	funcName := g.pickFunction(typeConstraint)

	// CAST has special syntax: CAST(expr AS type)
	if funcName == "CAST" {
		expr := g.generateExpression(scope, depth+1, "any")
		castType := g.pickCastTargetType()
		return fmt.Sprintf("CAST(%s AS %s)", expr, g.dialectCastType(castType))
	}

	// IF → CASE WHEN for PostgreSQL dialect; IF(condition, then, else) for MySQL
	if funcName == "IF" {
		condition := g.generateBoolExpr(scope, depth+1)
		thenVal := g.generateExpression(scope, depth+1, typeConstraint)
		elseVal := g.generateExpression(scope, depth+1, typeConstraint)
		return g.dialectFunctionIF(condition, thenVal, elseVal)
	}

	// IFNULL → COALESCE for PostgreSQL dialect; IFNULL(expr) for MySQL
	if funcName == "IFNULL" {
		expr := g.generateExpression(scope, depth+1, typeConstraint)
		return g.dialectFunctionIFNULL(expr)
	}

	// COALESCE: 2-3 arguments (same syntax across dialects)
	if funcName == "COALESCE" {
		numArgs := g.randInt(2, 3)
		args := make([]string, numArgs)
		for i := 0; i < numArgs; i++ {
			args[i] = g.generateExpression(scope, depth+1, typeConstraint)
		}
		return fmt.Sprintf("COALESCE(%s)", joinStrings(args, ", "))
	}

	// NULLIF: same syntax across dialects
	if funcName == "NULLIF" {
		left := g.generateExpression(scope, depth+1, typeConstraint)
		right := g.generateExpression(scope, depth+1, typeConstraint)
		return fmt.Sprintf("NULLIF(%s, %s)", left, right)
	}

	// Standard functions (ABS, FLOOR, etc. — same syntax across dialects)
	numArgs := functionArgCount(funcName)
	args := make([]string, numArgs)
	for i := 0; i < numArgs; i++ {
		args[i] = g.generateExpression(scope, depth+1, "any")
	}
	return fmt.Sprintf("%s(%s)", funcName, joinStrings(args, ", "))
}

// pickCastTargetType: randomly choose a CAST target type (MySQL-style, dialectCastType converts)
func (g *QueryGenerator) pickCastTargetType() string {
	types := []string{"SIGNED", "UNSIGNED", "DOUBLE", "DECIMAL(40,20)", "CHAR(20)", "VARCHAR(20)", "DATE", "DATETIME"}
	return pickRandom(g.Rand, types)
}

// pickFunction: randomly choose a SQL function name
// MySQL dialect includes IF/IFNULL; PostgreSQL dialect uses COALESCE/NULLIF only
func (g *QueryGenerator) pickFunction(typeConstraint string) string {
	intFuncs := []string{"ABS", "CEILING", "FLOOR", "ROUND", "SIGN"}
	floatFuncs := []string{"ABS", "ACOS", "ASIN", "ATAN", "COS", "EXP", "LN", "LOG", "LOG2", "SQRT"}
	stringFuncs := []string{"CONCAT", "LENGTH", "LOWER", "UPPER", "TRIM", "LEFT", "RIGHT", "SUBSTRING", "REPLACE", "REVERSE"}

	// Dialect-specific function pools
	var anyFuncs []string
	if g.isMySQLDialect() {
		anyFuncs = []string{"COALESCE", "IFNULL", "NULLIF", "IF", "CAST"}
	} else {
		// PostgreSQL/GaussDB-A: no IF/IFNULL, use COALESCE/NULLIF/CASE WHEN
		anyFuncs = []string{"COALESCE", "NULLIF", "CAST"}
	}

	switch typeConstraint {
	case "int", "bigint", "smallint", "tinyint", "int4", "int8", "int2":
		return pickRandom(g.Rand, intFuncs)
	case "float", "double", "decimal", "numeric", "float4", "float8", "double precision":
		return pickRandom(g.Rand, floatFuncs)
	case "varchar", "char", "text", "bpchar", "varchar(20)", "varchar(50)", "name":
		return pickRandom(g.Rand, stringFuncs)
	default:
		return pickRandom(g.Rand, anyFuncs)
	}
}

// functionArgCount: return the number of arguments for a standard function
func functionArgCount(funcName string) int {
	switch funcName {
	case "PI":
		return 0
	case "ABS", "CEILING", "FLOOR", "ROUND", "SIGN", "ACOS", "ASIN", "ATAN",
		"COS", "EXP", "LN", "LOG2", "SQRT", "LENGTH", "LOWER", "UPPER",
		"TRIM", "REVERSE":
		return 1
	case "MOD", "POW", "LOG", "LEFT", "RIGHT", "SUBSTRING", "REPLACE", "CONCAT":
		return 2
	default:
		return 1
	}
}

// generateCaseExpr: generate a CASE WHEN expression (same syntax across dialects)
func (g *QueryGenerator) generateCaseExpr(scope *Scope, depth int, typeConstraint string) string {
	condition := g.generateBoolExpr(scope, depth+1)
	thenVal := g.generateExpression(scope, depth+1, typeConstraint)
	elseVal := g.generateExpression(scope, depth+1, typeConstraint)
	return fmt.Sprintf("CASE WHEN %s THEN %s ELSE %s END", condition, thenVal, elseVal)
}

// generateInExpr: generate an IN expression
// Parenthesize left-hand expr to avoid syntax errors with NOT IN
func (g *QueryGenerator) generateInExpr(scope *Scope, depth int) string {
	expr := g.generateExpression(scope, depth+1, "any")
	numValues := g.randInt(2, 3)
	values := make([]string, numValues)
	for i := 0; i < numValues; i++ {
		values[i] = g.generateLeaf(scope, "any")
	}
	not := ""
	if g.randBool() {
		not = " NOT"
	}
	return fmt.Sprintf("(%s)%s IN (%s)", expr, not, joinStrings(values, ", "))
}