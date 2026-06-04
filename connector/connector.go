package connector

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"strconv"
	"time"
)

// DefaultQueryTimeout: maximum time allowed for a single query execution
const DefaultQueryTimeout = 60 * time.Second

// Connector: connect to MySQL, execute raw sql statements, return raw execution result or error.
type Connector struct {
	Host            string
	Port            int
	Username        string
	Password        string
	DbName          string
	db              *sql.DB
}

// NewConnector: create Connector. CREATE DATABASE IF NOT EXISTS dbname + USE dbname when dbname != ""
func NewConnector(host string, port int, username string, password string, dbname string) (*Connector, error) {
	// First, create a connection without database to create the database
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?allowOldPasswords=true",
		username, password, host, port, "")
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "[NewConnector]open dsn error")
	}
	conn := &Connector{
		Host:            host,
		Port:            port,
		Username:        username,
		Password:        password,
		DbName:          dbname,
		db:              db,
	}
	if dbname != "" {
		// CREATE DATABASE IF NOT EXISTS conn.DbName
		result := conn.ExecSQL("CREATE DATABASE IF NOT EXISTS " + conn.DbName)
		if result.Err != nil {
			return nil, result.Err
		}
		// Close the connection without database
		conn.db.Close()

		// Create a new connection with the database name in the DSN
		// This ensures all connections in the pool have the database selected
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?allowOldPasswords=true",
			username, password, host, port, dbname)
		db, err = sql.Open("mysql", dsn)
		if err != nil {
			return nil, errors.Wrap(err, "[NewConnector]open dsn with database error")
		}
		conn.db = db
	}
	return conn, nil
}

// Connector.ExecSQL: execute sql, return *Result.
func (conn *Connector) ExecSQL(sql string) *Result {
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), DefaultQueryTimeout)
	defer cancel()
	rows, err := conn.db.QueryContext(ctx, sql)
	if err != nil {
		return &Result{
			Err: errors.Wrap(err, "[Connector.ExecSQL]execute sql error"),
		}
	}
	defer rows.Close()

	result := &Result{
		ColumnNames: make([]string, 0),
		ColumnTypes: make([]string, 0),
		Rows: make([][]string, 0),
		Err: nil,
	}
	for rows.Next() {
		columnTypes, err := rows.ColumnTypes()
		if err != nil {
			return &Result{
				Err: errors.Wrap(err, "[Connector.ExecSQL]get column type error"),
			}
		}
		if len(result.ColumnNames) == 0 {
			for _, columnType := range columnTypes {
				result.ColumnNames = append(result.ColumnNames, columnType.Name())
				result.ColumnTypes = append(result.ColumnTypes, columnType.DatabaseTypeName())
			}
		} else {
			if len(columnTypes) != len(result.ColumnNames) {
				return &Result{
					Err: errors.New("[Connector.ExecSQL]|columnTypes|("+strconv.Itoa(len(columnTypes))+") != "+
						"|columnNames|("+strconv.Itoa(len(result.ColumnNames))+")"),
				}
			}
			for i, columnType := range columnTypes {
				if columnType.Name() != result.ColumnNames[i] {
					return &Result{
						Err: errors.New("[Connector.ExecSQL]columnType.Name()("+columnType.Name()+") != "+
							"result.ColumnNames[i]("+result.ColumnNames[i]+")"),
					}
				}
				if columnType.DatabaseTypeName() != result.ColumnTypes[i] {
					return &Result{
						Err: errors.New("[Connector.ExecSQL]columnType.DatabaseTypeName()("+columnType.DatabaseTypeName()+") != "+
							"result.ColumnTypes[i]("+result.ColumnTypes[i]+")"),
					}
				}
			}
		}

		// gorm cannot convert NULL to string, we should use []byte
		data := make([][]byte, len(columnTypes))
		dataI := make([]interface{}, len(columnTypes))
		for i, _ := range data {
			dataI[i] = &data[i]
		}
		err = rows.Scan(dataI...)
		if err != nil {
			return &Result{
				Err: errors.Wrap(err, "[Connector.ExecSQL]scan rows error"),
			}
		}

		dataS := make([]string, len(columnTypes))
		for i, _ := range data {
			if data[i] == nil {
				dataS[i] = NullMarker
			} else {
				dataS[i] = string(data[i])
			}
		}
		result.Rows = append(result.Rows, dataS)
	}
	if rows.Err() != nil {
		return &Result{
			Err: errors.Wrap(rows.Err(), "[Connector.ExecSQL]rows error"),
		}
	}

	result.Time = time.Since(startTime)
	return result
}

// Connector.InitDB:
//   DROP DATABASE IF EXISTS Connector.DbName
//   CREATE DATABASE Connector.DbName
//   USE Connector.DbName
func (conn *Connector) InitDB() error {
	result := conn.ExecSQL("DROP DATABASE IF EXISTS " + conn.DbName)
	if result.Err != nil {
		return result.Err
	}
	result = conn.ExecSQL("CREATE DATABASE " + conn.DbName)
	if result.Err != nil {
		return result.Err
	}
	result = conn.ExecSQL("USE " + conn.DbName)
	if result.Err != nil {
		return result.Err
	}
	return nil
}

// Connector.InitDBWithDDL: init database and execute ddl sqls
func (conn *Connector) InitDBWithDDL(ddlSqls []*EachSql) error {
	err := conn.InitDB()
	if err != nil {
		return err
	}
	for _, ddlSql := range ddlSqls {
		result := conn.ExecSQL(ddlSql.Sql)
		if result.Err != nil {
			return result.Err
		}
	}
	// Re-select database after executing DDL statements
	// DDL operations (especially CREATE TABLE) may reset the database context
	if conn.DbName != "" {
		result := conn.ExecSQL("USE " + conn.DbName)
		if result.Err != nil {
			return errors.Wrap(result.Err, "[InitDBWithDDL]re-select database error")
		}
	}
	return nil
}

// Connector.InitDBWithDDLPath: init database and execute ddl sqls from ddlPath
func (conn *Connector) InitDBWithDDLPath(ddlPath string) error {
	ddlSqls, err := ExtractSqlFromPath(ddlPath)
	if err != nil {
		return err
	}
	err = conn.InitDBWithDDL(ddlSqls)
	if err != nil {
		return err
	}
	return nil
}

func (conn *Connector) Close() {
	_ = conn.db.Close()
}