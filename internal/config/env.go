package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Env struct {
	AppName    string
	AppEnv     string
	AppKey     string
	AppPort    string
	DBDriver   string
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBFile     string
}

func InitEnv() error {
	if _, err := os.Stat(".env"); err == nil {
		return nil
	}

	defaultContent := `# App configuration
APP_NAME=GohanApp
APP_ENV=local
APP_KEY=
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

	log.Println("[info] .env file has been created automatically!")
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
		AppName:	getEnvVal("APP_NAME", "GohanApp"),
		AppEnv:		getEnvVal("APP_ENV", "local"),
		AppKey:		getEnvVal("APP_KEY", ""),
		DBDriver:   getEnvVal("DB_DRIVER", "sqlite"),
		DBHost:     getEnvVal("DB_HOST", "127.0.0.1"),
		DBPort:     getEnvVal("DB_PORT", "3306"),
		DBUser:     getEnvVal("DB_USER", "root"),
		DBPassword: getEnvVal("DB_PASSWORD", ""),
		DBName:     getEnvVal("DB_NAME", "gohan_db"),
		DBFile:     getEnvVal("DB_FILE", "gohan.db"),
	}
}

func getEnvVal(key, fallback string) string {
	if val, exists := os.LookupEnv(key); exists && val != "" {
		return val
	}
	return fallback
}