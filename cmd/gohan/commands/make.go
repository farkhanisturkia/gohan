package commands

import (
    "bufio"
    "fmt"
    "go/format"
    "os"
    "path/filepath"
    "strings"
    "time"

    "github.com/farkhanisturkia/gohan/cmd/gohan/templates"
    "github.com/farkhanisturkia/gohan/cmd/gohan/utils"
)

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
    cleanName := strings.TrimSuffix(name, ".go")
    prefix := toPascalCase(cleanName)
    targetPath := filepath.Join("controllers", cleanName+".go")
    moduleName := utils.GetModuleName()

    reader := bufio.NewReader(os.Stdin)
    fmt.Print("Do you want to include Redis Caching in this controller? [y/N]: ")
    input, _ := reader.ReadString('\n')
    input = strings.TrimSpace(strings.ToLower(input))

    useRedis := input == "y" || input == "yes"

    data := templates.MakeData{
        ModuleName: moduleName,
        Prefix:     prefix,
        UseRedis:   useRedis,
    }

    generateFromTemplate("controller.go.tmpl", targetPath, data)
}

func MakeMigration(name string) {
    cleanName := strings.TrimSuffix(name, ".go")
    prefix := toPascalCase(cleanName)
    timestamp := time.Now().Format("20060102150405")

    fileName := fmt.Sprintf("%s_%s.go", timestamp, cleanName)
    targetPath := filepath.Join("database", "migrations", fileName)
    moduleName := utils.GetModuleName()

    data := templates.MakeData{
        ModuleName: moduleName,
        Prefix:     prefix,
    }

    generateFromTemplate("migration.go.tmpl", targetPath, data)

    appendMigrationToDefault(prefix)
}

func MakeSeeder(name string) {
    cleanName := strings.TrimSuffix(name, ".go")
    prefix := toPascalCase(cleanName)
    targetPath := filepath.Join("database", "seeders", strings.ToLower(prefix)+".go")
    moduleName := utils.GetModuleName()

    data := templates.MakeData{
        ModuleName: moduleName,
        Prefix:     prefix,
    }

    generateFromTemplate("seeder.go.tmpl", targetPath, data)

    appendSeederToDefault(prefix)
}

func appendMigrationToDefault(prefix string) {
    defaultPath := filepath.Join("database", "migrations", "default.go")
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

func appendSeederToDefault(prefix string) {
    defaultPath := filepath.Join("database", "seeders", "default.go")
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