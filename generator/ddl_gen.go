package generator

import (
	"fmt"
	"github.com/qaqcatz/impomysql/connector"
)

// generateDDLFromSchema: generate CREATE TABLE statements from SchemaInfo
func generateDDLFromSchema(schema *connector.SchemaInfo) []string {
	ddl := make([]string, len(schema.Tables))
	for i, table := range schema.Tables {
		ddl[i] = generateCreateTable(table)
	}
	return ddl
}

// generateCreateTable: generate a CREATE TABLE statement for a given table
func generateCreateTable(table connector.TableInfo) string {
	colDefs := make([]string, len(table.Columns))
	for i, col := range table.Columns {
		colDefs[i] = generateColumnDef(col)
	}
	return fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s)", table.Name, joinStrings(colDefs, ", "))
}

// generateColumnDef: generate a column definition for CREATE TABLE
func generateColumnDef(col connector.ColumnInfo) string {
	def := fmt.Sprintf("%s %s", col.Name, col.Type)
	if col.IsKey {
		def += " NOT NULL"
		if isIntType(col.Type) {
			def += " AUTO_INCREMENT"
		}
	} else if !col.Nullable {
		def += " NOT NULL"
	}

	// Add PRIMARY KEY for the first key column found
	// (This is a simplified approach - real DDL may have composite keys)
	return def
}

// generatePrimaryKeyClause: generate PRIMARY KEY clause from table columns
func generatePrimaryKeyClause(table connector.TableInfo) string {
	keyCols := make([]string, 0)
	for _, col := range table.Columns {
		if col.IsKey {
			keyCols = append(keyCols, col.Name)
		}
	}
	if len(keyCols) == 0 {
		return ""
	}
	return fmt.Sprintf("PRIMARY KEY (%s)", joinStrings(keyCols, ", "))
}

// isIntType: check if a type is an integer type
func isIntType(typeStr string) bool {
	base := normalizeType(typeStr)
	return base == "int" || base == "bigint" || base == "smallint" || base == "tinyint" || base == "mediumint"
}

// joinStrings: join string slice with separator
func joinStrings(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	result := items[0]
	for i := 1; i < len(items); i++ {
		result += sep + items[i]
	}
	return result
}