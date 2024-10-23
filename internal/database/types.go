package database

import "database/sql"

// Config holds all configuration for database connections
type Config struct {
	ConnectionString string
	Host             string
	Port             int
	Database         string
	User             string
	Password         string
	SSHKey           string
	SSHUser          string
	SSHHost          string
	SSHPort          int
	OutputFile       string
}

// TableInfo stores information about a table's structure
type TableInfo struct {
	Name    string
	Columns []ColumnInfo
}

// ColumnInfo stores information about a column's structure
type ColumnInfo struct {
	Name     string
	Type     string
	Nullable bool
	Default  *string
}

// Migrator handles the database migration process
type Migrator struct {
	sourceDB *sql.DB
	cleanup  func()
}
