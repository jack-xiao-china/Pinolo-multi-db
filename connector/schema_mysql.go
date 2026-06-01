package connector

import (
	"fmt"
	"github.com/pkg/errors"
)

// DiscoverSchema: discover database schema from MySQL INFORMATION_SCHEMA
// Queries table and column metadata from the connected database
func (conn *Connector) DiscoverSchema() (*SchemaInfo, error) {
	schema := &SchemaInfo{
		Tables: make([]TableInfo, 0),
	}

	// Query all base tables in the current database
	tableResult := conn.ExecSQL(
		fmt.Sprintf("SELECT TABLE_NAME FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = '%s' AND TABLE_TYPE = 'BASE TABLE'",
			conn.DbName))
	if tableResult.Err != nil {
		return nil, errors.Wrap(tableResult.Err, "[Connector.DiscoverSchema]query tables error")
	}

	for _, row := range tableResult.Rows {
		tableName := row[0]
		tableInfo := TableInfo{
			Name:    tableName,
			Columns: make([]ColumnInfo, 0),
		}

		// Query column metadata for this table
		colResult := conn.ExecSQL(
			fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, COLUMN_TYPE, COLUMN_KEY, IS_NULLABLE "+
				"FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = '%s' AND TABLE_NAME = '%s' "+
				"ORDER BY ORDINAL_POSITION",
				conn.DbName, tableName))
		if colResult.Err != nil {
			return nil, errors.Wrap(colResult.Err,
				fmt.Sprintf("[Connector.DiscoverSchema]query columns for table %s error", tableName))
		}

		for _, colRow := range colResult.Rows {
			colInfo := ColumnInfo{
				Name:     colRow[0], // COLUMN_NAME
				Type:     colRow[2], // COLUMN_TYPE (full type, e.g., "int", "varchar(20)")
				IsKey:    colRow[3] == "PRI" || colRow[3] == "UNI", // COLUMN_KEY
				Nullable: colRow[4] == "YES",                       // IS_NULLABLE
			}
			tableInfo.Columns = append(tableInfo.Columns, colInfo)
		}

		schema.Tables = append(schema.Tables, tableInfo)
	}

	return schema, nil
}