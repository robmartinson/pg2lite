package database

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
)

// NewMigrator creates a new database migrator
func NewMigrator(config Config) (*Migrator, error) {
	var connStr string
	var cleanup func()
	var err error

	if config.ConnectionString != "" {
		connStr = config.ConnectionString
	} else if config.SSHKey != "" {
		connStr, cleanup, err = SetupTunnel(config)
		if err != nil {
			return nil, fmt.Errorf("failed to setup SSH tunnel: %w", err)
		}
	} else {
		connStr = fmt.Sprintf(
			"host=%s port=%d dbname=%s user=%s",
			config.Host,
			config.Port,
			config.Database,
			config.User,
		)
		if config.Password != "" {
			connStr += fmt.Sprintf(" password=%s", config.Password)
		}
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		if cleanup != nil {
			cleanup()
		}
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	return &Migrator{
		sourceDB: db,
		cleanup:  cleanup,
	}, nil
}

// Close closes the database connections and cleans up resources
func (m *Migrator) Close() {
	if m.sourceDB != nil {
		m.sourceDB.Close()
	}
	if m.cleanup != nil {
		m.cleanup()
	}
}

// GetTables returns all tables in the database
func (m *Migrator) GetTables() ([]TableInfo, error) {
	var tables []TableInfo

	rows, err := m.sourceDB.Query(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE' ORDER BY table_name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}

		columns, err := m.getColumns(tableName)
		if err != nil {
			return nil, err
		}

		tables = append(tables, TableInfo{
			Name:    tableName,
			Columns: columns,
		})
	}

	return tables, nil
}

func (m *Migrator) getColumns(tableName string) ([]ColumnInfo, error) {
	var columns []ColumnInfo

	rows, err := m.sourceDB.Query(`
		SELECT column_name, data_type, is_nullable, column_default
		FROM information_schema.columns
		WHERE table_schema = 'public'
		AND table_name = $1
		ORDER BY ordinal_position
	`, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var col ColumnInfo
		var isNullable string
		var defaultValue sql.NullString

		if err := rows.Scan(&col.Name, &col.Type, &isNullable, &defaultValue); err != nil {
			return nil, err
		}

		col.Nullable = isNullable == "YES"
		if defaultValue.Valid {
			col.Default = &defaultValue.String
		}

		columns = append(columns, col)
	}

	return columns, nil
}
