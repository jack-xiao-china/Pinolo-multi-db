package connector

import (
	"database/sql"
	"fmt"
	"github.com/pkg/errors"
	_ "gitee.com/opengauss/openGauss-connector-go-pq"
)

// GaussDBAConnector: connect to GaussDB/openGauss A mode (Oracle compatibility)
// Uses openGauss-connector-go-pq driver for SHA256 authentication support
// Database should be created with: CREATE DATABASE dbname WITH DBCOMPATIBILITY 'A'
type GaussDBAConnector struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
	db       *sql.DB
}

// NewGaussDBAConnector: create GaussDBAConnector for A mode
func NewGaussDBAConnector(host string, port int, username string, password string, dbname string) (*GaussDBAConnector, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, dbname)

	db, err := sql.Open("opengauss", connStr)
	if err != nil {
		return nil, errors.Wrap(err, "[NewGaussDBAConnector]sql.Open error")
	}

	// Test connection
	err = db.Ping()
	if err != nil {
		return nil, errors.Wrap(err, "[NewGaussDBAConnector]ping error")
	}

	gaConn := &GaussDBAConnector{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
		db:       db,
	}

	return gaConn, nil
}

// ExecSQL: execute SQL and return result
func (gaConn *GaussDBAConnector) ExecSQL(sqlStr string) *Result {
	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows:        make([][]string, 0),
		Err:         nil,
	}

	rows, err := gaConn.db.Query(sqlStr)
	if err != nil {
		result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]query error")
		return result
	}
	defer rows.Close()

	// Get column info
	columns, err := rows.Columns()
	if err != nil {
		result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]get columns error")
		return result
	}
	result.ColumnNames = columns

	// Get column types
	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]get column types error")
		return result
	}
	for _, ct := range columnTypes {
		result.ColumnTypes = append(result.ColumnTypes, ct.DatabaseTypeName())
	}

	// Scan rows
	for rows.Next() {
		// Create values slice for scanning
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]scan error")
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
		result.Err = errors.Wrap(rows.Err(), "[GaussDBAConnector.ExecSQL]rows error")
		return result
	}

	return result
}

// InitDB: initialize database
// Note: For GaussDB A mode, database should be created manually:
// CREATE DATABASE dbname WITH DBCOMPATIBILITY 'A'
func (gaConn *GaussDBAConnector) InitDB() error {
	// Execute DDL directly, assuming database already exists in A mode
	return nil
}

// InitDBWithDDL: init database and execute DDL
func (gaConn *GaussDBAConnector) InitDBWithDDL(ddlSqls []*EachSql) error {
	for _, ddlSql := range ddlSqls {
		result := gaConn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			return result.Err
		}
	}
	return nil
}

// Close: close connection
func (gaConn *GaussDBAConnector) Close() {
	if gaConn.db != nil {
		gaConn.db.Close()
	}
}