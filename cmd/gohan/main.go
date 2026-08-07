package main

import (
	"fmt"
	"os"

	"github.com/farkhanisturkia/gohan/cmd/gohan/commands"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	command := os.Args[1]
	switch command {
	case "init":
		commands.InitBoilerplate()
	default:
		fmt.Printf("The command '%s' is not recognized.\n", command)
		printHelp()
	}
}

func printHelp() {
	fmt.Println("Gohan Framework CLI")
	fmt.Println("\nUsage:")
	fmt.Println("  gohan init   : Generate standard Gohan Framework")
}