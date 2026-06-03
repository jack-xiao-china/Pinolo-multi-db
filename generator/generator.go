package generator

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/qaqcatz/impomysql/connector"
)

// GeneratorConfig: configuration for SQL random generation
// Controls expression depth, query count, feature toggles, and SQL dialect
type GeneratorConfig struct {
	Seed       int64  // Random seed (<=0: use current time)
	MaxDepth   int    // Maximum expression depth (default: 3)
	QueriesNum int    // Number of SELECT queries to generate
	Dialect    string // SQL dialect: "mysql", "postgresql", "gaussdb_a"
	// Feature toggles (default: all true for maximum coverage)
	EnableJoin     bool // Generate JOINs in FROM clause
	EnableSelfJoin bool // Generate self-joins (same table with different aliases)
	EnableSubquery bool // Generate subqueries (derived tables, EXISTS, IN)
	EnableUnion    bool // Generate UNION/UNION ALL queries
	EnableCTE      bool // Generate WITH/CTE queries
	EnableGroupBy  bool // Generate GROUP BY/HAVING
	EnableOrderBy  bool // Generate ORDER BY
	EnableLimit    bool // Generate LIMIT (requires ORDER BY)
}

// DefaultGeneratorConfig: returns a config with sensible defaults
func DefaultGeneratorConfig() *GeneratorConfig {
	return &GeneratorConfig{
		Seed:           0,
		MaxDepth:       3,
		QueriesNum:     100,
		EnableJoin:     true,
		EnableSelfJoin: true,
		EnableSubquery: true,
		EnableUnion:    true,
		EnableCTE:      true,
		EnableGroupBy:  true,
		EnableOrderBy:  true,
		EnableLimit:    true,
	}
}

// QueryGenerator: main SQL random generation engine
// Generates DDL (CREATE TABLE) and DML (SELECT) statements based on discovered schema
type QueryGenerator struct {
	Config *GeneratorConfig
	Schema *connector.SchemaInfo
	Scope  *Scope
	Rand   *rand.Rand
	// Counter for unique alias generation
	tableAliasSeq int
	colAliasSeq   int
	cteNameSeq    int
}

// NewQueryGenerator: create a new QueryGenerator with config and schema
func NewQueryGenerator(config *GeneratorConfig, schema *connector.SchemaInfo) *QueryGenerator {
	seed := config.Seed
	if seed <= 0 {
		seed = time.Now().UnixNano()
	}
	return &QueryGenerator{
		Config: config,
		Schema: schema,
		Scope:  NewScope(schema, 0),
		Rand:   rand.New(rand.NewSource(seed)),
	}
}

// GenerateDDL: generate CREATE TABLE statements from schema info
func (g *QueryGenerator) GenerateDDL() []string {
	return generateDDLFromSchema(g.Schema)
}

// GenerateSelect: generate a single random SELECT query
func (g *QueryGenerator) GenerateSelect() string {
	g.resetCounters()
	// Reset scope for fresh query
	g.Scope = NewScope(g.Schema, 0)

	// Choose query shape (4 shapes like SQLancer EET)
	shape := g.Rand.Intn(4)
	switch shape {
	case 0:
		return g.generatePlainSelect()
	case 1:
		if g.Config.EnableUnion {
			return g.generateUnionSelect()
		}
		return g.generatePlainSelect()
	case 2:
		if g.Config.EnableCTE {
			return g.generateCTESelect()
		}
		return g.generatePlainSelect()
	case 3:
		if g.Config.EnableSubquery {
			return g.generateDerivedSelect()
		}
		return g.generatePlainSelect()
	}
	return g.generatePlainSelect()
}

// GenerateSelects: generate multiple random SELECT queries
func (g *QueryGenerator) GenerateSelects(n int) []string {
	sqls := make([]string, n)
	for i := 0; i < n; i++ {
		sqls[i] = g.GenerateSelect()
	}
	return sqls
}

// resetCounters: reset alias counters for a fresh query
func (g *QueryGenerator) resetCounters() {
	g.tableAliasSeq = 0
	g.colAliasSeq = 0
	g.cteNameSeq = 0
}

// nextTableAlias: generate a unique table alias (t0, t1, t2, ...)
func (g *QueryGenerator) nextTableAlias() string {
	alias := "t" + itoa(g.tableAliasSeq)
	g.tableAliasSeq++
	return alias
}

// nextColAlias: generate a unique column alias (ref0, ref1, ref2, ...)
func (g *QueryGenerator) nextColAlias() string {
	alias := "ref" + itoa(g.colAliasSeq)
	g.colAliasSeq++
	return alias
}

// nextCTEName: generate a unique CTE name (cte0, cte1, cte2, ...)
func (g *QueryGenerator) nextCTEName() string {
	name := "cte" + itoa(g.cteNameSeq)
	g.cteNameSeq++
	return name
}

// itoa: simple int to string conversion
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

// dialect helper methods

// isMySQLDialect: returns true for MySQL-compatible dialects (mysql, gaussdb_m)
func (g *QueryGenerator) isMySQLDialect() bool {
	d := g.Config.Dialect
	return d == "mysql" || d == "opengauss_m" || d == "gaussdb_m" || d == ""
}

// isPostgreSQLDialect: returns true for PostgreSQL-compatible dialects (postgresql, gaussdb_a)
func (g *QueryGenerator) isPostgreSQLDialect() bool {
	d := g.Config.Dialect
	return d == "postgresql" || d == "opengauss_a" || d == "gaussdb_a"
}

// dialectFunctionIF: generate IF() for MySQL, CASE WHEN for PostgreSQL
func (g *QueryGenerator) dialectFunctionIF(condition string, thenVal string, elseVal string) string {
	if g.isMySQLDialect() {
		return fmt.Sprintf("IF(%s, %s, %s)", condition, thenVal, elseVal)
	}
	return fmt.Sprintf("CASE WHEN %s THEN %s ELSE %s END", condition, thenVal, elseVal)
}

// dialectFunctionIFNULL: generate IFNULL() for MySQL, COALESCE for PostgreSQL
func (g *QueryGenerator) dialectFunctionIFNULL(expr string) string {
	if g.isMySQLDialect() {
		return fmt.Sprintf("IFNULL(%s)", expr)
	}
	return fmt.Sprintf("COALESCE(%s)", expr)
}

// dialectCastType: map CAST target type per dialect
// MySQL: SIGNED, UNSIGNED, DOUBLE, CHAR(20)...
// PostgreSQL: integer, bigint, double precision, varchar(20), date...
func (g *QueryGenerator) dialectCastType(castType string) string {
	if g.isMySQLDialect() {
		return castType // Already MySQL-style
	}
	// PostgreSQL type mapping
	switch castType {
	case "SIGNED":
		return "integer"
	case "UNSIGNED":
		return "bigint"
	case "DOUBLE":
		return "double precision"
	case "DECIMAL(40,20)":
		return "numeric(40,20)"
	case "CHAR(20)":
		return "char(20)"
	case "VARCHAR(20)":
		return "varchar(20)"
	case "DATE":
		return "date"
	case "DATETIME":
		return "timestamp"
	default:
		return castType
	}
}

// dialectStringConstant: MySQL uses backtick-free strings, PostgreSQL same
func (g *QueryGenerator) dialectStringConstant(val string) string {
	return val // Both use single-quoted strings
}

// dialectUnionAll: UNION ALL syntax is the same across dialects
func (g *QueryGenerator) dialectUnionAll() string {
	return "UNION ALL"
}

// dialectUnionDistinct: MySQL uses UNION DISTINCT, PostgreSQL uses just UNION
func (g *QueryGenerator) dialectUnionDistinct() string {
	if g.isPostgreSQLDialect() {
		return "UNION"
	}
	return "UNION DISTINCT"
}

// dialectLimitClause: MySQL uses LIMIT n, PostgreSQL uses LIMIT n (same syntax)
func (g *QueryGenerator) dialectLimitClause(limit int) string {
	return fmt.Sprintf("LIMIT %d", limit)
}