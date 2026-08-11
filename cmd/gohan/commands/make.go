package commands

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/farkhanisturkia/gohan/cmd/gohan/templates"
	"github.com/farkhanisturkia/gohan/cmd/gohan/utils"
)

type TemplateData struct {
	ModuleName string
	Prefix string
	Name   string
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

func generateFromTemplate(tmplFileName, targetPath string, data TemplateData) {
	tmplContentStr, err := templates.GetMakeTemplate(tmplFileName)
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	tmpl, err := template.New("make").Parse(tmplContentStr)
	if err != nil {
		fmt.Printf("[error] Failed to parse template: %v\n", err)
		return
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Printf("[error] Failed to render template: %v\n", err)
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

	if err := os.WriteFile(targetPath, buf.Bytes(), 0644); err != nil {
		fmt.Printf("[error] Failed to create file %s: %v\n", targetPath, err)
		return
	}

	fmt.Printf("[info] Created: %s\n", targetPath)
}

func MakeController(name string) {
	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	targetPath := filepath.Join("controllers", cleanName+".go")
	moduleName := utils.GetModuleName()

	data := TemplateData{
		ModuleName: moduleName,
		Prefix:     prefix,
		Name:       cleanName,
	}

	generateFromTemplate("controller.go.tmpl", targetPath, data)
}

func MakeMigration(name string) {
	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	timestamp := time.Now().Format("20060102150405")
	targetPath := filepath.Join("database", "migrations", fmt.Sprintf("%s_%s.go", timestamp, cleanName))
	moduleName := utils.GetModuleName()

	data := TemplateData{
		ModuleName: moduleName,
		Prefix:     prefix,
		Name:       cleanName,
	}

	generateFromTemplate("migration.go.tmpl", targetPath, data)
}

func MakeSeeder(name string) {
	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	targetPath := filepath.Join("database", "seeders", cleanName+".go")
	moduleName := utils.GetModuleName()

	data := TemplateData{
		ModuleName: moduleName,
		Prefix:     prefix,
		Name:       cleanName,
	}

	generateFromTemplate("seeder.go.tmpl", targetPath, data)
}