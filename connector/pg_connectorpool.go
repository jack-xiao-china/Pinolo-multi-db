package connector

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"
)

// PgConnectorPool: thread pool of PostgreSQLConnector instances
// Each connector has its own database (DbPrefix + thread id)
type PgConnectorPool struct {
	Host       string
	Port       int
	Username   string
	Password   string
	DbPrefix   string
	ThreadNum  int
	ThreadPool chan *PostgreSQLConnector
}

// NewPgConnectorPool: create a pool of PostgreSQLConnector instances
// 1. Connect to maintenance database (postgres) to create per-thread databases
// 2. Create PostgreSQLConnector for each thread's database
func NewPgConnectorPool(host string, port int, username string, password string,
	dbPrefix string, threadNum int) (*PgConnectorPool, error) {

	// 1. Create per-thread databases using maintenance connection
	maintConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable",
		username, password, host, port)
	maintConn, err := pgx.Connect(context.Background(), maintConnStr)
	if err != nil {
		return nil, errors.Wrap(err, "[NewPgConnectorPool]connect to maintenance database error")
	}

	for i := 0; i < threadNum; i++ {
		dbName := dbPrefix + strconv.Itoa(i)
		// Drop if exists, then create
		_, _ = maintConn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
		_, err := maintConn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
		if err != nil {
			maintConn.Close(context.Background())
			return nil, errors.Wrap(err, fmt.Sprintf("[NewPgConnectorPool]create database %s error", dbName))
		}
	}
	maintConn.Close(context.Background())

	// 2. Create PostgreSQLConnector pool
	connectorPool := &PgConnectorPool{
		Host:      host,
		Port:      port,
		Username:  username,
		Password:  password,
		DbPrefix:  dbPrefix,
		ThreadNum: threadNum,
	}
	threadPool := make(chan *PostgreSQLConnector, threadNum)
	for i := 0; i < threadNum; i++ {
		dbName := dbPrefix + strconv.Itoa(i)
		conn, err := NewPostgreSQLConnector(host, port, username, password, dbName)
		if err != nil {
			// Clean up already created connectors
			for j := 0; j < i; j++ {
				conn := <-threadPool
				conn.Close()
			}
			return nil, errors.Wrap(err, fmt.Sprintf("[NewPgConnectorPool]create connector for %s error", dbName))
		}
		threadPool <- conn
	}
	connectorPool.ThreadPool = threadPool
	return connectorPool, nil
}

// WaitForFree: wait for a free connector from the pool
func (connPool *PgConnectorPool) WaitForFree() *PostgreSQLConnector {
	conn := <-connPool.ThreadPool
	return conn
}

// BackToPool: return a connector to the pool
func (connPool *PgConnectorPool) BackToPool(conn *PostgreSQLConnector) {
	connPool.ThreadPool <- conn
}

// Close: close all connectors and the channel
func (connPool *PgConnectorPool) Close() {
	for i := 0; i < connPool.ThreadNum; i++ {
		conn := <-connPool.ThreadPool
		conn.Close()
	}
	close(connPool.ThreadPool)
}

// ResetDB: drop and recreate a specific thread's database
// Used when a task needs to reset its database state
func (connPool *PgConnectorPool) ResetDB(threadId int) error {
	dbName := connPool.DbPrefix + strconv.Itoa(threadId)

	maintConnStr := fmt.Sprintf("postgres://%s:%s@%s:%d/postgres?sslmode=disable",
		connPool.Username, connPool.Password, connPool.Host, connPool.Port)
	maintConn, err := pgx.Connect(context.Background(), maintConnStr)
	if err != nil {
		return errors.Wrap(err, "[PgConnectorPool.ResetDB]connect to maintenance database error")
	}

	_, _ = maintConn.Exec(context.Background(), fmt.Sprintf("DROP DATABASE IF EXISTS %s", dbName))
	_, err = maintConn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		maintConn.Close(context.Background())
		return errors.Wrap(err, fmt.Sprintf("[PgConnectorPool.ResetDB]create database %s error", dbName))
	}
	maintConn.Close(context.Background())
	return nil
}