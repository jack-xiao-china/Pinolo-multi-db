package connector

// SchemaInfo: database schema metadata discovered from a live database
type SchemaInfo struct {
	Tables []TableInfo
}

// TableInfo: metadata for a single database table
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// ColumnInfo: metadata for a single table column
type ColumnInfo struct {
	Name     string // Column name
	Type     string // Database type name, e.g., "int", "varchar(20)", "double", "bigint"
	IsKey    bool   // Whether this column is a primary key or unique key
	Nullable bool   // Whether this column allows NULL values
}

// SchemaDiscoverer: interface for database schema discovery
// Each connector type implements this to query INFORMATION_SCHEMA or catalog tables
type SchemaDiscoverer interface {
	DiscoverSchema() (*SchemaInfo, error)
}

// Ensure Connector implements SchemaDiscoverer
var _ SchemaDiscoverer = (*Connector)(nil)

// Ensure PostgreSQLConnector implements SchemaDiscoverer
var _ SchemaDiscoverer = (*PostgreSQLConnector)(nil)

// Ensure OpenGaussConnector implements SchemaDiscoverer (GaussDB-M)
var _ SchemaDiscoverer = (*OpenGaussConnector)(nil)

// Ensure GaussDBAConnector implements SchemaDiscoverer (GaussDB-A)
var _ SchemaDiscoverer = (*GaussDBAConnector)(nil)

// Ensure UniversalConnector implements SchemaDiscoverer
var _ SchemaDiscoverer = (*UniversalConnector)(nil)