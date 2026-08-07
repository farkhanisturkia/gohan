package main

import (
	"fmt"
	"os"
)

const routesTemplate = `package main

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
`

const mainTemplate = `package main

import (
	"log"

	"github.com/farkhanisturkia/gohan"
)

func main() {
	env := gohan.GetEnv()

	SetupRoutes()

	if err := gohan.Serve(&env.AppPort); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
`

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]

	switch command {
	case "init":
		createMainFile()
		createRoutesFile()
	default:
		fmt.Printf("The command '%s' is not recognized.\n", command)
		printHelp()
	}
}

func createMainFile() {
	filePath := "main.go"

	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("[info] The 'main.go' file already exists in this directory.")
		return
	}

	err := os.WriteFile(filePath, []byte(mainTemplate), 0644)
	if err != nil {
		fmt.Printf("[error] Failed to create the 'main.go' file: %v\n", err)
		return
	}

	fmt.Println("[info] The 'main.go' file was successfully created!")
}

func createRoutesFile() {
	filePath := "routes.go"

	if _, err := os.Stat(filePath); err == nil {
		fmt.Println("[info] The 'routes.go' file already exists in this directory.")
		return
	}

	err := os.WriteFile(filePath, []byte(routesTemplate), 0644)
	if err != nil {
		fmt.Printf("[error] Failed to create the 'routes.go' file: %v\n", err)
		return
	}

	fmt.Println("[info] The 'routes.go' file was successfully created!")
}

func printHelp() {
	fmt.Println("Gohan Framework CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  gohan init   : Generate standar Gohan Framework")
}