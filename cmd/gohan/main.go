package main

import (
	"fmt"
	"os"

	"github.com/farkhanisturkia/gohan/cmd/gohan/commands"
)

const AppVersion = "v1.5.7"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	arg := os.Args[1]

	switch arg {
	case "init":
		commands.InitBoilerplate()

	case "key:generate":
        commands.GenerateAppKey()

	case "make:controller":
		if len(os.Args) < 3 {
			fmt.Println("[error] Controller name is required. Example: gohan make:controller user_controller")
			os.Exit(1)
		}
		commands.MakeController(os.Args[2])

	case "make:migration":
		if len(os.Args) < 3 {
			fmt.Println("[error] Migration name is required. Example: gohan make:migration create_user_table")
			os.Exit(1)
		}
		commands.MakeMigration(os.Args[2])

	case "make:seeder":
		if len(os.Args) < 3 {
			fmt.Println("[error] Seeder name is required. Example: gohan make:seeder user_seeder")
			os.Exit(1)
		}
		commands.MakeSeeder(os.Args[2])

	case "-v", "--version":
		printVersion()

	case "-h", "--help":
		printHelp()

	default:
		fmt.Printf("[error] The command '%s' is not recognized.\n", arg)
		printHelp()
		os.Exit(1)
	}
}

func printVersion() {
	fmt.Printf("Gohan CLI version %s\n", AppVersion)
}

func printHelp() {
	fmt.Println("Gohan Framework CLI Generator")
	fmt.Println("\nUsage:")
	fmt.Println("  gohan <command>/<flags>")
	fmt.Println("\nAvailable Commands:")
	fmt.Println("  init			Generate standard Gohan Framework")
	fmt.Println("  key:generate		Generate a new application encryption key")
	fmt.Println("  make:controller	Generate a new controller file")
	fmt.Println("  make:migration	Generate a new migration file")
	fmt.Println("  make:seeder		Generate a new seeder file")
	fmt.Println("\nFlags:")
	fmt.Println("  -v, --version		Show the CLI version")
	fmt.Println("  -h, --help		Display the CLI usage instructions")
}