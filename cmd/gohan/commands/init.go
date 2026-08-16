package commands

import (
	"bufio"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"

	"github.com/farkhanisturkia/gohan/cmd/gohan/templates"
	"github.com/farkhanisturkia/gohan/cmd/gohan/utils"
)

type InitConfig struct {
	Architecture      string
	UseAuth           bool
	AuthType          string
	UseForgotPassword bool
}

func InitBoilerplate() {
	reader := bufio.NewReader(os.Stdin)
	config := InitConfig{}

	fmt.Println("🚀 Initializing Gohan Framework Project...")
	fmt.Println("--------------------------------------------------")

	fmt.Println("\n[1] Select Architecture API:")
	fmt.Println("  1) REST API (Default)")
	fmt.Println("  2) gRPC")
	fmt.Print("Choose option [1-2]: ")
	archChoice := readInput(reader)

	if archChoice == "2" {
		fmt.Println("\n[info] gRPC architecture is currently ON GOING / COMING SOON!")
		fmt.Println("[info] Initialization aborted.")
		return
	}
	config.Architecture = "rest"
	fmt.Println("   ↳ Selected: REST API")

	fmt.Print("\n[2] Do you want to include Authentication? (y/N): ")
	authChoice := readInput(reader)

	if strings.ToLower(authChoice) == "y" || strings.ToLower(authChoice) == "yes" {
		config.UseAuth = true

		fmt.Println("\n    Select Authentication Type:")
		fmt.Println("      1) PAT (Default)")
		fmt.Println("      2) JWT")
		fmt.Print("    Choose option [1-2]: ")
		authTypeChoice := readInput(reader)

		if authTypeChoice == "2" {
			fmt.Println("\n[info] JWT Authentication is currently ON GOING / COMING SOON!")
			fmt.Println("[info] Initialization aborted.")
			return
		}
		config.AuthType = "pat"
		fmt.Println("       ↳ Selected: Personal Access Token (PAT)")

		fmt.Print("\n[3] Do you want to include Forgot Password features? (y/N): ")
		forgotChoice := readInput(reader)
		if strings.ToLower(forgotChoice) == "y" || strings.ToLower(forgotChoice) == "yes" {
			config.UseForgotPassword = true
		} else {
			config.UseForgotPassword = false
		}
	} else {
		config.UseAuth = false
		config.UseForgotPassword = false
	}

	fmt.Println("\n--------------------------------------------------")
	fmt.Println("[info] Generating project boilerplate with:")
	fmt.Printf("       - Architecture : %s\n", strings.ToUpper(config.Architecture))
	if config.UseAuth {
		fmt.Printf("       - Auth         : YES (%s)\n", strings.ToUpper(config.AuthType))
		fmt.Printf("       - Forgot Pass  : %v\n", config.UseForgotPassword)
	} else {
		fmt.Println("       - Auth         : NO")
		fmt.Println("       - Forgot Pass  : NO")
	}
	fmt.Println("--------------------------------------------------\n")

	moduleName := utils.GetModuleName()
	templateMap, err := templates.GetBoilerplateTemplates(moduleName, config.UseAuth, config.UseForgotPassword)
	if err != nil {
		fmt.Printf("[error] Failed to load boilerplate templates: %v\n", err)
		return
	}

	for path, content := range templateMap {
		if shouldSkipFile(path, config) {
			continue
		}

		dir := filepath.Dir(path)
		if dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}

		fileBytes := []byte(content)

		if strings.HasSuffix(path, ".go") {
			formatted, err := format.Source(fileBytes)
			if err == nil {
				fileBytes = formatted
			}
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, fileBytes, 0644); err != nil {
				fmt.Printf("[error] Failed to create the %s file: %v\n", path, err)
			} else {
				fmt.Printf("[info] %s file created\n", path)
			}
		} else {
			fmt.Printf("[error] The %s file already exists\n", path)
		}
	}

	fmt.Println("\n✅ Gohan project initialized successfully!")
}

func readInput(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func shouldSkipFile(path string, cfg InitConfig) bool {
	filename := filepath.Base(path)

	if !cfg.UseAuth {
		if filename == "auth_controller.go" ||
			filename == "password_reset_controller.go" ||
			filename == "00000000000002_create_personal_access_token_table.go" ||
			filename == "00000000000003_create_password_reset_table.go" ||
			filename == "auth.go" {
			return true
		}
	}

	if cfg.UseAuth && !cfg.UseForgotPassword {
		if filename == "password_reset_controller.go" ||
			filename == "00000000000003_create_password_reset_table.go" {
			return true
		}
	}

	return false
}