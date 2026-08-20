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

type TemplateData struct {
    ModuleName        string
    UseAuth           bool
    AuthType          string
    UseForgotPassword bool
}

func GetBoilerplateTemplates(moduleName string, useAuth bool, authType string, useForgotPassword bool) (map[string]string, error) {
    result := make(map[string]string)
    data := TemplateData{
        ModuleName:        moduleName,
        UseAuth:           useAuth,
        AuthType:          authType,
        UseForgotPassword: useForgotPassword,
    }

    err := fs.WalkDir(templateFS, "files", func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }

        if d.IsDir() {
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

        relPath, err := filepath.Rel("files", path)
        if err != nil {
            return err
        }

        slashPath := filepath.ToSlash(relPath)
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

//go:embed makes
var MakeFS embed.FS

type MakeData struct {
    Prefix     string
    ModuleName string
}

func RenderMakeTemplate(filename string, data MakeData) (string, error) {
    path := "makes/" + filename
    content, err := MakeFS.ReadFile(path)
    if err != nil {
        return "", fmt.Errorf("failed to read make template %s: %w", filename, err)
    }

    tmpl, err := template.New(filename).Parse(string(content))
    if err != nil {
        return "", fmt.Errorf("failed to parse make template %s: %w", filename, err)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to execute make template %s: %w", filename, err)
    }

    return buf.String(), nil
}