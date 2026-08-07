package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

var templateFS embed.FS

type TemplateData struct {
	ModuleName string
}

func GetBoilerplateTemplates(moduleName string) (map[string]string, error) {
	result := make(map[string]string)
	data := TemplateData{ModuleName: moduleName}

	filesFS, err := fs.Sub(templateFS, "files")
	if err != nil {
		return nil, fmt.Errorf("failed to create sub filesystem: %w", err)
	}

	err = fs.WalkDir(filesFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		content, err := fs.ReadFile(filesFS, path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		tmpl, err := template.New(d.Name()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		slashPath := filepath.ToSlash(path)
		targetPath := strings.TrimSuffix(slashPath, ".tmpl")

		if targetPath == "env" {
			targetPath = ".env"
		}

		result[targetPath] = buf.String()
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to process boilerplate templates: %w", err)
	}

	return result, nil
}