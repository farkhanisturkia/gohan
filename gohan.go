package gohan

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

type Env struct {
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

	defaultContent := `# Choose DB_DRIVER: mysql | postgres | sqlite
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
		DBDriver:   getEnvVal("DB_DRIVER", "sqlite"),
		DBHost:     getEnvVal("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvVal("DB_PORT", "3306"),
		DBUser:     getEnvVal("DB_USER", "root"),
		DBPassword: getEnvVal("DB_PASSWORD", ""),
		DBName:     getEnvVal("DB_NAME", "gohan_db"),
		DBFile:     getEnvVal("DB_FILE", "gohan.db"),
	}
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

func (db *DB) SetTable(tableName string, schema string) error {
	var query string

	switch db.Driver {
	case "mysql":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (%s);", tableName, schema)
	case "postgres":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS \"%s\" (%s);", tableName, schema)
	case "sqlite3":
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, schema)
	default:
		query = fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (%s);", tableName, schema)
	}

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("[error] Failed to create the '%s' table: %w", tableName, err)
	}

	log.Printf("[info] The '%s' table is ready to use (created/verified).\n", tableName)
	return nil
}

func getEnvVal(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return fallback
}