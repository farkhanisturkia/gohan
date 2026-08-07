package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getModuleName() string {
	file, err := os.Open("go.mod")
	if err != nil {
		return "myproject"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return "myproject"
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "init":
		initBoilerplate()
	default:
		fmt.Printf("The command '%s' is not recognized.\n", command)
		printHelp()
	}
}

func initBoilerplate() {
	moduleName := getModuleName()

	templates := map[string]string{
		"main.go": fmt.Sprintf(`package main

import (
	"log"

	"github.com/farkhanisturkia/gohan"
	"%s/database/migrations"
)

func main() {
	env := gohan.GetEnv()

	db, err := gohan.GetConn(env)
	if err != nil {
		log.Fatalf("[error] Failed to open database connection: %%v", err)
	}
	defer db.Close()

	migrations.SetMigrations(db)

	SetupRoutes()

	if err := gohan.Serve(&env.AppPort); err != nil {
		log.Fatalf("[error] Server error: %%v", err)
	}
}
`, moduleName),

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
		log.Printf("[error] Failed to migrate the User table: %v\n", err)
	}
}
`,
	}

	for path, content := range templates {
		dir := filepath.Dir(path)
		if dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				fmt.Printf("[error] Failed to create the %s file: %v\n", path, err)
			} else {
				fmt.Printf("[info] %s file created\n", path)
			}
		} else {
			fmt.Printf("[error] The %s file already exists\n", path)
		}
	}
}

func printHelp() {
	fmt.Println("Gohan Framework CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  gohan init   : Generate standar Gohan Framework")
}