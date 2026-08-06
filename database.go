package gohan

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
	Driver string
}

func GetConn(e *Env) (*DB, error) {
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
		dbType := getSQLType(field.Type.Kind(), db.Driver)
		constraints := ""

		if gohanTag != "" {
			tagParts := strings.Split(gohanTag, ";")
			for _, part := range tagParts {
				part = strings.TrimSpace(part)
				if strings.HasPrefix(part, "type:") {
					dbType = strings.TrimPrefix(part, "type:")
				} else if part == "primary_key" {
					if db.Driver == "sqlite3" {
						constraints += " PRIMARY KEY AUTOINCREMENT"
					} else if db.Driver == "mysql" {
						constraints += " PRIMARY KEY AUTO_INCREMENT"
					} else if db.Driver == "postgres" {
						dbType = "SERIAL"
						constraints += " PRIMARY KEY"
					}
				} else if part == "not_null" {
					constraints += " NOT NULL"
				} else if part == "unique" {
					constraints += " UNIQUE"
				}
			}
		}

		colDef := fmt.Sprintf("%s %s%s", colName, dbType, constraints)
		columns = append(columns, colDef)
	}

	var query string
	switch db.Driver {
	case "mysql":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s);", tableName, strings.Join(columns, ", "))
	case "postgres":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (%s);", tableName, strings.Join(columns, ", "))
	default:
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, strings.Join(columns, ", "))
	}

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("[error] Failed to create the '%s' table: %w", tableName, err)
	}

	log.Printf("[info] The '%s' table is ready to use (created/verified).\n", tableName)
	return nil
}

func getSQLType(kind reflect.Kind, driver string) string {
	switch kind {
	case reflect.Int, reflect.Int64:
		return "INTEGER"
	case reflect.String:
		return "TEXT"
	case reflect.Float32, reflect.Float64:
		return "REAL"
	case reflect.Bool:
		return "BOOLEAN"
	default:
		return "TEXT"
	}
}

func (db *DB) Create(model interface{}) error {
	val := reflect.ValueOf(model)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return fmt.Errorf("[error] Create memerlukan struct atau pointer ke struct")
	}

	typ := val.Type()
	tableName := strings.ToLower(typ.Name()) + "s"

	var columns []string
	var placeholders []string
	var values []interface{}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldValue := val.Field(i)
		gohanTag := field.Tag.Get("gohan")

		if gohanTag == "-" {
			continue
		}

		if strings.Contains(gohanTag, "primary_key") && (fieldValue.Kind() == reflect.Int || fieldValue.Kind() == reflect.Int64) && fieldValue.Int() == 0 {
			continue
		}

		colName := strings.ToLower(field.Name)
		columns = append(columns, colName)

		if db.Driver == "postgres" {
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(values)+1))
		} else {
			placeholders = append(placeholders, "?")
		}

		values = append(values, fieldValue.Interface())
	}

	var query string
	switch db.Driver {
	case "mysql":
		query = fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	case "postgres":
		query = fmt.Sprintf("INSERT INTO \"%s\" (%s) VALUES (%s)", tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	default:
		query = fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", tableName, strings.Join(columns, ", "), strings.Join(placeholders, ", "))
	}

	_, err := db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("[error] gagal insert data ke '%s': %w", tableName, err)
	}

	log.Printf("[info] Data berhasil disimpan ke tabel '%s'\n", tableName)
	return nil
}