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

	SetupRoutes(db)

	if err := gohan.Serve(&env.AppPort); err != nil {
		log.Fatalf("[error] Server error: %%v", err)
	}
}
`, moduleName, moduleName),

		"routes.go": fmt.Sprintf(`package main

import (
	"net/http"

	"github.com/farkhanisturkia/gohan"
	"%s/middleware"
)

func SetupRoutes(db *gohan.DB) {
	gohan.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		gohan.JSON(w, http.StatusOK, map[string]string{
			"message": "pong",
		})
	})

	gohan.Get("/api/me", middleware.Authenticate(db, func(w http.ResponseWriter, r *http.Request) {
		user, ok := middleware.GetAuthUser(r)
		if !ok {
			gohan.JSON(w, http.StatusInternalServerError, map[string]string{
				"message": "Failed to get user context",
			})
			return
		}

		gohan.JSON(w, http.StatusOK, map[string]interface{}{
			"message": "User profile retrieved successfully",
			"data":    user,
		})
	}))
}
`, moduleName),

		"database/migrations/default.go": `package migrations

import "github.com/farkhanisturkia/gohan"

func SetMigrations(db *gohan.DB) {
	UserMigration(db)
	PersonalAccessTokenMigration(db)
	PasswordResetMigration(db)
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

		"database/migrations/personal_access_token.go": `package migrations

import (
	"log"
	"time"

	"github.com/farkhanisturkia/gohan"
)

type PersonalAccessToken struct {
	ID        int       ` + "`" + `gohan:"primary_key" json:"id"` + "`" + `
	UserID    int       ` + "`" + `gohan:"type:INT;not_null" json:"user_id"` + "`" + `
	Name      string    ` + "`" + `gohan:"type:VARCHAR(100);not_null" json:"name"` + "`" + `
	Token     string    ` + "`" + `gohan:"type:VARCHAR(255);unique;not_null" json:"token"` + "`" + `
	IPAddress string    ` + "`" + `gohan:"type:VARCHAR(45)" json:"ip_address"` + "`" + `
	UserAgent string    ` + "`" + `gohan:"type:TEXT" json:"user_agent"` + "`" + `
	ExpiresAt time.Time ` + "`" + `gohan:"type:DATETIME;not_null" json:"expires_at"` + "`" + `
	CreatedAt time.Time ` + "`" + `gohan:"type:DATETIME;not_null" json:"created_at"` + "`" + `
}

func PersonalAccessTokenMigration(db *gohan.DB) {
	if err := db.SetTable(&PersonalAccessToken{}); err != nil {
		log.Printf("[error] Failed to migrate PersonalAccessToken table: %%v\n", err)
	}
}
`,

		"database/migrations/password_reset.go": `package migrations

import (
	"log"
	"time"

	"github.com/farkhanisturkia/gohan"
)

type PasswordReset struct {
    ID        int       ` + "`" + `gohan:"primary_key" json:"id"` + "`" + `
    Email     string    ` + "`" + `gohan:"type:VARCHAR(150);not_null" json:"email"` + "`" + `
    Token     string    ` + "`" + `gohan:"type:VARCHAR(255);not_null" json:"token"` + "`" + `
    CreatedAt time.Time ` + "`" + `gohan:"type:DATETIME;not_null" json:"created_at"` + "`" + `
}

func PasswordResetMigration(db *gohan.DB) {
	if err := db.SetTable(&PasswordReset{}); err != nil {
		log.Printf("[error] Failed to migrate PasswordReset table: %%v\n", err)
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

		"middleware/auth.go": fmt.Sprintf(`package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/farkhanisturkia/gohan"
	"%s/database/migrations"
)

type contextKey string

const UserContextKey contextKey = "authenticated_user"

func Authenticate(db *gohan.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			gohan.JSON(w, http.StatusUnauthorized, map[string]string{
				"message": "Authorization header is required",
			})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			gohan.JSON(w, http.StatusUnauthorized, map[string]string{
				"message": "Invalid authorization format. Expected 'Bearer <token>'",
			})
			return
		}

		tokenString := parts[1]

		var tokenData migrations.PersonalAccessToken
		err := db.FindOne(&tokenData, "token = ?", tokenString)
		if err != nil {
			gohan.JSON(w, http.StatusUnauthorized, map[string]string{
				"message": "Invalid or expired token",
			})
			return
		}

		if time.Now().After(tokenData.ExpiresAt) {
			gohan.JSON(w, http.StatusUnauthorized, map[string]string{
				"message": "Token has expired",
			})
			return
		}

		var user migrations.User
		err = db.FindOne(&user, "id = ?", tokenData.UserID)
		if err != nil {
			gohan.JSON(w, http.StatusUnauthorized, map[string]string{
				"message": "User associated with this token not found",
			})
			return
		}

		ctx := context.WithValue(r.Context(), UserContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

func GetAuthUser(r *http.Request) (migrations.User, bool) {
	user, ok := r.Context().Value(UserContextKey).(migrations.User)
	return user, ok
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