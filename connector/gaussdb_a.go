package connector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

// GaussDBAConnector: connect to GaussDB/openGauss A mode (Oracle compatibility)
// Uses PostgreSQL protocol with pgx driver
// Database should be created with: CREATE DATABASE dbname WITH DBCOMPATIBILITY 'A'
type GaussDBAConnector struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
	conn     *pgx.Conn
}

// NewGaussDBAConnector: create GaussDBAConnector for A mode
func NewGaussDBAConnector(host string, port int, username string, password string, dbname string) (*GaussDBAConnector, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, dbname)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, errors.Wrap(err, "[NewGaussDBAConnector]connect error")
	}

	gaConn := &GaussDBAConnector{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
		conn:     conn,
	}

	return gaConn, nil
}

// ExecSQL: execute SQL and return result
func (gaConn *GaussDBAConnector) ExecSQL(sql string) *Result {
	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows:        make([][]string, 0),
		Err:         nil,
	}

	rows, err := gaConn.conn.Query(context.Background(), sql)
	if err != nil {
		result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]query error")
		return result
	}
	defer rows.Close()

	// Get column info
	fieldDescriptions := rows.FieldDescriptions()
	for _, fd := range fieldDescriptions {
		result.ColumnNames = append(result.ColumnNames, string(fd.Name))
		result.ColumnTypes = append(result.ColumnTypes, fmt.Sprintf("OID_%d", fd.DataTypeOID))
	}

	// Scan rows
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			result.Err = errors.Wrap(err, "[GaussDBAConnector.ExecSQL]scan values error")
			return result
		}

		rowData := make([]string, len(values))
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
	if gaConn.conn != nil {
		gaConn.conn.Close(context.Background())
	}
}