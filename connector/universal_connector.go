package connector

import (
	"fmt"
	"github.com/pkg/errors"
)

// DBType: database type enumeration
type DBType string

const (
	DBMySQL      DBType = "mysql"
	DBPostgreSQL DBType = "postgresql" // Native PostgreSQL
	DBOpenGaussM DBType = "opengauss_m" // openGauss M mode (MySQL compatibility)
	DBOpenGaussA DBType = "opengauss_a" // openGauss A mode (Oracle compatibility)
	DBGaussDBM   DBType = "gaussdb_m"   // GaussDB M mode (MySQL compatibility)
	DBGaussDBA   DBType = "gaussdb_a"   // GaussDB A mode (Oracle compatibility)
)

// UniversalConnector: unified connector supporting multiple database types
type UniversalConnector struct {
	DBType     DBType
	Host       string
	Port       int
	Username   string
	Password   string
	DbName     string
	mysqlConn  *Connector          // MySQL connection
	ogConn     *OpenGaussConnector // openGauss M mode connection
	gaConn     *GaussDBAConnector  // GaussDB A mode connection
	pgConn     *PostgreSQLConnector // Native PostgreSQL connection
}

// NewUniversalConnector: create connector based on database type
func NewUniversalConnector(dbType DBType, host string, port int, username string, password string, dbname string) (*UniversalConnector, error) {
	uc := &UniversalConnector{
		DBType:   dbType,
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		DbName:   dbname,
	}

	switch dbType {
	case DBMySQL:
		conn, err := NewConnector(host, port, username, password, dbname)
		if err != nil {
			return nil, err
		}
		uc.mysqlConn = conn
	case DBPostgreSQL:
		conn, err := NewPostgreSQLConnector(host, port, username, password, dbname)
		if err != nil {
			return nil, err
		}
		uc.pgConn = conn
	case DBOpenGaussM, DBGaussDBM:
		conn, err := NewOpenGaussConnector(host, port, username, password, dbname)
		if err != nil {
			return nil, err
		}
		uc.ogConn = conn
	case DBOpenGaussA, DBGaussDBA:
		conn, err := NewGaussDBAConnector(host, port, username, password, dbname)
		if err != nil {
			return nil, err
		}
		uc.gaConn = conn
	default:
		return nil, errors.New(fmt.Sprintf("[NewUniversalConnector]unsupported database type: %s", dbType))
	}

	return uc, nil
}

// ExecSQL: execute SQL and return result
func (uc *UniversalConnector) ExecSQL(sql string) *Result {
	switch uc.DBType {
	case DBMySQL:
		return uc.mysqlConn.ExecSQL(sql)
	case DBPostgreSQL:
		return uc.pgConn.ExecSQL(sql)
	case DBOpenGaussM, DBGaussDBM:
		return uc.ogConn.ExecSQL(sql)
	case DBOpenGaussA, DBGaussDBA:
		return uc.gaConn.ExecSQL(sql)
	default:
		return &Result{
			Err: errors.New(fmt.Sprintf("[UniversalConnector.ExecSQL]unsupported database type: %s", uc.DBType)),
		}
	}
}

// InitDB: initialize database
func (uc *UniversalConnector) InitDB() error {
	switch uc.DBType {
	case DBMySQL:
		return uc.mysqlConn.InitDB()
	case DBPostgreSQL:
		return uc.pgConn.InitDB()
	case DBOpenGaussM, DBGaussDBM:
		return uc.ogConn.InitDB()
	case DBOpenGaussA, DBGaussDBA:
		return uc.gaConn.InitDB()
	default:
		return errors.New(fmt.Sprintf("[UniversalConnector.InitDB]unsupported database type: %s", uc.DBType))
	}
}

// InitDBWithDDL: init database and execute DDL
func (uc *UniversalConnector) InitDBWithDDL(ddlSqls []*EachSql) error {
	switch uc.DBType {
	case DBMySQL:
		return uc.mysqlConn.InitDBWithDDL(ddlSqls)
	case DBPostgreSQL:
		return uc.pgConn.InitDBWithDDL(ddlSqls)
	case DBOpenGaussM, DBGaussDBM:
		return uc.ogConn.InitDBWithDDL(ddlSqls)
	case DBOpenGaussA, DBGaussDBA:
		return uc.gaConn.InitDBWithDDL(ddlSqls)
	default:
		return errors.New(fmt.Sprintf("[UniversalConnector.InitDBWithDDL]unsupported database type: %s", uc.DBType))
	}
}

// Close: close connection
func (uc *UniversalConnector) Close() {
	switch uc.DBType {
	case DBMySQL:
		uc.mysqlConn.Close()
	case DBPostgreSQL:
		uc.pgConn.Close()
	case DBOpenGaussM, DBGaussDBM:
		uc.ogConn.Close()
	case DBOpenGaussA, DBGaussDBA:
		uc.gaConn.Close()
	}
}

// GetDBType: get database type
func (uc *UniversalConnector) GetDBType() DBType {
	return uc.DBType
}

// DiscoverSchema: discover database schema (implements SchemaDiscoverer)
func (uc *UniversalConnector) DiscoverSchema() (*SchemaInfo, error) {
	switch uc.DBType {
	case DBMySQL:
		return uc.mysqlConn.DiscoverSchema()
	case DBPostgreSQL:
		return uc.pgConn.DiscoverSchema()
	case DBOpenGaussM, DBGaussDBM:
		return uc.ogConn.DiscoverSchema()
	case DBOpenGaussA, DBGaussDBA:
		return uc.gaConn.DiscoverSchema()
	default:
		return nil, errors.New(fmt.Sprintf("[UniversalConnector.DiscoverSchema]unsupported database type: %s", uc.DBType))
	}
}