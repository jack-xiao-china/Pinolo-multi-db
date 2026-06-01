package generator

import (
	"testing"

	"github.com/pingcap/tidb/parser"
	_ "github.com/pingcap/tidb/parser/test_driver"
	"github.com/qaqcatz/impomysql/connector"
)

// TestSchemaInfo: create a minimal test schema for unit testing
func createTestSchema() *connector.SchemaInfo {
	return &connector.SchemaInfo{
		Tables: []connector.TableInfo{
			{
				Name: "t1",
				Columns: []connector.ColumnInfo{
					{Name: "col_int", Type: "int", IsKey: true, Nullable: false},
					{Name: "col_float", Type: "float", IsKey: false, Nullable: true},
					{Name: "col_varchar", Type: "varchar(20)", IsKey: false, Nullable: true},
				},
			},
			{
				Name: "t2",
				Columns: []connector.ColumnInfo{
					{Name: "id", Type: "bigint", IsKey: true, Nullable: false},
					{Name: "name", Type: "varchar(50)", IsKey: false, Nullable: true},
					{Name: "score", Type: "double", IsKey: false, Nullable: true},
				},
			},
		},
	}
}

// TestGenerateDDL: verify DDL generation produces valid CREATE TABLE statements
func TestGenerateDDL(t *testing.T) {
	schema := createTestSchema()
	config := DefaultGeneratorConfig()
	gen := NewQueryGenerator(config, schema)

	ddl := gen.GenerateDDL()
	if len(ddl) != 2 {
		t.Fatalf("expected 2 DDL statements, got %d", len(ddl))
	}

	p := parser.New()
	for _, sql := range ddl {
		_, _, err := p.Parse(sql, "", "")
		if err != nil {
			t.Errorf("DDL not parseable: %s, error: %v", sql, err)
		}
	}
}

// TestGenerateSelect: verify SELECT queries are parseable by TiDB parser
func TestGenerateSelect(t *testing.T) {
	schema := createTestSchema()
	config := DefaultGeneratorConfig()
	config.QueriesNum = 20
	config.Seed = 42 // Fixed seed for deterministic test
	gen := NewQueryGenerator(config, schema)

	sqls := gen.GenerateSelects(20)
	p := parser.New()

	parseSuccess := 0
	for _, sql := range sqls {
		_, _, err := p.Parse(sql, "", "")
		if err != nil {
			t.Logf("SELECT not parseable: %s, error: %v", sql, err)
		} else {
			parseSuccess++
		}
	}

	// Allow some failures since random generation may produce invalid combinations
	// But at least 50% should parse successfully
	if parseSuccess < 10 {
		t.Errorf("too few parseable queries: %d out of 20", parseSuccess)
	} else {
		t.Logf("parseable queries: %d out of 20", parseSuccess)
	}
}

// TestGeneratorConfigDefaults: verify default config values
func TestGeneratorConfigDefaults(t *testing.T) {
	config := DefaultGeneratorConfig()
	if config.MaxDepth != 3 {
		t.Errorf("expected MaxDepth=3, got %d", config.MaxDepth)
	}
	if !config.EnableJoin {
		t.Error("expected EnableJoin=true")
	}
	if !config.EnableSubquery {
		t.Error("expected EnableSubquery=true")
	}
	if !config.EnableUnion {
		t.Error("expected EnableUnion=true")
	}
}

// TestScopePopulation: verify scope is correctly populated from schema
func TestScopePopulation(t *testing.T) {
	schema := createTestSchema()
	scope := NewScope(schema, 0)

	// Add table t1 with alias t0
	scope.AddTable("t1", "t0")

	if scope.NumTables() != 1 {
		t.Errorf("expected 1 table, got %d", scope.NumTables())
	}
	// t1 has 3 columns
	if scope.NumColumns() != 3 {
		t.Errorf("expected 3 columns, got %d", scope.NumColumns())
	}

	// Verify column references are correct
	intCols := scope.ColumnsOfType("int")
	if len(intCols) < 1 {
		t.Errorf("expected at least 1 int column, got %d", len(intCols))
	}
}