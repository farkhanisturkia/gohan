package commands

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/farkhanisturkia/gohan/cmd/gohan/templates"
	"github.com/farkhanisturkia/gohan/cmd/gohan/utils"
)

type GohanConfig struct {
	AppName  string     `json:"app_name"`
	AppType  string     `json:"app_type"`
	AppSpecs GohanSpecs `json:"app_specs"`
}

type GohanSpecs struct {
	BackendType  string `json:"backend_type"`
	FrontendType string `json:"frontend_type"`
	Auth         bool   `json:"auth"`
	AuthType     string `json:"auth_type"`
	RBAC         bool   `json:"rbac"`
	Redis        bool   `json:"redis"`
}

func getGohanConfig() (*GohanConfig, string, error) {
	currDir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	for {
		configPath := filepath.Join(currDir, "gohan.json")
		if _, err := os.Stat(configPath); err == nil {
			content, err := os.ReadFile(configPath)
			if err != nil {
				return nil, "", err
			}
			var cfg GohanConfig
			if err := json.Unmarshal(content, &cfg); err != nil {
				return nil, "", err
			}
			return &cfg, currDir, nil
		}

		parentDir := filepath.Dir(currDir)
		if parentDir == currDir {
			break
		}
		currDir = parentDir
	}

	return nil, "", fmt.Errorf("gohan.json not found. Please run 'gohan init' or execute this command inside a Gohan project")
}

func resolveBasePath(projectRoot string, appType string, subPath string) string {
	if strings.ToLower(appType) == "fullstack" {
		return filepath.Join(projectRoot, "backend", subPath)
	}
	return filepath.Join(projectRoot, subPath)
}

func toPascalCase(s string) string {
	s = strings.TrimSuffix(s, ".go")
	s = strings.TrimPrefix(s, "create_")
	s = strings.TrimSuffix(s, "_controller")
	s = strings.TrimSuffix(s, "_seeder")
	s = strings.TrimSuffix(s, "_migration")
	s = strings.TrimSuffix(s, "_table")

	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})

	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, "")
}

func generateFromTemplate(tmplFileName, targetPath string, data templates.MakeData) {
	renderedContent, err := templates.RenderMakeTemplate(tmplFileName, data)
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	dir := filepath.Dir(targetPath)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Printf("[error] Failed to create directory %s: %v\n", dir, err)
			return
		}
	}

	if _, err := os.Stat(targetPath); !os.IsNotExist(err) {
		fmt.Printf("[error] The file %s already exists\n", targetPath)
		return
	}

	fileBytes := []byte(renderedContent)

	if strings.HasSuffix(targetPath, ".go") {
		formatted, err := format.Source(fileBytes)
		if err == nil {
			fileBytes = formatted
		}
	}

	if err := os.WriteFile(targetPath, fileBytes, 0644); err != nil {
		fmt.Printf("[error] Failed to create file %s: %v\n", targetPath, err)
		return
	}

	fmt.Printf("[info] Created: %s\n", targetPath)
}

func MakeController(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := "rest"
	if cfg.AppSpecs.BackendType != "" {
		backendType = strings.ToLower(cfg.AppSpecs.BackendType)
	}

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)

	baseDir := resolveBasePath(projectRoot, cfg.AppType, "controllers")
	targetPath := filepath.Join(baseDir, cleanName+".go")
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
		UseRedis:   cfg.AppSpecs.Redis,
	}

	tmplPath := fmt.Sprintf("backend/%s/controller.go.tmpl", backendType)
	generateFromTemplate(tmplPath, targetPath, data)
}

func MakeMigration(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := "rest"
	if cfg.AppSpecs.BackendType != "" {
		backendType = strings.ToLower(cfg.AppSpecs.BackendType)
	}

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	timestamp := time.Now().Format("20060102150405")

	fileName := fmt.Sprintf("%s_%s.go", timestamp, cleanName)

	baseDir := resolveBasePath(projectRoot, cfg.AppType, "database/migrations")
	targetPath := filepath.Join(baseDir, fileName)
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
	}

	tmplPath := fmt.Sprintf("backend/%s/migration.go.tmpl", backendType)
	generateFromTemplate(tmplPath, targetPath, data)

	appendMigrationToDefault(baseDir, prefix)
}

func MakeSeeder(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := "rest"
	if cfg.AppSpecs.BackendType != "" {
		backendType = strings.ToLower(cfg.AppSpecs.BackendType)
	}

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)

	baseDir := resolveBasePath(projectRoot, cfg.AppType, "database/seeders")
	targetPath := filepath.Join(baseDir, cleanName+".go")
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
	}

	tmplPath := fmt.Sprintf("backend/%s/seeder.go.tmpl", backendType)
	generateFromTemplate(tmplPath, targetPath, data)

	appendSeederToDefault(baseDir, prefix)
}

func appendMigrationToDefault(baseDir, prefix string) {
	defaultPath := filepath.Join(baseDir, "default.go")
	content, err := os.ReadFile(defaultPath)
	if err != nil {
		return
	}

	targetCall := fmt.Sprintf("%sMigration(db)", prefix)
	if strings.Contains(string(content), targetCall) {
		return
	}

	updated := strings.Replace(
		string(content),
		"}",
		fmt.Sprintf("\t%s\n}", targetCall),
		1,
	)

	formatted, err := format.Source([]byte(updated))
	if err == nil {
		_ = os.WriteFile(defaultPath, formatted, 0644)
	}
}

func appendSeederToDefault(baseDir, prefix string) {
	defaultPath := filepath.Join(baseDir, "default.go")
	content, err := os.ReadFile(defaultPath)
	if err != nil {
		return
	}

	targetCall := fmt.Sprintf("%sSeeder(db)", prefix)
	if strings.Contains(string(content), targetCall) {
		return
	}

	updated := strings.Replace(
		string(content),
		"}",
		fmt.Sprintf("\t%s\n}", targetCall),
		1,
	)

	formatted, err := format.Source([]byte(updated))
	if err == nil {
		_ = os.WriteFile(defaultPath, formatted, 0644)
	}
}