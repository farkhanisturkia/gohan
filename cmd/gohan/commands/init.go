package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/farkhanisturkia/gohan/cmd/gohan/templates"
	"github.com/farkhanisturkia/gohan/cmd/gohan/utils"
)

func InitBoilerplate() {
	moduleName := utils.GetModuleName()
	templateMap, err := templates.GetBoilerplateTemplates(moduleName)
	if err != nil {
		fmt.Printf("[error] Failed to load boilerplate templates: %v\n", err)
		return
	}

	for path, content := range templateMap {
		dir := filepath.Dir(path)
		if dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				fmt.Printf("[error] Failed to create the %s file: %v\n", path, err)
			} else {
				fmt.Printf("[info] %s file created\n", path)
			}
		} else {
			fmt.Printf("[error] The %s file already exists\n", path)
		}
	}
}