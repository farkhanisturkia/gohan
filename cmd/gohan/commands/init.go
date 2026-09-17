package commands

import (
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
    UseRedis          bool
    UseAuth           bool
    AuthType          string
    UseForgotPassword bool
    UseRole           bool
}

func InitBoilerplate() {
    config := InitConfig{}

    fmt.Println("🚀 Initializing Gohan Framework Project...")
    fmt.Println("--------------------------------------------------")

    // 1. Select App Type (API Only vs Fullstack)
    appChoice, err := utils.RunSelect(
        "Select Application Type", 
        []string{
            "API Only (Default)", 
            "Fullstack",
        },
    )
    if err != nil {
        fmt.Println("[info] Initialization cancelled.")
        return
    }

    if strings.Contains(appChoice, "Fullstack") {
        config.AppType = "fullstack"
    } else {
        config.AppType = "api"
    }

    // 2. Select Backend Type (REST, gRPC, GraphQL)
    backendChoice, err := utils.RunSelect(
        "Select Backend Architecture", 
        []string{
            "REST API (Default)", 
            "gRPC (*Coming soon)", 
            "GraphQL (*Coming soon)",
        },
    )
    if err != nil {
        fmt.Println("[info] Initialization cancelled.")
        return
    }

    if strings.Contains(backendChoice, "gRPC") {
        fmt.Println("\n[info] gRPC backend architecture is currently ON GOING / COMING SOON!")
        fmt.Println("[info] Initialization aborted.")
        return
    }
    if strings.Contains(backendChoice, "GraphQL") {
        fmt.Println("\n[info] GraphQL backend architecture is currently ON GOING / COMING SOON!")
        fmt.Println("[info] Initialization aborted.")
        return
    }
    config.BackendType = "rest"

    // 3. Select Frontend Type
    if config.AppType == "fullstack" {
        feChoice, err := utils.RunSelect(
            "Select Frontend Framework", 
            []string{
                "Vue 3 (Default)", 
                "React (*Coming soon)",
            },
        )
        if err != nil {
            fmt.Println("[info] Initialization cancelled.")
            return
        }

        if strings.Contains(feChoice, "React") {
            fmt.Println("\n[info] React frontend Framework is currently ON GOING / COMING SOON!")
            fmt.Println("[info] Initialization aborted.")
            return
        }
        config.FrontendType = "vue"
    }

    // 4. Redis Integration Option
    redisChoice, err := utils.RunSelect(
        "Include Redis Support (Caching / Session)?", 
        []string{
            "Yes", 
            "No",
        },
    )
    if err != nil {
        fmt.Println("[info] Initialization cancelled.")
        return
    }
    config.UseRedis = (redisChoice == "Yes")

    // 5. Auth & Middleware Options
    authChoice, err := utils.RunSelect(
        "Include Authentication?", 
        []string{
            "Yes", 
            "No",
        },
    )
    if err != nil {
        fmt.Println("[info] Initialization cancelled.")
        return
    }

    if authChoice == "Yes" {
        config.UseAuth = true

        // Select Auth Type
        authTypeChoice, err := utils.RunSelect(
            "  ↳ Select Authentication Type", 
            []string{
                "PAT - Personal Access Token (Default)", 
                "JWT - JSON Web Token",
            },
        )
        if err != nil {
            fmt.Println("[info] Initialization cancelled.")
            return
        }

        if strings.Contains(authTypeChoice, "JWT") {
            config.AuthType = "jwt"
        } else {
            config.AuthType = "pat"
        }

        // Forgot Password Prompt
        forgotChoice, err := utils.RunSelect(
            "  ↳ Include Forgot Password features?", 
            []string{
                "Yes", 
                "No",
            },
        )
        if err != nil {
            fmt.Println("[info] Initialization cancelled.")
            return
        }
        config.UseForgotPassword = (forgotChoice == "Yes")

        // Role-Based Middleware Prompt
        roleChoice, err := utils.RunSelect(
            "  ↳ Include Role-Based Middleware (RBAC)?", 
            []string{
                "Yes", 
                "No",
            },
        )
        if err != nil {
            fmt.Println("[info] Initialization cancelled.")
            return
        }
        config.UseRole = (roleChoice == "Yes")
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
    fmt.Printf("       - Backend        : %s\n", strings.ToUpper(config.BackendType))
    if config.AppType == "fullstack" {
        fmt.Printf("       - Frontend       : %s\n", strings.ToUpper(config.FrontendType))
    }

    redisStr := "NO"
    if config.UseRedis {
        redisStr = "YES"
    }
    fmt.Printf("       - Redis Support  : %s\n", redisStr)

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
        UseRedis:          config.UseRedis,
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
				// fmt.Printf("[info] %s file created\n", targetPath)
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
