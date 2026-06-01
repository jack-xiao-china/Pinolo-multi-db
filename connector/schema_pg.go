package connector

import (
	"fmt"
	"github.com/pkg/errors"
)

// DiscoverSchema: discover database schema from PostgreSQL pg_catalog
// Queries table and column metadata from the connected database
func (pgConn *PostgreSQLConnector) DiscoverSchema() (*SchemaInfo, error) {
	schema := &SchemaInfo{
		Tables: make([]TableInfo, 0),
	}

	// Query all tables in the public schema
	tableResult := pgConn.ExecSQL(
		"SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
	if tableResult.Err != nil {
		return nil, errors.Wrap(tableResult.Err, "[PostgreSQLConnector.DiscoverSchema]query tables error")
	}

	for _, row := range tableResult.Rows {
		tableName := row[0]
		tableInfo := TableInfo{
			Name:    tableName,
			Columns: make([]ColumnInfo, 0),
		}

		// Query column metadata for this table
		colResult := pgConn.ExecSQL(
			fmt.Sprintf("SELECT column_name, data_type, udt_name, "+
				"CASE WHEN column_name IN ("+
				"SELECT kcu.column_name FROM information_schema.table_constraints tc "+
				"JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name "+
				"WHERE tc.table_name = '%s' AND tc.constraint_type = 'PRIMARY KEY') "+
				"THEN true ELSE false END AS is_key, "+
				"is_nullable = 'YES' AS nullable "+
				"FROM information_schema.columns "+
				"WHERE table_schema = 'public' AND table_name = '%s' "+
				"ORDER BY ordinal_position",
				tableName, tableName))
		if colResult.Err != nil {
			return nil, errors.Wrap(colResult.Err,
				fmt.Sprintf("[PostgreSQLConnector.DiscoverSchema]query columns for table %s error", tableName))
		}

		for _, colRow := range colResult.Rows {
			colInfo := ColumnInfo{
				Name:     colRow[0],                       // column_name
				Type:     colRow[2],                       // udt_name (e.g., "int4", "varchar", "float8")
				IsKey:    colRow[3] == "true",             // is_key
				Nullable: colRow[4] == "true" || colRow[4] == "t", // nullable
			}
			tableInfo.Columns = append(tableInfo.Columns, colInfo)
		}

		schema.Tables = append(schema.Tables, tableInfo)
	}

	return schema, nil
}