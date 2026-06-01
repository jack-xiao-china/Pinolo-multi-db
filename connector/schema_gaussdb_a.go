package connector

import (
	"fmt"
	"github.com/pkg/errors"
)

// DiscoverSchema: discover database schema from GaussDB/openGauss A mode (Oracle compatibility)
// Uses pg_catalog queries adapted for A mode
func (gaConn *GaussDBAConnector) DiscoverSchema() (*SchemaInfo, error) {
	schema := &SchemaInfo{
		Tables: make([]TableInfo, 0),
	}

	// Query all user tables in the current schema
	tableResult := gaConn.ExecSQL(
		fmt.Sprintf("SELECT TABLE_NAME FROM ALL_TABLES WHERE OWNER = '%s'",
			gaConn.Username))
	if tableResult.Err != nil {
		// Fall back to pg_tables for tables in public schema
		tableResult = gaConn.ExecSQL(
			"SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
		if tableResult.Err != nil {
			return nil, errors.Wrap(tableResult.Err, "[GaussDBAConnector.DiscoverSchema]query tables error")
		}
	}

	for _, row := range tableResult.Rows {
		tableName := row[0]
		tableInfo := TableInfo{
			Name:    tableName,
			Columns: make([]ColumnInfo, 0),
		}

		// Query column metadata - try ALL_TAB_COLUMNS first (A mode)
		colResult := gaConn.ExecSQL(
			fmt.Sprintf("SELECT COLUMN_NAME, DATA_TYPE, DATA_TYPE, 'NO', NULLABLE "+
				"FROM ALL_TAB_COLUMNS WHERE TABLE_NAME = '%s' AND OWNER = '%s' "+
				"ORDER BY COLUMN_ID",
				tableName, gaConn.Username))
		if colResult.Err != nil {
			// Fall back to information_schema.columns
			colResult = gaConn.ExecSQL(
				fmt.Sprintf("SELECT column_name, data_type, udt_name, "+
					"CASE WHEN column_name IN ("+
					"SELECT kcu.column_name FROM information_schema.table_constraints tc "+
					"JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name "+
					"WHERE tc.table_name = '%s' AND tc.constraint_type = 'PRIMARY KEY') "+
					"THEN 'PRI' ELSE '' END AS column_key, "+
					"is_nullable "+
					"FROM information_schema.columns "+
					"WHERE table_schema = 'public' AND table_name = '%s' "+
					"ORDER BY ordinal_position",
					tableName, tableName))
			if colResult.Err != nil {
				return nil, errors.Wrap(colResult.Err,
					fmt.Sprintf("[GaussDBAConnector.DiscoverSchema]query columns for table %s error", tableName))
			}
		}

		for _, colRow := range colResult.Rows {
			colInfo := ColumnInfo{
				Name:     colRow[0],
				Type:     colRow[2],
				IsKey:    colRow[3] == "PRI" || colRow[3] == "UNI" || colRow[3] == "P",
				Nullable: colRow[4] == "YES" || colRow[4] == "Y" || colRow[4] == "true" || colRow[4] == "t",
			}
			tableInfo.Columns = append(tableInfo.Columns, colInfo)
		}

		schema.Tables = append(schema.Tables, tableInfo)
	}

	return schema, nil
}