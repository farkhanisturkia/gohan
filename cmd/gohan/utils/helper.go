package utils

import (
    "bufio"
    "os"
    "path/filepath"
    "strings"
)

func GetModuleName() string {
	if file, err := os.Open("go.mod"); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if moduleName, found := strings.CutPrefix(line, "module "); found {
				moduleName = strings.TrimSpace(moduleName)
				moduleName = strings.Trim(moduleName, `"'`)
				if moduleName != "" {
					return moduleName
				}
			}
		}
	}

	if dir, err := os.Getwd(); err == nil {
		folderName := filepath.Base(dir)
		if folderName != "" && folderName != "." && folderName != "/" {
			return folderName
		}
	}

	return "myproject"
}