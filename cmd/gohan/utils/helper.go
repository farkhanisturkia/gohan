package utils

import (
	"bufio"
	"os"
	"strings"
)

func GetModuleName() string {
	file, err := os.Open("go.mod")
	if err != nil {
		return "myproject"
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimPrefix(line, "module ")
		}
	}
	return "myproject"
}