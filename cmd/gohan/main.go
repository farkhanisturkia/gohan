package main

import (
	"fmt"
	"os"
)

const routesTemplate = `package main

import "github.com/farkhanisturkia/gohan"

func SetupRoutes(app *gohan.Gohan) {
	app.Get("/ping", func(req *gohan.Request, res *gohan.Response) {
		res.JSON(200, map[string]string{
			"message": "pong",
		})
	})
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
		createRoutesFile()
	default:
		fmt.Printf("The command '%s' is not recognized.\n", command)
		printHelp()
	}
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