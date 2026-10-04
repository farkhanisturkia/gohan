package commands

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/farkhanisturkia/gohan/utils"
)

var protocTools = []struct {
	bin  string
	hint string
}{
	{"protoc", "https://grpc.io/docs/protoc-installation/"},
	{"protoc-gen-go", "go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12"},
	{"protoc-gen-connect-go", "go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.21.0"},
}

func protocPathEnv() string {
	pathEnv := os.Getenv("PATH")

	binDir := ""
	if out, err := exec.Command("go", "env", "GOBIN").Output(); err == nil {
		binDir = strings.TrimSpace(string(out))
	}
	if binDir == "" {
		if out, err := exec.Command("go", "env", "GOPATH").Output(); err == nil {
			if goPath := strings.TrimSpace(string(out)); goPath != "" {
				binDir = filepath.Join(goPath, "bin")
			}
		}
	}

	if binDir != "" {
		pathEnv += string(os.PathListSeparator) + binDir
	}

	return pathEnv
}

func findOnPath(name, pathEnv string) bool {
	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.IsDir() {
			continue
		}
		if info.Mode()&0111 != 0 {
			return true
		}
	}
	return false
}

func missingProtocTools() []string {
	pathEnv := protocPathEnv()

	var missing []string
	for _, tool := range protocTools {
		if !findOnPath(tool.bin, pathEnv) {
			missing = append(missing, tool.bin)
		}
	}
	return missing
}

func protocInstallHelp(missing []string) string {
	var sb strings.Builder
	sb.WriteString("the protobuf toolchain is incomplete (missing: " + strings.Join(missing, ", ") + ").\n")
	sb.WriteString("  install protoc                : https://grpc.io/docs/protoc-installation/\n")
	sb.WriteString("  install protoc-gen-go         : go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.12\n")
	sb.WriteString("  install protoc-gen-connect-go : go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.21.0\n")
	sb.WriteString("then run 'make proto' to generate gen/ from proto/.")
	return sb.String()
}

func runProtoc(backendDir, moduleName string) error {
	if missing := missingProtocTools(); len(missing) > 0 {
		return errors.New(protocInstallHelp(missing))
	}

	protoRoot := filepath.Join(backendDir, "proto")

	var files []string
	err := filepath.WalkDir(protoRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}
		rel, err := filepath.Rel(protoRoot, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fmt.Errorf("no proto/ directory found in %s: run 'gohan init' or create proto/*.proto first", backendDir)
	}
	if len(files) == 0 {
		return fmt.Errorf("no .proto files found in %s", protoRoot)
	}
	sort.Strings(files)

	args := []string{
		"--proto_path=proto",
		"--go_out=.",
		"--go_opt=module=" + moduleName,
		"--connect-go_out=.",
		"--connect-go_opt=module=" + moduleName,
	}
	args = append(args, files...)

	cmd := exec.Command("protoc", args...)
	cmd.Dir = backendDir
	cmd.Env = append(os.Environ(), "PATH="+protocPathEnv())

	out, err := cmd.CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("protoc failed:\n%s", message)
	}

	return rewriteGeneratedImports(backendDir)
}

var generatedImportRewrites = []struct {
	from string
	to   string
}{
	{"google.golang.org/protobuf/reflect/protoreflect", "github.com/farkhanisturkia/gohan/pkg/protobuf/reflect/protoreflect"},
	{"google.golang.org/protobuf/runtime/protoimpl", "github.com/farkhanisturkia/gohan/pkg/protobuf/runtime/protoimpl"},
	{"google.golang.org/protobuf/proto", "github.com/farkhanisturkia/gohan/pkg/protobuf/proto"},
	{"connectrpc.com/connect", "github.com/farkhanisturkia/gohan/pkg/connect"},
}

func rewriteGeneratedImports(backendDir string) error {
	genDir := filepath.Join(backendDir, "gen")

	var files []string
	err := filepath.WalkDir(genDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("cannot rewrite generated imports: %w", err)
	}

	for _, path := range files {
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		changed := false
		for i, line := range lines {
			if strings.HasPrefix(strings.TrimSpace(line), "//") {
				continue
			}
			for _, rewrite := range generatedImportRewrites {
				if strings.Contains(line, rewrite.from) {
					lines[i] = strings.ReplaceAll(line, rewrite.from, rewrite.to)
					line = lines[i]
					changed = true
				}
			}
		}

		if changed {
			if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func generateProtoCode(projectRoot, appType, moduleName string) error {
	return runProtoc(resolveBasePath(projectRoot, appType, ""), moduleName)
}

func generateProtoOnInit(backendType, appType, moduleName string) {
	if backendType != "grpc" {
		return
	}

	fmt.Println("[info] Generating protobuf code (gen/ from proto/)...")
	if err := generateProtoCode(".", appType, moduleName); err != nil {
		fmt.Printf("[warning] %v\n", err)
		fmt.Println("[warning] The project will not compile until gen/ exists: install the toolchain above and run 'make proto'.")
		return
	}
	fmt.Println("[info] Generated gen/ from proto/ with protoc")
}

func moduleNameAt(dir string) string {
	if content, err := os.ReadFile(filepath.Join(dir, "go.mod")); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			if name, found := strings.CutPrefix(strings.TrimSpace(line), "module "); found {
				name = strings.Trim(strings.TrimSpace(name), `"'`)
				if name != "" {
					return name
				}
			}
		}
	}
	return utils.GetModuleName()
}

func MakeProto() {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendDir := resolveBasePath(projectRoot, cfg.AppType, "")
	if err := runProtoc(backendDir, moduleNameAt(backendDir)); err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}
	fmt.Println("✅ gen/ regenerated from proto/")
}
