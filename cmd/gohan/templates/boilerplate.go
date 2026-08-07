package templates

import "fmt"

func GetBoilerplateTemplates(moduleName string) map[string]string {
	return map[string]string{
		"main.go": fmt.Sprintf(`package main

import (
	"log"

	"github.com/farkhanisturkia/gohan"
	"%s/database/migrations"
	"%s/database/seeders"
)

func main() {
	env := gohan.GetEnv()

	db, err := gohan.GetConn(env)
	if err != nil {
		log.Fatalf("[error] Failed to open database connection: %%v", err)
	}
	defer db.Close()

	migrations.SetMigrations(db)
	seeders.SetSeeders(db)

	SetupRoutes()

	if err := gohan.Serve(&env.AppPort); err != nil {
		log.Fatalf("[error] Server error: %%v", err)
	}
}
`, moduleName, moduleName),

		"routes.go": `package main

import (
	"net/http"

	"github.com/farkhanisturkia/gohan"
)

func SetupRoutes() {
	gohan.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		gohan.JSON(w, http.StatusOK, map[string]string{
			"message": "pong",
		})
	})
}
`,

		"database/migrations/default.go": `package migrations

import "github.com/farkhanisturkia/gohan"

func SetMigrations(db *gohan.DB) {
	UserMigration(db)
}
`,

		"database/migrations/user.go": `package migrations

import (
	"log"

	"github.com/farkhanisturkia/gohan"
)

type User struct {
	ID       int    ` + "`" + `gohan:"primary_key" json:"id"` + "`" + `
	Name     string ` + "`" + `gohan:"type:VARCHAR(100);not_null" json:"name"` + "`" + `
	Email    string ` + "`" + `gohan:"type:VARCHAR(150);unique;not_null" json:"email"` + "`" + `
	Password string ` + "`" + `gohan:"type:VARCHAR(150);not_null" json:"-"` + "`" + `
}

func UserMigration(db *gohan.DB) {
	if err := db.SetTable(&User{}); err != nil {
		log.Printf("[error] Failed to migrate the User table: %%v\n", err)
	}
}
`,

		"database/seeders/default.go": `package seeders

import "github.com/farkhanisturkia/gohan"

func SetSeeders(db *gohan.DB) {
	// UserSeeder(db)
}
`,

		"database/seeders/user.go": fmt.Sprintf(`package seeders

import (
	"log"

	"github.com/farkhanisturkia/gohan"
	"golang.org/x/crypto/bcrypt"
	"%s/database/migrations"
)

func UserSeeder(db *gohan.DB) {
	var existingUsers []migrations.User
	if err := db.FindAll(&existingUsers); err == nil && len(existingUsers) > 0 {
		log.Println("[info] User seeder skipped: Data already exists")
		return
	}

	plainPassword := "password"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[error] Failed to hash password: %%v\n", err)
	}

	users := []migrations.User{
		{
			Name:     "Admin",
			Email:    "admin@example.com",
			Password: string(hashedPassword),
		},
	}

	for _, user := range users {
		if err := db.Create(&user); err != nil {
			log.Printf("[error] Failed to seed user (%%s): %%v\n", user.Email, err)
		}
	}

	log.Println("[info] User seeder successfully executed")
}
`, moduleName),

		".env": `# App configuration
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
`,
	}
}