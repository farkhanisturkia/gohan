package database

import (
	"database/sql"
	"fmt"
	"log"
	"reflect"
	"regexp"
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

type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

func (db *DB) execer() Execer {
	return db.DB
}

func (db *DB) driver() string {
	return db.Driver
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
	return setTable(db, model)
}

func setTable(c Commander, model interface{}) error {
    val := reflect.ValueOf(model)
    if val.Kind() == reflect.Ptr {
        val = val.Elem()
    }

    if val.Kind() != reflect.Struct {
        return fmt.Errorf("[error] SetTable requires a struct or a pointer to a struct")
    }

    typ := val.Type()
    tableName := toSnakeCase(typ.Name()) + "s"

    var columns []string

    for i := 0; i < typ.NumField(); i++ {
        field := typ.Field(i)

        if !field.IsExported() || field.Tag.Get("gohan") == "-" {
            continue
        }

        gohanTag := field.Tag.Get("gohan")
        colName := getColumnName(field)
        dbType := getSQLType(field.Type, c.driver())
        constraints := ""
        isPrimaryKey := false

        if gohanTag != "" {
            tagParts := strings.Split(gohanTag, ";")
            for _, part := range tagParts {
                part = strings.TrimSpace(part)
                if strings.HasPrefix(part, "type:") {
                    dbType = strings.TrimSpace(strings.TrimPrefix(part, "type:"))
                } else if part == "primary_key" {
                    isPrimaryKey = true
                } else if part == "not_null" {
                    constraints += " NOT NULL"
                } else if part == "unique" {
                    constraints += " UNIQUE"
                } else if strings.HasPrefix(part, "default:") {
                    defaultValue := strings.TrimSpace(strings.TrimPrefix(part, "default:"))
                    constraints += " DEFAULT " + defaultValue
                }
            }
        }

        if c.driver() == "postgres" && strings.EqualFold(dbType, "DATETIME") {
            dbType = "TIMESTAMP"
        }

        if isPrimaryKey {
            switch c.driver() {
            case "sqlite3", "sqlite":
                dbType = "INTEGER"
                constraints = " PRIMARY KEY AUTOINCREMENT"

            case "mysql":
                if dbType == "" {
                    dbType = "INT"
                }
                constraints = " PRIMARY KEY AUTO_INCREMENT"

            case "postgres":
                upperType := strings.ToUpper(dbType)
                if upperType == "INT" || upperType == "INTEGER" || upperType == "" {
                    dbType = "SERIAL"
                } else if upperType == "BIGINT" {
                    dbType = "BIGSERIAL"
                }
                constraints = " PRIMARY KEY"
            }
        }

        colDef := fmt.Sprintf("%s %s%s", quoteIdentifier(colName, c.driver()), dbType, constraints)
        columns = append(columns, colDef)
    }

    quotedTable := quoteIdentifier(tableName, c.driver())
    query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", quotedTable, strings.Join(columns, ", "))

    _, err := c.execer().Exec(query)
    if err != nil {
        return fmt.Errorf("[error] Failed to create the '%s' table: %w", tableName, err)
    }

    log.Printf("[info] The '%s' table is ready to use (created/verified).\n", tableName)
    return nil
}
func getSQLType(t reflect.Type, driver string) string {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

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

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap = regexp.MustCompile("([a-z0-9])([A-Z])")

func toSnakeCase(str string) string {
	snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

func getColumnName(field reflect.StructField) string {
	gohanTag := field.Tag.Get("gohan")
	if gohanTag != "" {
		for _, part := range strings.Split(gohanTag, ";") {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "column:") {
				return strings.TrimPrefix(part, "column:")
			}
		}
	}
	return toSnakeCase(field.Name)
}