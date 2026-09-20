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

//go:embed files
var templateFS embed.FS

//go:embed makes
var MakeFS embed.FS

type TemplateData struct {
	ModuleName        string
	UseRedis		  bool
	UseAuth           bool
	AuthType          string
	UseRegister       bool
	UseForgotPassword bool
	UseRole           bool
	AppType           string
	FrontendType      string
}

type MakeData struct {
	Prefix     string
	ModuleName string
	UseRedis   bool
}

func GetBoilerplateTemplates(
	data TemplateData,
	shouldSkipFunc func(slashPath string) bool,
) (map[string]string, error) {
	result := make(map[string]string)

	err := fs.WalkDir(templateFS, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel("files", path)
		if err != nil {
			return err
		}

		slashPath := filepath.ToSlash(relPath)

		if strings.HasPrefix(slashPath, "frontend/") {
			if data.AppType != "fullstack" {
				return nil
			}

			targetFrontendPrefix := "frontend/" + data.FrontendType + "/"
			if !strings.HasPrefix(slashPath, targetFrontendPrefix) {
				return nil
			}

			relFrontendPath := strings.TrimPrefix(slashPath, targetFrontendPrefix)
			slashPath = "frontend/" + relFrontendPath
		}

		if shouldSkipFunc != nil && shouldSkipFunc(slashPath) {
			return nil
		}

		content, err := templateFS.ReadFile(path)
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

func RenderMakeTemplate(filename string, data MakeData) (string, error) {
	path := "makes/" + filename
	content, err := MakeFS.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read make template %s: %w", filename, err)
	}

	funcMap := template.FuncMap{
		"toLower": strings.ToLower,
	}

	tmpl, err := template.New(filename).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return "", fmt.Errorf("failed to parse make template %s: %w", filename, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute make template %s: %w", filename, err)
	}

	return buf.String(), nil
}