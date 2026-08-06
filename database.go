package gohan

import (
	"database/sql"
	"fmt"
	"log"

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