package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

// Migrate performs the migration from PostgreSQL to SQLite
func (m *Migrator) Migrate(sqliteFile string, withData bool) error {
	// Remove existing SQLite file if it exists
	os.Remove(sqliteFile)

	// Connect to SQLite
	destDB, err := sql.Open("sqlite3", sqliteFile)
	if err != nil {
		return fmt.Errorf("failed to create SQLite database: %w", err)
	}
	defer destDB.Close()

	// Enable foreign keys
	if _, err := destDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Get all tables
	tables, err := m.GetTables()
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	// Begin transaction for schema creation
	tx, err := destDB.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create tables in SQLite
	for _, table := range tables {
		if err := m.createTable(tx, table); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to create table %s: %w", table.Name, err)
		}
		fmt.Printf("Created table: %s\n", table.Name)
	}

	// Commit schema changes
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit schema changes: %w", err)
	}

	// Migrate data if requested
	if withData {
		for _, table := range tables {
			if err := m.migrateData(destDB, table); err != nil {
				return fmt.Errorf("failed to migrate data for table %s: %w", table.Name, err)
			}
			fmt.Printf("Migrated data for table: %s\n", table.Name)
		}
	}

	// Vacuum the database to optimize storage
	if _, err := destDB.Exec("VACUUM"); err != nil {
		return fmt.Errorf("failed to vacuum database: %w", err)
	}

	return nil
}

func (m *Migrator) createTable(tx *sql.Tx, table TableInfo) error {
	var columnDefs []string
	var constraints []string

	// Process columns and collect constraints
	for _, col := range table.Columns {
		def := fmt.Sprintf("\"%s\" %s", col.Name, mapPostgreSQLTypeToSQLite(col.Type))

		if !col.Nullable {
			def += " NOT NULL"
		}

		if col.Default != nil {
			// Handle special PostgreSQL default values
			defaultVal := *col.Default
			switch {
			case strings.HasPrefix(defaultVal, "nextval"):
				def += " PRIMARY KEY AUTOINCREMENT"
			case defaultVal == "CURRENT_TIMESTAMP":
				def += " DEFAULT CURRENT_TIMESTAMP"
			case defaultVal == "now()":
				def += " DEFAULT CURRENT_TIMESTAMP"
			case defaultVal == "true":
				def += " DEFAULT 1"
			case defaultVal == "false":
				def += " DEFAULT 0"
			// if defaultVal has :: then it is a type cast, so we need to remove it
			case strings.Contains(defaultVal, "::"):
				def += fmt.Sprintf(" DEFAULT %s", strings.Split(defaultVal, "::")[0])
			default:
				def += fmt.Sprintf(" DEFAULT %s", defaultVal)
			}
		}

		columnDefs = append(columnDefs, def)
	}

	// Create table query
	query := fmt.Sprintf("CREATE TABLE \"%s\" (\n\t%s%s\n)",
		table.Name,
		strings.Join(columnDefs, ",\n\t"),
		func() string {
			if len(constraints) > 0 {
				return ",\n\t" + strings.Join(constraints, ",\n\t")
			}
			return ""
		}())

	//log.Printf("Creating table: %s\n", table.Name)
	//log.Printf("Query: %s\n", query)

	// Execute the create table query
	if _, err := tx.Exec(query); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Create indexes for the table
	// if err := m.createIndexes(tx, table.Name); err != nil {
	// 	return fmt.Errorf("failed to create indexes: %w", err)
	// }

	return nil
}

func (m *Migrator) createIndexes(tx *sql.Tx, tableName string) error {
	// Query to get indexes from PostgreSQL
	rows, err := m.sourceDB.Query(`
		SELECT
			i.relname as index_name,
			array_agg(a.attname) as column_names,
			ix.indisunique as is_unique
		FROM
			pg_class t,
			pg_class i,
			pg_index ix,
			pg_attribute a
		WHERE
			t.oid = ix.indrelid
			AND i.oid = ix.indexrelid
			AND a.attrelid = t.oid
			AND a.attnum = ANY(ix.indkey)
			AND t.relkind = 'r'
			AND t.relname = $1
		GROUP BY
			i.relname,
			ix.indisunique
		ORDER BY
			i.relname;
	`, tableName)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var indexName string
		var columnNames []string
		var isUnique bool

		if err := rows.Scan(&indexName, &columnNames, &isUnique); err != nil {
			return err
		}

		// Skip primary key indexes as they're handled differently in SQLite
		if strings.HasSuffix(indexName, "_pkey") {
			continue
		}

		// Create the index
		createIndexSQL := fmt.Sprintf("CREATE %s INDEX %s ON %s (%s)",
			func() string {
				if isUnique {
					return "UNIQUE"
				}
				return ""
			}(),
			indexName,
			tableName,
			strings.Join(columnNames, ", "),
		)

		if _, err := tx.Exec(createIndexSQL); err != nil {
			return fmt.Errorf("failed to create index %s: %w", indexName, err)
		}
	}

	return nil
}

func (m *Migrator) migrateData(destDB *sql.DB, table TableInfo) error {
	// Begin transaction for data migration
	tx, err := destDB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Get data from source table
	rows, err := m.sourceDB.Query(fmt.Sprintf("SELECT * FROM %s", table.Name))
	if err != nil {
		return err
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	// Prepare the INSERT statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	// surround column names with quotes
	for i, col := range columns {
		columns[i] = fmt.Sprintf("\"%s\"", col)
	}

	insertStmt, err := tx.Prepare(fmt.Sprintf(
		"INSERT INTO \"%s\" (%s) VALUES (%s)",
		table.Name,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	))
	if err != nil {
		return err
	}
	defer insertStmt.Close()

	// Prepare value holders
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Insert data in batches
	batchSize := 1000
	count := 0

	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return err
		}

		// Convert PostgreSQL-specific types to SQLite compatible values
		for i, v := range values {
			values[i] = convertValue(v)
		}

		_, err = insertStmt.Exec(values...)
		if err != nil {
			return err
		}

		count++
		if count%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return err
			}
			tx, err = destDB.Begin()
			if err != nil {
				return err
			}
			insertStmt, err = tx.Prepare(fmt.Sprintf(
				"INSERT INTO %s (%s) VALUES (%s)",
				table.Name,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
			))
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func mapPostgreSQLTypeToSQLite(pgType string) string {
	switch strings.ToLower(pgType) {
	case "integer", "smallint", "bigint", "serial", "bigserial":
		return "INTEGER"
	case "real", "double precision", "numeric", "decimal", "money":
		return "REAL"
	case "boolean":
		return "BOOLEAN"
	case "timestamp", "timestamp without time zone", "timestamp with time zone":
		return "DATETIME"
	case "date":
		return "DATE"
	case "time", "time without time zone", "time with time zone":
		return "TEXT"
	case "bytea":
		return "BLOB"
	case "json", "jsonb":
		return "TEXT"
	case "uuid":
		return "TEXT"
	case "inet", "cidr":
		return "TEXT"
	case "point", "line", "polygon", "box":
		return "TEXT"
	default:
		return "TEXT"
	}
}

func convertValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	switch val := v.(type) {
	case []byte:
		// Try to convert bytea to string if it's not actually binary data
		if isUTF8(val) {
			return string(val)
		}
		return val
	case bool:
		if val {
			return 1
		}
		return 0
	default:
		return val
	}
}

func isUTF8(b []byte) bool {
	return strings.Contains(string(b), "\x00") == false
}
