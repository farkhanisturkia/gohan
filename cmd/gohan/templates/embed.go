package templates

import (
	"bytes"
	"embed"
	"fmt"
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
	baseDir := "files"

	err := walkEmbedDir(baseDir, baseDir, data, result)
	if err != nil {
		return nil, fmt.Errorf("failed to process boilerplate templates: %w", err)
	}

	return result, nil
}

func walkEmbedDir(currentDir, baseDir string, data TemplateData, result map[string]string) error {
	entries, err := templateFS.ReadDir(currentDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(currentDir, entry.Name())

		if entry.IsDir() {
			if err := walkEmbedDir(path, baseDir, data, result); err != nil {
				return err
			}
			continue
		}

		content, err := templateFS.ReadFile(path)
		if err != nil {
			return err
		}

		tmpl, err := template.New(entry.Name()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", path, err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template %s: %w", path, err)
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		targetPath := strings.TrimSuffix(relPath, ".tmpl")
		result[targetPath] = buf.String()
	}

	return nil
}