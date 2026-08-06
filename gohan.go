package gohan

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"reflect"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Env struct {
	AppPort    string
	DBDriver   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBFile     string
}

type DB struct {
	*sql.DB
	Driver string
}

func InitEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		return nil
	}

	defaultContent := `# App configuration
APP_PORT=8080
	
# Choose DB_DRIVER: mysql | postgres | sqlite
DB_DRIVER=sqlite

# MySQL / Postgres Configuration
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=gohan_db

# SQLite Configuration
DB_FILE=gohan.db
`
	err := os.WriteFile(".env", []byte(defaultContent), 0644)
	if err != nil {
		return fmt.Errorf("[error] Failed to create .env file: %w", err)
	}

	log.Println("[info] .env file has beed created automatically!")
	return nil
}

func GetEnv() *Env {
	_ = InitEnv()

	err := godotenv.Load()
	if err != nil {
		log.Println("[warning] Unable to load .env, using OS default environment variables")
	}

	return &Env{
		AppPort:    getEnvVal("APP_PORT", "8080"),
		DBDriver:   getEnvVal("DB_DRIVER", "sqlite"),
		DBHost:     getEnvVal("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvVal("DB_PORT", "3306"),
		DBUser:     getEnvVal("DB_USER", "root"),
		DBPassword: getEnvVal("DB_PASSWORD", ""),
		DBName:     getEnvVal("DB_NAME", "gohan_db"),
		DBFile:     getEnvVal("DB_FILE", "gohan.db"),
	}
}

func Serve(port interface{}) error {
	var portStr string

	switch v := port.(type) {
	case string:
		portStr = v
	case *string:
		if v != nil {
			portStr = *v
		} else {
			portStr = "8080"
		}
	default:
		portStr = fmt.Sprintf("%v", port)
	}

	if !strings.HasPrefix(portStr, ":") {
		portStr = ":" + portStr
	}

	log.Printf("[info] Server running in http://localhost%s\n", portStr)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "Welcome to Gohan Framework!"}`))
	})

	return http.ListenAndServe(portStr, mux)
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

func getEnvVal(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return fallback
}