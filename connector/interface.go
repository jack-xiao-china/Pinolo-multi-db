package connector

// SQLExecutor: common interface for all database connectors
// Allows stage1 and stage2 to work with any database type
type SQLExecutor interface {
	ExecSQL(sql string) *Result
}

// Ensure Connector implements SQLExecutor
var _ SQLExecutor = (*Connector)(nil)

// Ensure OpenGaussConnector implements SQLExecutor (M mode)
var _ SQLExecutor = (*OpenGaussConnector)(nil)

// Ensure UniversalConnector implements SQLExecutor
var _ SQLExecutor = (*UniversalConnector)(nil)

// Ensure GaussDBAConnector implements SQLExecutor (A mode)
var _ SQLExecutor = (*GaussDBAConnector)(nil)

// Ensure PostgreSQLConnector implements SQLExecutor
var _ SQLExecutor = (*PostgreSQLConnector)(nil)