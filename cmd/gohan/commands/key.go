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

	randomToken, err := gohan.GenerateRandomToken(32)
	if err != nil {
		fmt.Printf("[error] Failed to generate random key: %v\n", err)
		return
	}

	newKey := "base64:" + randomToken

	lines := strings.Split(string(content), "\n")
	keyFound := false

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "APP_KEY=") {
			lines[i] = fmt.Sprintf("APP_KEY=%s", newKey)
			keyFound = true
			break
		}
	}

	if !keyFound {
		lines = append([]string{fmt.Sprintf("APP_KEY=%s", newKey)}, lines...)
	}

	output := strings.Join(lines, "\n")
	if err := os.WriteFile(envPath, []byte(output), 0644); err != nil {
		fmt.Printf("[error] Failed to update .env file: %v\n", err)
		return
	}

	fmt.Printf("[info] Application key [%s] set successfully.\n", newKey)
}