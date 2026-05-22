package connector

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	_ "gitee.com/opengauss/openGauss-connector-go-pq"
)

// OpenGaussConnector: connect to openGauss database (M mode - MySQL compatibility)
// Uses openGauss-connector-go-pq driver for SHA256 authentication support
type OpenGaussConnector struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
	db       *sql.DB
}

// NewOpenGaussConnector: create OpenGaussConnector
// For M mode database, use: CREATE DATABASE dbname WITH DBCOMPATIBILITY 'M'
func NewOpenGaussConnector(host string, port int, username string, password string, dbname string) (*OpenGaussConnector, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbname)

	db, err := sql.Open("opengauss", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "[NewOpenGaussConnector]sql.Open error")
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "[NewOpenGaussConnector]ping error")
	}

	ogConn := &OpenGaussConnector{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
		db:       db,
	}

	return ogConn, nil
}

// ExecSQL: execute SQL and return result
func (ogConn *OpenGaussConnector) ExecSQL(sqlStr string) *Result {
	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows:        make([][]string, 0),
		Err:         nil,
	}

	rows, err := ogConn.db.Query(sqlStr)
	if err != nil {
		result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]query error")
		return result
	}
	defer rows.Close()

	// Get column info
	columns, err := rows.Columns()
	if err != nil {
		result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]get columns error")
		return result
	}
	result.ColumnNames = columns

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]get column types error")
		return result
	}
	for _, ct := range columnTypes {
		result.ColumnTypes = append(result.ColumnTypes, ct.DatabaseTypeName())
	}

	// Scan rows
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]scan error")
			return result
		}

		rowData := make([]string, len(columns))
		for i, v := range values {
			if v == nil {
				rowData[i] = "NULL"
			} else {
				rowData[i] = fmt.Sprintf("%v", v)
			}
		}
		result.Rows = append(result.Rows, rowData)
	}

	if rows.Err() != nil {
		result.Err = errors.Wrap(rows.Err(), "[OpenGaussConnector.ExecSQL]rows error")
		return result
	}

	return result
}

// InitDB: initialize database
// Note: For openGauss M mode, database should be created manually:
// CREATE DATABASE dbname WITH DBCOMPATIBILITY 'M'
func (ogConn *OpenGaussConnector) InitDB() error {
	// Execute DDL directly, assuming database already exists
	return nil
}

// InitDBWithDDL: init database and execute DDL
func (ogConn *OpenGaussConnector) InitDBWithDDL(ddlSqls []*EachSql) error {
	// Execute DDL directly (assuming database already exists in M mode)
	for _, ddlSql := range ddlSqls {
		result := ogConn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

// Close: close connection
func (ogConn *OpenGaussConnector) Close() {
	if ogConn.db != nil {
		ogConn.db.Close()
	}
}