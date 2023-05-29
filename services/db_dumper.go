package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

// DatabaseCopy interface to support multiple databases
type DatabaseCopy interface {
	Copy() error
	Dump() error
	ImportDump() error
}

// MySQLCopy
type MySQLCopy struct {
	URL         string
	TargetFile  string
	TargetDBURL string
}

// Copy copies one mysql database to another
func (d MySQLCopy) Copy() error {

	dbHost, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName)
	dumpFile, err := d.dump(connectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	targetConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, "localhost:3306", dbName)
	return d.importDumpFile(targetConnectionString, dumpFile)
}

// Dump creates a dump for MySQL database
func (d MySQLCopy) Dump() error {
	dbHost, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	connectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, dbHost, dbName)
	_, err = d.dump(connectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

// dump creates a dump for MySQL database
func (d MySQLCopy) dump(connectionString string) (string, error) {

	db, err := sql.Open("mysql", connectionString)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	// Open the output file
	file, err := os.OpenFile(d.TargetFile, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer file.Close()

	rows, err := db.Query("SHOW TABLES")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	tables := make([]string, 0)
	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return "", err
		}
		tables = append(tables, tableName)
	}

	// Create a map to store table dependencies
	dependencies := make(map[string][]string)
	for _, tableName := range tables {
		fkRows, err := db.Query(fmt.Sprintf("SELECT REFERENCED_TABLE_NAME FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = '%s' AND REFERENCED_TABLE_NAME IS NOT NULL", tableName))
		if err != nil {
			return "", err
		}
		defer fkRows.Close()

		for fkRows.Next() {
			var refTable string
			err = fkRows.Scan(&refTable)
			if err != nil {
				return "", err
			}
			dependencies[tableName] = append(dependencies[tableName], refTable)
		}
	}

	// Sort tables based on dependencies
	sortedTables, err := topologicalSort(tables, dependencies)
	if err != nil {
		return "", err
	}
	for i := len(sortedTables) - 1; i >= 0; i-- {
		tableName := sortedTables[i]

		createTableStmt, err := db.Query(fmt.Sprintf("SHOW CREATE TABLE %s", tableName))
		if err != nil {
			return "", err
		}
		defer createTableStmt.Close()

		createTableStmt.Next()
		var table, createStmt string
		err = createTableStmt.Scan(&table, &createStmt)
		if err != nil {
			return "", err
		}

		fmt.Fprintln(file, createStmt+";\n")

		dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
		if err != nil {
			return "", err
		}
		defer dataRows.Close()

		columns, err := dataRows.Columns()
		if err != nil {
			return "", err
		}
		values := make([]interface{}, len(columns))
		valuesPtrs := make([]interface{}, len(columns))

		for dataRows.Next() {
			for i := range values {
				valuesPtrs[i] = &values[i]
			}

			err = dataRows.Scan(valuesPtrs...)
			if err != nil {
				return "", err
			}

			var dump strings.Builder

			for i, value := range values {
				if i > 0 {
					dump.WriteString(", ")
				}

				if value == nil {
					dump.WriteString(fmt.Sprintf("%s", "NULL"))

				} else {
					dump.WriteString(fmt.Sprintf("%q", value))
				}
			}

			fmt.Fprintln(file, fmt.Sprintf("INSERT INTO %s(", tableName)+strings.Join(columns, ", ")+") VALUES ("+dump.String()+");\n")

		}
	}

	return d.TargetFile, nil
}

// topologicalSort sorts tables based on their dependencies using Kahn's algorithm
func topologicalSort(tables []string, dependencies map[string][]string) ([]string, error) {
	inDegree := make(map[string]int)
	for _, tableName := range tables {
		inDegree[tableName] = 0
	}

	for _, refs := range dependencies {
		for _, ref := range refs {
			inDegree[ref]++
		}
	}

	queue := make([]string, 0)
	for _, tableName := range tables {
		if inDegree[tableName] == 0 {
			queue = append(queue, tableName)
		}
	}

	sortedTables := make([]string, 0)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		sortedTables = append(sortedTables, current)

		for _, ref := range dependencies[current] {
			inDegree[ref]--
			if inDegree[ref] == 0 {
				queue = append(queue, ref)
			}
		}
	}

	if len(sortedTables) != len(tables) {
		return nil, fmt.Errorf("there is a cyclic dependency between tables")
	}

	return sortedTables, nil
}

// ImportDump imports a dump into MySQL database
func (d MySQLCopy) ImportDump() error {
	_, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	targetConnectionString := fmt.Sprintf("%s:%s@tcp(%s)/%s", dbUser, dbPass, "localhost:3306", dbName)
	return d.importDumpFile(targetConnectionString, d.TargetFile)
}

// importDumpFile imports a dump into MySQL database
func (d MySQLCopy) importDumpFile(targetConnectionString, dumpFile string) error {
	db, err := sql.Open("mysql", targetConnectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return err
	}

	b, err := os.ReadFile(dumpFile) // just pass the file name
	if err != nil {
		return err
	}

	dumpFileContent := string(b) // convert content to a 'string'
	statements := strings.Split(dumpFileContent, ";\n")

	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}

		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}

// PostgresCopy
type PostgresCopy struct {
	URL         string
	TargetFile  string
	TargetDBURL string
}

// Copy copies one postgres database to another
func (d PostgresCopy) Copy() error {

	dbHost, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbUser, dbPass, dbName)

	dumpFile, err := d.dump(connectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	targetConnectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", "localhost:5432", dbUser, dbPass, dbName)

	return d.importDumpFile(targetConnectionString, dumpFile)
}

// Dump creates a dump for PostgreSQL database
func (d PostgresCopy) Dump() error {
	dbHost, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", dbHost, dbUser, dbPass, dbName)

	_, err = d.dump(connectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

// dump creates a dump for PostgreSQL database
func (d PostgresCopy) dump(connectionString string) (string, error) {

	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return "", err
	}

	// Open the output file
	file, err := os.OpenFile(d.TargetFile, os.O_APPEND|os.O_CREATE, 0666)
	if err != nil {
		return "", err
	}
	defer file.Close()

	rows, err := db.Query(`SELECT tablename FROM pg_tables WHERE schemaname='public'`)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		err = rows.Scan(&tableName)
		if err != nil {
			return "", err
		}

		createTableStmt, err := db.Query(fmt.Sprintf("SELECT pg_dump_table_def('%s')", tableName))
		if err != nil {
			return "", err
		}
		defer createTableStmt.Close()

		createTableStmt.Next()
		var createStmt string
		err = createTableStmt.Scan(&createStmt)
		if err != nil {
			return "", err
		}

		fmt.Fprintln(file, createStmt+";\n")

		dataRows, err := db.Query(fmt.Sprintf("SELECT * FROM %s", tableName))
		if err != nil {
			return "", err
		}
		defer dataRows.Close()

		columns, err := dataRows.Columns()
		if err != nil {
			return "", err
		}
		values := make([]interface{}, len(columns))
		valuesPtrs := make([]interface{}, len(columns))

		for dataRows.Next() {
			for i := range values {
				valuesPtrs[i] = &values[i]
			}

			err = dataRows.Scan(valuesPtrs...)
			if err != nil {
				return "", err
			}

			var dump strings.Builder

			for i, value := range values {
				if i > 0 {
					dump.WriteString(", ")
				}
				switch value := value.(type) {
				case int64, float64, bool:
					dump.WriteString(fmt.Sprintf("%v", value))
				default:
					dump.WriteString(fmt.Sprintf("'%v'", value))
				}
			}

			fmt.Fprintln(file, fmt.Sprintf("INSERT INTO %s(", tableName)+strings.Join(columns, ", ")+") VALUES ("+dump.String()+");\n")

		}
	}

	return d.TargetFile, nil
}

// ImportDump imports a dump into PostgreSQL database
func (d PostgresCopy) ImportDump() error {
	_, dbName, dbUser, dbPass, err := GetSQLEnVars(d.URL)
	if err != nil {
		log.Fatal(err)
		return err
	}

	targetConnectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable", "localhost:5432", dbUser, dbPass, dbName)

	return d.importDumpFile(targetConnectionString, d.TargetFile)
}

// importDumpFile imports a dump into PostgreSQL database
func (d PostgresCopy) importDumpFile(targetConnectionString, dump string) error {
	db, err := sql.Open("postgres", targetConnectionString)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
		return err
	}

	statements := strings.Split(dump, ";\n")

	for _, stmt := range statements {
		if strings.TrimSpace(stmt) == "" {
			continue
		}

		_, err := db.Exec(stmt)
		if err != nil {
			return err
		}
	}

	return nil
}
