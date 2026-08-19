package database

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/farkhanisturkia/gohan/internal/config"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	Driver string
}

func GetConn(e *config.Env) (*DB, error) {
	var dsn string
	driver := e.DBDriver

	switch driver {
	case "mysql":
		dsn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
			e.DBUser, e.DBPassword, e.DBHost, e.DBPort, e.DBName)

	case "postgres", "postgresql":
		driver = "postgres"
		dsn = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
			e.DBHost, e.DBPort, e.DBUser, e.DBPassword, e.DBName)

	case "sqlite", "sqlite3":
		driver = "sqlite3"
		dsn = e.DBFile

	default:
		return nil, fmt.Errorf("[error] The '%s' driver is not supported", e.DBDriver)
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("[error] Failed to initialize the %s driver: %w", driver, err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("[error] Failed to connect to the %s database: %w", driver, err)
	}

	log.Printf("[info] Successfully connected to the database (%s)\n", driver)

	return &DB{
		DB:     db,
		Driver: driver,
	}, nil
}

func (db *DB) SetTable(model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("[error] SetTable requires a struct or a pointer to a struct")
	}

	typ := val.Type()
	tableName := strings.ToLower(typ.Name()) + "s"

	var columns []string

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		colName := strings.ToLower(field.Name)
		dbType := getSQLType(field.Type, db.Driver)
		constraints := ""
		isPrimaryKey := false

		if gohanTag != "" {
			tagParts := strings.Split(gohanTag, ";")
			for _, part := range tagParts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "type:") {
					dbType = strings.TrimPrefix(part, "type:")
				} else if part == "primary_key" {
					isPrimaryKey = true
				} else if part == "not_null" {
					constraints += " NOT NULL"
				} else if part == "unique" {
					constraints += " UNIQUE"
				}
			}
		}

		if db.Driver == "postgres" && strings.ToUpper(dbType) == "DATETIME" {
			dbType = "TIMESTAMP"
		}

		if isPrimaryKey {
			if db.Driver == "sqlite3" {
				constraints = " PRIMARY KEY AUTOINCREMENT" + constraints
			} else if db.Driver == "mysql" {
				constraints = " PRIMARY KEY AUTO_INCREMENT" + constraints
			} else if db.Driver == "postgres" {
				dbType = "SERIAL"
				constraints = " PRIMARY KEY" + constraints
			}
		}

		colDef := fmt.Sprintf("%s %s%s", db.quoteIdentifier(colName), dbType, constraints)
		columns = append(columns, colDef)
	}

	quotedTable := db.quoteIdentifier(tableName)
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", quotedTable, strings.Join(columns, ", "))

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("[error] Failed to create the '%s' table: %w", tableName, err)
	}

	log.Printf("[info] The '%s' table is ready to use (created/verified).\n", tableName)
	return nil
}

func getSQLType(t reflect.Type, driver string) string {
	if t.String() == "time.Time" {
		if driver == "postgres" {
			return "TIMESTAMP"
		}
		return "DATETIME"
	}

	switch t.Kind() {
	case reflect.Int, reflect.Int32, reflect.Int64:
		return "INT"
	case reflect.String:
		return "VARCHAR(255)"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func (db *DB) quoteIdentifier(name string) string {
	switch db.Driver {
	case "mysql":
		return fmt.Sprintf("`%s`", name)
	case "postgres":
		return fmt.Sprintf("\"%s\"", name)
	default:
		return name
	}
}

func (db *DB) getColumns(tableName string) ([]string, error) {
	query := fmt.Sprintf("SELECT * FROM %s LIMIT 1", db.quoteIdentifier(tableName))
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return rows.Columns()
}