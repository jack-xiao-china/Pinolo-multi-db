package connector

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pkg/errors"
	"time"
)

// PostgreSQLConnector: connect to native PostgreSQL database
// Uses pgx/v5 pgxpool for connection pooling (official PostgreSQL Go driver)
type PostgreSQLConnector struct {
	Host     string
	Port     int
	Username string
	Password string
	DbName   string
	pool     *pgxpool.Pool
}

// NewPostgreSQLConnector: create PostgreSQL connector with connection pool
func NewPostgreSQLConnector(host string, port int, username string, password string, dbname string) (*PostgreSQLConnector, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		username, password, host, port, dbname)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, errors.Wrap(err, "[NewPostgreSQLConnector]parse config error")
	}

	// Set pool defaults
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, errors.Wrap(err, "[NewPostgreSQLConnector]pool creation error")
	}

	// Test connection
	err = pool.Ping(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "[NewPostgreSQLConnector]ping error")
	}

	pgConn := &PostgreSQLConnector{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
		pool:     pool,
	}

	return pgConn, nil
}

// ExecSQL: execute SQL and return Result (implements SQLExecutor interface)
func (pgConn *PostgreSQLConnector) ExecSQL(sql string) *Result {
	startTime := time.Now()

	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows:        make([][]string, 0),
		Err:         nil,
	}

	rows, err := pgConn.pool.Query(context.Background(), sql)
	if err != nil {
		result.Err = errors.Wrap(err, "[PostgreSQLConnector.ExecSQL]query error")
		return result
	}
	defer rows.Close()

	// Get column info
	fieldDescriptions := rows.FieldDescriptions()
	for _, fd := range fieldDescriptions {
		result.ColumnNames = append(result.ColumnNames, string(fd.Name))
		result.ColumnTypes = append(result.ColumnTypes, pgConn.getColumnTypeName(fd.DataTypeOID))
	}

	// Scan rows
	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			result.Err = errors.Wrap(err, "[PostgreSQLConnector.ExecSQL]scan values error")
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
		result.Err = errors.Wrap(rows.Err(), "[PostgreSQLConnector.ExecSQL]rows error")
		return result
	}

	result.Time = time.Since(startTime)
	return result
}

// getColumnTypeName: convert PostgreSQL OID to type name
func (pgConn *PostgreSQLConnector) getColumnTypeName(oid uint32) string {
	// Common PostgreSQL type OIDs
	typeNames := map[uint32]string{
		16:    "bool",
		20:    "int8",
		21:    "int2",
		23:    "int4",
		25:    "text",
		700:   "float4",
		701:   "float8",
		1042:  "bpchar",
		1043:  "varchar",
		1082:  "date",
		1083:  "time",
		1114:  "timestamp",
		1184:  "timestamptz",
		1700:  "numeric",
		2278:  "void",
	}

	if name, ok := typeNames[oid]; ok {
		return name
	}
	return fmt.Sprintf("OID_%d", oid)
}

// InitDB: clear all user tables in the current database
// PostgreSQL cannot DROP DATABASE while connected to it, so we drop tables instead
func (pgConn *PostgreSQLConnector) InitDB() error {
	result := pgConn.ExecSQL(
		"SELECT tablename FROM pg_tables WHERE schemaname = 'public'")
	if result.Err != nil {
		return errors.Wrap(result.Err, "[PostgreSQLConnector.InitDB]query tables error")
	}
	for _, row := range result.Rows {
		if len(row) > 0 {
			tableName := row[0]
			dropResult := pgConn.ExecSQL(fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName))
			if dropResult.Err != nil {
				return errors.Wrap(dropResult.Err, fmt.Sprintf("[PostgreSQLConnector.InitDB]drop table %s error", tableName))
			}
		}
	}
	return nil
}

// InitDBWithDDL: execute DDL statements to set up test tables
func (pgConn *PostgreSQLConnector) InitDBWithDDL(ddlSqls []*EachSql) error {
	for _, ddlSql := range ddlSqls {
		result := pgConn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			// Log but continue for some errors (e.g., DROP TABLE IF EXISTS)
			// Only return error for CREATE failures
			if !isIgnorableError(result.Err) {
				return result.Err
			}
		}
	}
	return nil
}

// isIgnorableError: check if error can be ignored during DDL execution
func isIgnorableError(err error) bool {
	if err == nil {
		return true
	}
	errMsg := err.Error()
	// Ignore "does not exist" errors for DROP statements
	return contains(errMsg, "does not exist") ||
		contains(errMsg, "already exists")
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Close: close connection pool
func (pgConn *PostgreSQLConnector) Close() {
	if pgConn.pool != nil {
		pgConn.pool.Close()
	}
}