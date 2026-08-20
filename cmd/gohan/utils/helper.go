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
        if moduleName, found := strings.CutPrefix(line, "module "); found {
            moduleName = strings.TrimSpace(moduleName)
            moduleName = strings.Trim(moduleName, `"'`)
            if moduleName != "" {
                return moduleName
            }
        }
    }
    return "myproject"
}