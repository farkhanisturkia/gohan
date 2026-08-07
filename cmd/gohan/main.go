package main

import (
	"fmt"
	"os"

	"github.com/farkhanisturkia/gohan/cmd/gohan/commands"
)

const AppVersion = "v1.1.6"

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	arg := os.Args[1]

	switch arg {
	case "init":
		commands.InitBoilerplate()

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
	fmt.Println("  init          Generate standard Gohan Framework")
	fmt.Println("\nFlags:")
	fmt.Println("  -v, --version Show the CLI version")
	fmt.Println("  -h, --help    Display the CLI usage instructions")
}