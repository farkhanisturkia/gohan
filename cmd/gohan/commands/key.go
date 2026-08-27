package commands

import (
    "fmt"
    "os"
    "strings"

    "github.com/farkhanisturkia/gohan"
)

func GenerateAppKey() {
    envPath := ".env"

    if _, err := os.Stat(envPath); os.IsNotExist(err) {
        fmt.Println("[error] File .env not found. Please run 'gohan init' or create .env first.")
        return
    }

    content, err := os.ReadFile(envPath)
    if err != nil {
        fmt.Printf("[error] Failed to read .env file: %v\n", err)
        return
    }

    lines := strings.Split(string(content), "\n")
    hasJwtInEnv := false
    for _, line := range lines {
        if strings.HasPrefix(strings.TrimSpace(line), "JWT_SECRET=") {
            hasJwtInEnv = true
            break
        }
    }

    appToken, err := gohan.GenerateRandomToken(32)
    if err != nil {
        fmt.Printf("[error] Failed to generate random app key: %v\n", err)
        return
    }
    newAppKey := "base64:" + appToken

    var jwtToken string
    if hasJwtInEnv {
        jwtToken, err = gohan.GenerateRandomToken(32)
        if err != nil {
            fmt.Printf("[error] Failed to generate random JWT secret: %v\n", err)
            return
        }
    }

    appKeyFound := false

    for i, line := range lines {
        trimmed := strings.TrimSpace(line)
        if strings.HasPrefix(trimmed, "APP_KEY=") {
            lines[i] = fmt.Sprintf("APP_KEY=%s", newAppKey)
            appKeyFound = true
        }
        if hasJwtInEnv && strings.HasPrefix(trimmed, "JWT_SECRET=") {
            lines[i] = fmt.Sprintf("JWT_SECRET=%s", jwtToken)
        }
    }

    if !appKeyFound {
        lines = append([]string{fmt.Sprintf("APP_KEY=%s", newAppKey)}, lines...)
    }

    output := strings.Join(lines, "\n")
    if err := os.WriteFile(envPath, []byte(output), 0644); err != nil {
        fmt.Printf("[error] Failed to update .env file: %v\n", err)
        return
    }

    fmt.Printf("[info] Application key [%s] set successfully.\n", newAppKey)
    if hasJwtInEnv {
        fmt.Printf("[info] JWT secret [%s] set successfully.\n", jwtToken)
    }
}