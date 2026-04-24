package connector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

// OpenGaussConnector: connect to openGauss database (M mode - MySQL compatibility)
// Uses PostgreSQL protocol with pgx driver
type OpenGaussConnector struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
	conn     *pgx.Conn
}

// NewOpenGaussConnector: create OpenGaussConnector
// For M mode database, use: CREATE DATABASE dbname WITH DBCOMPATIBILITY 'M'
func NewOpenGaussConnector(host string, port int, username string, password string, dbname string) (*OpenGaussConnector, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, dbname)

	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		return nil, errors.Wrap(err, "[NewOpenGaussConnector]connect error")
	}

	ogConn := &OpenGaussConnector{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
		conn:     conn,
	}

	return ogConn, nil
}

// ExecSQL: execute SQL and return result
func (ogConn *OpenGaussConnector) ExecSQL(sql string) *Result {
	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows:        make([][]string, 0),
		Err:         nil,
	}

	rows, err := ogConn.conn.Query(context.Background(), sql)
	if err != nil {
		result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]query error")
		return result
	}
	defer rows.Close()

	// Get column info
	fieldDescriptions := rows.FieldDescriptions()
	for _, fd := range fieldDescriptions {
		result.ColumnNames = append(result.ColumnNames, string(fd.Name))
		// PostgreSQL uses OID for type, convert to string
		result.ColumnTypes = append(result.ColumnTypes, fmt.Sprintf("OID_%d", fd.DataTypeOID))
	}

	// Scan rows
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			result.Err = errors.Wrap(err, "[OpenGaussConnector.ExecSQL]scan values error")
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
	if ogConn.conn != nil {
		ogConn.conn.Close(context.Background())
	}
}