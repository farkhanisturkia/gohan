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
	AppType           string
	BackendType       string
	FrontendType      string
	UseAuth           bool
	AuthType          string
	UseForgotPassword bool
	UseRole           bool
}

func InitBoilerplate() {
	reader := bufio.NewReader(os.Stdin)
	config := InitConfig{}

	fmt.Println("🚀 Initializing Gohan Framework Project...")
	fmt.Println("--------------------------------------------------")

	// 1. Select App Type (API Only vs Fullstack)
	fmt.Println("\n[1] Select Application Type:")
	fmt.Println("  1) API Only (Default)")
	fmt.Println("  2) Fullstack")
	fmt.Print("Choose option [1-2]: ")
	appChoice := readInput(reader)

	if appChoice == "2" {
		config.AppType = "fullstack"
		fmt.Println("   ↳ Selected: Fullstack")
	} else {
		config.AppType = "api"
		fmt.Println("   ↳ Selected: API Only")
	}

	// 2. Select Backend Type (REST vs gRPC)
	fmt.Println("\n[2] Select Backend Architecture:")
	fmt.Println("  1) REST API (Default)")
	fmt.Println("  2) gRPC (*Coming soon)")
	fmt.Print("Choose option [1-2]: ")
	backendChoice := readInput(reader)

	if backendChoice == "2" {
		fmt.Println("\n[info] gRPC backend architecture is currently ON GOING / COMING SOON!")
		fmt.Println("[info] Initialization aborted.")
		return
	}
	config.BackendType = "rest"
	fmt.Println("   ↳ Selected: REST API")

	// 3. Select Frontend Type
	if config.AppType == "fullstack" {
		fmt.Println("\n[3] Select Frontend Framework:")
		fmt.Println("  1) Vue 3 + Vite (Default)")
		fmt.Println("  2) React + Vite (*Coming soon)")
		fmt.Print("Choose option [1-2]: ")
		feChoice := readInput(reader)

		if feChoice == "2" {
			fmt.Println("\n[info] React + Vite frontend Framework is currently ON GOING / COMING SOON!")
			fmt.Println("[info] Initialization aborted.")
			return
		}
		config.FrontendType = "vue"
		fmt.Println("   ↳ Selected: Vue 3 + Vite")

	} else {
		config.FrontendType = ""
	}

	// 4. Auth & Middleware Options
	stepNum := 3
	if config.AppType == "fullstack" {
		stepNum = 4
	}

	fmt.Printf("\n[%d] Do you want to include Authentication? (y/N): ", stepNum)
	authChoice := readInput(reader)

	if strings.ToLower(authChoice) == "y" || strings.ToLower(authChoice) == "yes" {
		config.UseAuth = true

		fmt.Println("\n    Select Authentication Type:")
		fmt.Println("      1) PAT (Default)")
		fmt.Println("      2) JWT")
		fmt.Print("    Choose option [1-2]: ")
		authTypeChoice := readInput(reader)

		if authTypeChoice == "2" {
			config.AuthType = "jwt"
			fmt.Println("       ↳ Selected: JSON Web Token (JWT)")
		} else {
			config.AuthType = "pat"
			fmt.Println("       ↳ Selected: Personal Access Token (PAT)")
		}

		fmt.Print("\n    Do you want to include Forgot Password features? (y/N): ")
		forgotChoice := readInput(reader)
		config.UseForgotPassword = strings.ToLower(forgotChoice) == "y" || strings.ToLower(forgotChoice) == "yes"

		fmt.Print("\n    Do you want to include Role-Based Middleware (RBAC)? (y/N): ")
		roleChoice := readInput(reader)
		config.UseRole = strings.ToLower(roleChoice) == "y" || strings.ToLower(roleChoice) == "yes"
	} else {
		config.UseAuth = false
		config.AuthType = ""
		config.UseForgotPassword = false
		config.UseRole = false
	}

	// Output Summary
	fmt.Println("\n--------------------------------------------------")
	fmt.Println("[info] Generating project boilerplate with:")
	fmt.Printf("       - App Type       : %s\n", strings.ToUpper(config.AppType))
	fmt.Printf("       - Backend Type   : %s\n", strings.ToUpper(config.BackendType))
	if config.AppType == "fullstack" {
		fmt.Printf("       - Frontend       : %s\n", strings.ToUpper(config.FrontendType))
	}

	if config.UseAuth {
		forgotPassStr := "NO"
		if config.UseForgotPassword {
			forgotPassStr = "YES"
		}
		roleStr := "NO"
		if config.UseRole {
			roleStr = "YES"
		}
		fmt.Printf("       - Auth           : YES (%s)\n", strings.ToUpper(config.AuthType))
		fmt.Printf("       - Forgot Pass    : %s\n", forgotPassStr)
		fmt.Printf("       - Role Middleware: %s\n", roleStr)
	} else {
		fmt.Println("       - Auth           : NO")
		fmt.Println("       - Forgot Pass    : NO")
		fmt.Println("       - Role Middleware: NO")
	}
	fmt.Println("--------------------------------------------------\n")

	moduleName := utils.GetModuleName()

	templateData := templates.TemplateData{
		ModuleName:        moduleName,
		UseAuth:           config.UseAuth,
		AuthType:          config.AuthType,
		UseForgotPassword: config.UseForgotPassword,
		UseRole:           config.UseRole,
		AppType:           config.AppType,
		FrontendType:      config.FrontendType,
	}

	templateMap, err := templates.GetBoilerplateTemplates(templateData, func(path string) bool {
		return shouldSkipFile(path, config)
	})
	if err != nil {
		fmt.Printf("[error] Failed to load boilerplate templates: %v\n", err)
		return
	}

	for path, content := range templateMap {
		targetPath := path

		if config.AppType == "fullstack" && !strings.HasPrefix(path, "frontend/") {
			if path != "Makefile" {
				targetPath = filepath.Join("backend", path)
			}
		}

		dir := filepath.Dir(targetPath)
		if dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}

		fileBytes := []byte(content)

		if strings.HasSuffix(targetPath, ".go") {
			formatted, err := format.Source(fileBytes)
			if err == nil {
				fileBytes = formatted
			}
		}

		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			if err := os.WriteFile(targetPath, fileBytes, 0644); err != nil {
				fmt.Printf("[error] Failed to create the %s file: %v\n", targetPath, err)
			} else {
				fmt.Printf("[info] %s file created\n", targetPath)
			}
		} else {
			fmt.Printf("[error] The %s file already exists\n", targetPath)
		}
	}

	fmt.Println("\n✅ Gohan project initialized successfully!")
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Run 'make setup' to install all dependencies & generate key")
	fmt.Println("  2. Run 'make dev' to start the application")
}

func readInput(reader *bufio.Reader) string {
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func shouldSkipFile(path string, cfg InitConfig) bool {
    filename := filepath.Base(path)

    match := func(target string) bool {
        return filename == target || filename == target+".tmpl"
    }

    if !cfg.UseAuth {
        if match("auth_controller.go") ||
            match("password_reset_controller.go") ||
            match("role_controller.go") ||
            match("00000000000002_create_personal_access_token_table.go") ||
            match("00000000000003_create_password_reset_table.go") ||
            match("00000000000004_create_role_table.go") ||
            match("role.go") ||
            match("auth_middleware.go") ||
            match("role_middleware.go") {
            return true
        }

        if match("useAuth.ts") ||
			match("DashboardView.vue") ||
			match("LoginView.vue") ||
            match("UsersView.vue") {
            return true
        }
    }

    if cfg.UseAuth && cfg.AuthType == "jwt" {
        if match("00000000000002_create_personal_access_token_table.go") {
            return true
        }
    }

    if cfg.UseAuth && !cfg.UseForgotPassword {
        if match("password_reset_controller.go") ||
            match("00000000000003_create_password_reset_table.go") ||
            match("ForgotPasswordView.vue") ||
            match("ResetPasswordView.vue") {
            return true
        }
    }

    if cfg.UseAuth && !cfg.UseRole {
        if match("00000000000004_create_role_table.go") ||
            match("role_controller.go") ||
            match("role.go") ||
            match("role_middleware.go") {
            return true
        }
    }

    return false
}