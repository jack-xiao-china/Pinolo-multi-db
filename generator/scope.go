package generator

import (
	"github.com/qaqcatz/impomysql/connector"
	"math/rand"
)

// Scope: tracks available tables, columns, and type information during SQL generation
// Inspired by EET's scope struct (prod.hh) which provides type-safe generation context
type Scope struct {
	Tables  []TableRef  // Available table references (with aliases)
	Columns []ColumnRef // Available column references (with type info)
	Schema  *connector.SchemaInfo // Original schema metadata
	Level   int         // Nesting depth (0 = top-level, increases for subqueries)
}

// TableRef: a table reference with alias in the current scope
type TableRef struct {
	TableName string // Original table name from schema
	Alias     string // Alias used in current query (e.g., "t0")
}

// ColumnRef: a column reference with table alias and type information
type ColumnRef struct {
	TableName  string // Original table name
	TableAlias string // Table alias in current scope
	ColumnName string // Column name
	ColumnType string // Column type (e.g., "int", "varchar(20)", "double")
	IsKey      bool   // Whether this is a key column
	Nullable   bool   // Whether this column is nullable
}

// NewScope: create a new scope from schema metadata
func NewScope(schema *connector.SchemaInfo, level int) *Scope {
	scope := &Scope{
		Schema:  schema,
		Level:   level,
		Tables:  make([]TableRef, 0),
		Columns: make([]ColumnRef, 0),
	}
	// Initially empty - tables/columns populated when FROM clause is built
	return scope
}

// AddTable: add a table reference to the scope with given alias
func (s *Scope) AddTable(tableName string, alias string) {
	s.Tables = append(s.Tables, TableRef{
		TableName: tableName,
		Alias:     alias,
	})

	// Find columns from schema and add them with alias
	for _, table := range s.Schema.Tables {
		if table.Name == tableName {
			for _, col := range table.Columns {
				s.Columns = append(s.Columns, ColumnRef{
					TableName:  tableName,
					TableAlias: alias,
					ColumnName: col.Name,
					ColumnType: col.Type,
					IsKey:      col.IsKey,
					Nullable:   col.Nullable,
				})
			}
			break
		}
	}
}

// ColumnsOfType: return column references matching a type constraint
// Used for type-safe expression generation (inspired by EET's scope.refs_of_type)
func (s *Scope) ColumnsOfType(typeConstraint string) []ColumnRef {
	if typeConstraint == "" || typeConstraint == "any" {
		return s.Columns
	}
	result := make([]ColumnRef, 0)
	for _, col := range s.Columns {
		if typeMatch(col.ColumnType, typeConstraint) {
			result = append(result, col)
		}
	}
	return result
}

// NumTables: number of available tables in scope
func (s *Scope) NumTables() int {
	return len(s.Tables)
}

// NumColumns: number of available columns in scope
func (s *Scope) NumColumns() int {
	return len(s.Columns)
}

// EnumColumns: return column references that are ENUM type
func (s *Scope) EnumColumns() []ColumnRef {
	result := make([]ColumnRef, 0)
	for _, col := range s.Columns {
		if normalizeType(col.ColumnType) == "enum" {
			result = append(result, col)
		}
	}
	return result
}

// PickEnumValue: pick a random value from a random ENUM column in the scope
// Returns ("'value'", true) if found, ("", false) if no ENUM columns available
func (s *Scope) PickEnumValue(r *rand.Rand) (string, bool) {
	enumCols := s.EnumColumns()
	if len(enumCols) == 0 {
		return "", false
	}
	col := pickRandom(r, enumCols)
	values := parseEnumValues(col.ColumnType)
	if len(values) == 0 {
		return "", false
	}
	val := pickRandom(r, values)
	return "'" + val + "'", true
}

// typeMatch: check if a column type matches a type constraint
// Uses broad category matching for flexibility
func typeMatch(actualType string, constraint string) bool {
	if constraint == "" || constraint == "any" {
		return true
	}
	actual := normalizeType(actualType)
	req := normalizeType(constraint)

	if actual == req {
		return true
	}
	// Broad category matching
	switch req {
	case "int":
		return actual == "int" || actual == "bigint" || actual == "smallint" || actual == "tinyint" || actual == "mediumint"
	case "float":
		return actual == "float" || actual == "double" || actual == "decimal" || actual == "numeric"
	case "string":
		return actual == "varchar" || actual == "char" || actual == "text" || actual == "longtext" || actual == "mediumtext" || actual == "tinytext" || actual == "enum"
	case "bool":
		return actual == "tinyint" || actual == "bool" || actual == "boolean"
	}
	return false
}

// normalizeType: normalize a column type string to a base type name
// e.g., "varchar(20)" → "varchar", "int(11)" → "int", "bigint(20)" → "bigint"
// "enum('a','b','c')" → "enum"
func normalizeType(colType string) string {
	// Remove parentheses content: varchar(20) → varchar, int(11) → int
	for i := 0; i < len(colType); i++ {
		if colType[i] == '(' {
			return colType[:i]
		}
	}
	return colType
}

// parseEnumValues: extract values from an ENUM column type string
// "enum('a','b','c')" → ["a", "b", "c"]
// Returns nil if the type is not an ENUM
func parseEnumValues(colType string) []string {
	if len(colType) < 6 || colType[:4] != "enum" {
		return nil
	}
	// Find the opening paren
	start := -1
	for i := 0; i < len(colType); i++ {
		if colType[i] == '(' {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil
	}
	// Parse quoted values: 'val1','val2','val3'
	values := make([]string, 0)
	inQuote := false
	current := ""
	for i := start; i < len(colType); i++ {
		ch := colType[i]
		if ch == '\'' {
			if inQuote {
				// End of quoted value
				values = append(values, current)
				current = ""
				inQuote = false
			} else {
				inQuote = true
			}
		} else if inQuote {
			current += string(ch)
		} else if ch == ')' {
			break
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}