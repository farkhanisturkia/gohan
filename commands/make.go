package commands

import (
	"encoding/json"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/farkhanisturkia/gohan/templates"
	"github.com/farkhanisturkia/gohan/utils"
)

type GohanConfig struct {
	AppName  string     `json:"app_name"`
	AppType  string     `json:"app_type"`
	AppSpecs GohanSpecs `json:"app_specs"`
}

type GohanSpecs struct {
	BackendType  string `json:"backend_type"`
	FrontendType string `json:"frontend_type"`
	Auth         bool   `json:"auth"`
	AuthType     string `json:"auth_type"`
	RBAC         bool   `json:"rbac"`
	Redis        bool   `json:"redis"`
}

func getGohanConfig() (*GohanConfig, string, error) {
	currDir, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	for {
		configPath := filepath.Join(currDir, "gohan.json")
		if _, err := os.Stat(configPath); err == nil {
			content, err := os.ReadFile(configPath)
			if err != nil {
				return nil, "", err
			}
			var cfg GohanConfig
			if err := json.Unmarshal(content, &cfg); err != nil {
				return nil, "", err
			}
			return &cfg, currDir, nil
		}

		parentDir := filepath.Dir(currDir)
		if parentDir == currDir {
			break
		}
		currDir = parentDir
	}

	return nil, "", fmt.Errorf("gohan.json not found. Please run 'gohan init' or execute this command inside a Gohan project")
}

func backendTypeOf(cfg *GohanConfig) string {
	if cfg.AppSpecs.BackendType != "" {
		return strings.ToLower(cfg.AppSpecs.BackendType)
	}
	return "rest"
}

func frontendTypeOf(cfg *GohanConfig) string {
	if cfg.AppSpecs.FrontendType != "" {
		return strings.ToLower(cfg.AppSpecs.FrontendType)
	}
	return "vue"
}

func requireFullstack(cfg *GohanConfig) bool {
	if strings.ToLower(cfg.AppType) != "fullstack" {
		fmt.Println("[error] 'make:resource' is only available for fullstack applications.")
		fmt.Println("[info]  Set \"app_type\": \"fullstack\" in gohan.json and re-run 'gohan init' to add a frontend.")
		return false
	}
	return true
}

func resolveBasePath(projectRoot string, appType string, subPath string) string {
	if strings.ToLower(appType) == "fullstack" {
		return filepath.Join(projectRoot, "backend", subPath)
	}
	return filepath.Join(projectRoot, subPath)
}

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

func readTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func writeGoFile(path, content string) error {
	if formatted, err := format.Source([]byte(content)); err == nil {
		content = string(formatted)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

func appendToLastMarker(content, marker, block string) (string, bool) {
	idx := strings.LastIndex(content, marker)
	if idx < 0 {
		return content, false
	}
	return content[:idx] + block + content[idx:], true
}

func appendBeforeFirstMarker(content, marker, block string) (string, bool) {
	idx := strings.Index(content, marker)
	if idx < 0 {
		return content, false
	}
	return content[:idx] + block + content[idx:], true
}

type registrationStatus int

const (
	regInserted registrationStatus = iota
	regAlreadyExists
	regFailed
)

func generateGrpcProtoFiles(projectRoot string, cfg *GohanConfig, data templates.MakeData, snake string) {
	backendBase := resolveBasePath(projectRoot, cfg.AppType, "")

	generateFromTemplate(
		"backend/grpc/proto.proto.tmpl",
		filepath.Join(backendBase, "proto", snake, snake+".proto"),
		data,
	)
	generateFromTemplate(
		"backend/grpc/service.go.tmpl",
		filepath.Join(backendBase, "services", snake+"_service.go"),
		data,
	)
}

func generateProto(projectRoot string, cfg *GohanConfig, moduleName string) {
	if err := generateProtoCode(projectRoot, cfg.AppType, moduleName); err != nil {
		fmt.Printf("[warning] Could not generate gen/: %v\n", err)
		return
	}
	fmt.Println("[info] Regenerated gen/ from proto/ with protoc")
}

func registerGrpcService(projectRoot string, cfg *GohanConfig, moduleName, prefix, snake string, guard *ResourceGuard) {
	backendBase := resolveBasePath(projectRoot, cfg.AppType, "")
	routesPath := filepath.Join(backendBase, "routes.go")

	content, err := readTextFile(routesPath)
	if err != nil {
		fmt.Printf("[warn] Skipped %s service registration: cannot read %s\n", prefix, routesPath)
		return
	}

	// 1. Import the generated connect package.
	if !strings.Contains(content, fmt.Sprintf("%sv1connect \"", snake)) {
		imp := fmt.Sprintf("\n    %sv1connect \"%s/gen/%s/%sv1connect\"", snake, moduleName, snake, snake)

		updated, ok := appendBeforeFirstMarker(content, ")\n\n// SetupRoutes", imp)
		if !ok {
			updated, ok = appendBeforeFirstMarker(content, ")\n\nfunc SetupRoutes", imp)
		}
		if !ok {
			fmt.Printf("[warn] Could not add the %sv1connect import automatically. Add it manually to routes.go.\n", snake)
		} else {
			content = updated
		}
	}

	// 2. Register the handler before the REST gateway setup.
	if !strings.Contains(content, fmt.Sprintf("New%sServiceHandler(", prefix)) {
		block := grpcServiceRegistration(prefix, snake, guard)

		updated, ok := appendBeforeFirstMarker(content, "gateway.SetupGateway(mux, db)", block)
		if !ok {
			fmt.Printf("[warn] Could not register the %s service automatically. Add it manually to routes.go.\n", prefix)
		} else {
			content = updated
		}
	}

	if err := writeGoFile(routesPath, content); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", routesPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", routesPath)

	// 3. Public resources must bypass the auth interceptor.
	if cfg.AppSpecs.Auth && guard != nil && !guard.Auth {
		procBase := "/" + snake + ".v1." + prefix + "Service/"
		addPublicProcedures(filepath.Join(backendBase, "services", "interceptors", "interceptors.go"), []string{
			procBase + "List" + prefix + "s",
			procBase + "Create" + prefix,
			procBase + "Get" + prefix,
			procBase + "Update" + prefix,
			procBase + "Delete" + prefix,
		})
	}
}

func grpcServiceRegistration(prefix, snake string, guard *ResourceGuard) string {
	if guard != nil && len(guard.Roles) > 0 {
		return fmt.Sprintf(`    // %s Resource (typed Connect service, restricted to the %s role(s))
    %sPath, %sHandler := %sv1connect.New%sServiceHandler(
        services.New%sService(db),
        connectHandler,
        gohan.WithInterceptors(roleGuard.Intercept("%s.v1.%sService", %s)),
    )
    mux.Handle(%sPath, %sHandler)

`,
			prefix, strings.Join(guard.Roles, ", "),
			snake, snake, snake, prefix,
			prefix,
			snake, prefix, formatRolesGo(guard.Roles),
			snake, snake)
	}

	guardNote := ""
	if guard != nil && !guard.Auth {
		guardNote = " (public)"
	}

	return fmt.Sprintf(`    // %s Resource (typed Connect service)%s
    %sPath, %sHandler := %sv1connect.New%sServiceHandler(
        services.New%sService(db),
        connectHandler,
    )
    mux.Handle(%sPath, %sHandler)

`,
		prefix, guardNote,
		snake, snake, snake, prefix,
		prefix,
		snake, snake)
}

func addPublicProcedures(interceptorsPath string, procedures []string) {
	content, err := readTextFile(interceptorsPath)
	if err != nil {
		fmt.Printf("[warn] Skipped public procedure registration: cannot read %s\n", interceptorsPath)
		return
	}

	start := strings.Index(content, "publicProcedures = []string{")
	if start < 0 {
		fmt.Printf("[warn] Skipped public procedure registration: unexpected structure in %s\n", interceptorsPath)
		return
	}

	var block strings.Builder
	var fresh []string
	for _, procedure := range procedures {
		if strings.Contains(content, fmt.Sprintf("%q,", procedure)) {
			continue
		}
		fresh = append(fresh, procedure)
		block.WriteString(fmt.Sprintf("\n    %q,", procedure))
	}
	if len(fresh) == 0 {
		return
	}

	rel := content[start:]
	end := strings.Index(rel, "\n}")
	if end < 0 {
		fmt.Printf("[warn] Skipped public procedure registration: unexpected structure in %s\n", interceptorsPath)
		return
	}

	updated := content[:start+end] + block.String() + content[start+end:]

	if err := writeGoFile(interceptorsPath, updated); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", interceptorsPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", interceptorsPath)
}

func MakeController(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := backendTypeOf(cfg)

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
		Resource:   prefix,
		UseRedis:   cfg.AppSpecs.Redis,
		UseAuth:    cfg.AppSpecs.Auth,
		UseRole:    cfg.AppSpecs.RBAC,
	}

	var tmplPath, targetPath string
	switch backendType {
	case "grpc":
		tmplPath = "backend/grpc/controller.go.tmpl"
		targetPath = filepath.Join(
			resolveBasePath(projectRoot, cfg.AppType, "crud"),
			utils.ToSnakeCase(prefix)+"_crud.go",
		)
	default:
		tmplPath = fmt.Sprintf("backend/%s/controller.go.tmpl", backendType)
		targetPath = filepath.Join(
			resolveBasePath(projectRoot, cfg.AppType, "controllers"),
			cleanName+".go",
		)
	}

	generateFromTemplate(tmplPath, targetPath, data)

	if backendType == "grpc" {
		snake := utils.ToSnakeCase(prefix)
		generateGrpcProtoFiles(projectRoot, cfg, data, snake)
		generateProto(projectRoot, cfg, moduleName)

		fmt.Printf("[info] The '%s' resource is served on /api/%s (REST gateway) and by the generic gohan.v1.CrudService.\n", prefix, utils.ToKebabCase(prefix))
		fmt.Printf("[info] Typed contract: proto/%s/%s.proto + services/%s_service.go (register it in routes.go to expose it over Connect).\n", snake, snake, snake)
		fmt.Println("[info] After editing the .proto file run 'make proto'.")
		fmt.Println("[info] Make sure the migrations model exists: gohan make:migration create_" + snake + "_table")
	}
}

func MakeMigration(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := backendTypeOf(cfg)

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)
	timestamp := time.Now().Format("20060102150405")

	fileName := fmt.Sprintf("%s_%s.go", timestamp, cleanName)

	baseDir := resolveBasePath(projectRoot, cfg.AppType, "database/migrations")
	targetPath := filepath.Join(baseDir, fileName)
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
	}

	tmplPath := fmt.Sprintf("backend/%s/migration.go.tmpl", backendType)
	generateFromTemplate(tmplPath, targetPath, data)

	appendMigrationToDefault(baseDir, prefix)
}

func MakeSeeder(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	backendType := backendTypeOf(cfg)

	cleanName := strings.TrimSuffix(name, ".go")
	prefix := toPascalCase(cleanName)

	baseDir := resolveBasePath(projectRoot, cfg.AppType, "database/seeders")
	targetPath := filepath.Join(baseDir, cleanName+".go")
	moduleName := utils.GetModuleName()

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
	}

	tmplPath := fmt.Sprintf("backend/%s/seeder.go.tmpl", backendType)
	generateFromTemplate(tmplPath, targetPath, data)

	appendSeederToDefault(baseDir, prefix)
}

type ResourceGuard struct {
	Auth  bool
	Roles []string
}

var promptResourceGuardFunc = promptResourceGuard

func promptResourceGuard(resource string) (*ResourceGuard, error) {
	guard := &ResourceGuard{}

	authChoice, err := utils.RunSelect(
		fmt.Sprintf("Protect the %s routes with auth middleware?", resource),
		[]string{
			"Yes (Default)",
			"No (Public)",
		},
	)
	if err != nil {
		return nil, err
	}
	if strings.Contains(authChoice, "No") {
		return guard, nil
	}
	guard.Auth = true

	roleChoice, err := utils.RunSelect(
		"Restrict access to specific roles?",
		[]string{
			"No, any authenticated user (Default)",
			"Yes, specific roles",
		},
	)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(roleChoice, "Yes") {
		return guard, nil
	}

	rolesInput, err := utils.RunInput("Allowed roles (separate with commas, e.g. admin,editor):")
	if err != nil {
		return nil, err
	}

	guard.Roles = parseRolesInput(rolesInput)
	if len(guard.Roles) == 0 {
		fmt.Println("[info] No valid roles provided. Falling back to any authenticated user.")
	}

	return guard, nil
}

func parseRolesInput(input string) []string {
	var roles []string
	for _, part := range strings.Split(input, ",") {
		role := strings.TrimSpace(part)
		if role != "" {
			roles = append(roles, role)
		}
	}
	return roles
}

func MakeResource(name string) {
	cfg, projectRoot, err := getGohanConfig()
	if err != nil {
		fmt.Printf("[error] %v\n", err)
		return
	}

	if !requireFullstack(cfg) {
		return
	}

	backendType := backendTypeOf(cfg)
	frontendType := frontendTypeOf(cfg)

	switch backendType {
	case "rest", "grpc":
		// supported
	default:
		fmt.Printf("[error] 'make:resource' is not supported for the '%s' backend architecture yet.\n", backendType)
		return
	}
	switch frontendType {
	case "vue", "react", "angular", "astro":
		// supported
	default:
		fmt.Printf("[error] 'make:resource' is not supported for the '%s' frontend yet.\n", frontendType)
		return
	}

	name = strings.TrimSpace(strings.TrimSuffix(name, ".go"))
	if name == "" {
		fmt.Println("[error] Resource name is required. Example: gohan make:resource product")
		return
	}

	prefix := toPascalCase(name)
	snake := utils.ToSnakeCase(name)
	kebab := utils.ToKebabCase(name)
	moduleName := utils.GetModuleName()

	guard, err := promptResourceGuardFunc(prefix)
	if err != nil {
		fmt.Println("[info] Resource generation cancelled.")
		return
	}

	if !cfg.AppSpecs.Auth {
		guard.Auth = false
		guard.Roles = nil
	} else if !cfg.AppSpecs.RBAC {
		guard.Roles = nil
	}

	data := templates.MakeData{
		ModuleName: moduleName,
		Prefix:     prefix,
		Resource:   prefix,
		UseRedis:   cfg.AppSpecs.Redis,
		UseAuth:    cfg.AppSpecs.Auth,
		UseRole:    cfg.AppSpecs.RBAC,
		Roles:      guard.Roles,
		Public:     !guard.Auth,
	}

	fmt.Printf("🧩 Generating %s resource...\n", prefix)

	// ---------- Backend ----------
	backendBase := resolveBasePath(projectRoot, cfg.AppType, "")

	if backendType == "grpc" {
		generateFromTemplate(
			"backend/grpc/controller.go.tmpl",
			filepath.Join(backendBase, "crud", snake+"_crud.go"),
			data,
		)
		generateGrpcProtoFiles(projectRoot, cfg, data, snake)
	} else {
		generateFromTemplate(
			fmt.Sprintf("backend/%s/controller.go.tmpl", backendType),
			filepath.Join(backendBase, "controllers", snake+"_controller.go"),
			data,
		)
	}

	timestamp := time.Now().Format("20060102150405")
	generateFromTemplate(
		fmt.Sprintf("backend/%s/migration.go.tmpl", backendType),
		filepath.Join(backendBase, "database", "migrations", fmt.Sprintf("%s_create_%s_table.go", timestamp, snake)),
		data,
	)
	appendMigrationToDefault(filepath.Join(backendBase, "database", "migrations"), prefix)

	generateFromTemplate(
		fmt.Sprintf("backend/%s/seeder.go.tmpl", backendType),
		filepath.Join(backendBase, "database", "seeders", snake+"_seeder.go"),
		data,
	)
	appendSeederToDefault(filepath.Join(backendBase, "database", "seeders"), prefix)

	if backendType == "grpc" {
		registerGrpcService(projectRoot, cfg, moduleName, prefix, snake, guard)
		generateProto(projectRoot, cfg, moduleName)
	} else {
		registerResourceRoutes(filepath.Join(backendBase, "routes.go"), prefix, kebab, guard)
	}

	// ---------- Frontend ----------
	frontendBase := filepath.Join(projectRoot, "frontend", "src")

	var viewTemplateFile, viewFileExt, viewFileName string
	viewDir := filepath.Join(frontendBase, "views")
	switch frontendType {
	case "react":
		viewTemplateFile = "frontend/react/view.tsx.tmpl"
		viewFileExt = "tsx"
	case "astro":
		viewTemplateFile = "frontend/astro/view.tsx.tmpl"
		viewFileExt = "tsx"
	case "angular":
		viewTemplateFile = "frontend/angular/view.ts.tmpl"
		viewFileExt = "ts"
		viewDir = filepath.Join(frontendBase, "app", "views")
		viewFileName = kebab + ".view." + viewFileExt
	default: // vue
		viewTemplateFile = "frontend/vue/view.vue.tmpl"
		viewFileExt = "vue"
	}

	if viewFileName == "" {
		viewFileName = prefix + "View." + viewFileExt
	}

	generateFromTemplate(
		viewTemplateFile,
		filepath.Join(viewDir, viewFileName),
		data,
	)

	switch frontendType {
	case "react":
		registerReactResourceRoute(filepath.Join(frontendBase, "App.tsx"), prefix, kebab, guard)
		registerResourceMenuItem(filepath.Join(frontendBase, "hooks", "useMenu.ts"), prefix, kebab, guard)
	case "angular":
		registerAngularResourceRoute(filepath.Join(frontendBase, "app", "app.routes.ts"), prefix, kebab, guard)
		registerResourceMenuItem(filepath.Join(frontendBase, "app", "services", "menu.service.ts"), prefix, kebab, guard)
	case "astro":
		registerReactResourceRoute(filepath.Join(frontendBase, "App.tsx"), prefix, kebab, guard)
		registerAstroResourcePage(frontendBase, kebab)
		registerResourceMenuItem(filepath.Join(frontendBase, "hooks", "useMenu.ts"), prefix, kebab, guard)
	default: // vue
		registerResourceRoute(filepath.Join(frontendBase, "router", "index.ts"), prefix, kebab, guard)
		registerResourceMenuItem(filepath.Join(frontendBase, "composables", "useMenu.ts"), prefix, kebab, guard)
	}

	fmt.Println("\n✅ Resource generated successfully!")
	if backendType == "grpc" {
		fmt.Printf("   Connect: /%s.v1.%sService | REST: /api/%s\n", snake, prefix, kebab)
		fmt.Printf("   Edit proto/%s/%s.proto (or services/%s_service.go), then run 'make proto' after changing the .proto.\n", snake, snake, snake)
	}
	fmt.Printf("   Next: run 'make dev' (migrations & seeders run automatically), then check /%s on the frontend.\n", kebab)
}

func registerResourceRoutes(routesPath, prefix, kebab string, guard *ResourceGuard) {
	content, err := readTextFile(routesPath)
	if err != nil {
		fmt.Printf("[warn] Skipped routes registration: cannot read %s\n", routesPath)
		return
	}

	endpoint := "/api/" + kebab
	check := fmt.Sprintf("%sIndex(db)", prefix)
	if strings.Contains(content, check) {
		fmt.Printf("[info] Routes for %s already registered in routes.go\n", prefix)
		return
	}

	block := fmt.Sprintf(`
    // %s Resource Routes
    gohan.Get("%s", %scontrollers.%sIndex(db)%s)
    gohan.Post("%s", %scontrollers.%sStore(db)%s)
    gohan.Get("%s/{id}", %scontrollers.%sShow(db)%s)
    gohan.Put("%s/{id}", %scontrollers.%sUpdate(db)%s)
    gohan.Delete("%s/{id}", %scontrollers.%sDestroy(db)%s)
`,
		prefix,
		endpoint, guard.Middleware(), prefix, guard.ClosingParens(),
		endpoint, guard.Middleware(), prefix, guard.ClosingParens(),
		endpoint, guard.Middleware(), prefix, guard.ClosingParens(),
		endpoint, guard.Middleware(), prefix, guard.ClosingParens(),
		endpoint, guard.Middleware(), prefix, guard.ClosingParens(),
	)

	updated, ok := appendToLastMarker(content, "\n}", block+"\n")
	if !ok {
		fmt.Printf("[warn] Skipped routes registration: unexpected structure in %s\n", routesPath)
		return
	}

	if guard.Auth && strings.Contains(block, "middleware.") {
		if !strings.Contains(updated, "/middleware\"") {
			imp := fmt.Sprintf("    \"%s/middleware\"\n", utils.GetModuleName())
			var imported bool
			updated, imported = appendBeforeFirstMarker(updated, ")\n\nfunc SetupRoutes", imp)
			if !imported {
				fmt.Println("[warn] Could not add the middleware import automatically. Add it manually to routes.go.")
			}
		}
	}

	if err := writeGoFile(routesPath, updated); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", routesPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", routesPath)
}

func (g *ResourceGuard) Middleware() string {
	if g == nil || !g.Auth {
		return ""
	}
	if len(g.Roles) > 0 {
		return fmt.Sprintf("middleware.Authenticate(db, middleware.RequireRole([]string{%s}, ", formatRolesGo(g.Roles))
	}
	return "middleware.Authenticate(db, "
}

func (g *ResourceGuard) ClosingParens() string {
	if g == nil || !g.Auth {
		return ""
	}
	if len(g.Roles) > 0 {
		return "))"
	}
	return ")"
}

func formatRolesGo(roles []string) string {
	quoted := make([]string, 0, len(roles))
	for _, role := range roles {
		quoted = append(quoted, fmt.Sprintf("%q", role))
	}
	return strings.Join(quoted, ", ")
}

func formatRolesTS(roles []string) string {
	quoted := make([]string, 0, len(roles))
	for _, role := range roles {
		quoted = append(quoted, fmt.Sprintf("'%s'", role))
	}
	return strings.Join(quoted, ", ")
}

func registerResourceRoute(routerPath, prefix, kebab string, guard *ResourceGuard) {
	content, err := readTextFile(routerPath)
	if err != nil {
		fmt.Printf("[warn] Skipped router registration: cannot read %s\n", routerPath)
		return
	}

	updated, status := registerResourceRouteString(content, prefix, kebab, guard)
	switch status {
	case regAlreadyExists:
		fmt.Printf("[info] Route for %s already registered in router/index.ts\n", prefix)
		return
	case regFailed:
		fmt.Printf("[warn] Skipped router registration: unexpected structure in %s\n", routerPath)
		return
	}

	if err := os.WriteFile(routerPath, []byte(updated), 0644); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", routerPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", routerPath)
}

func resourceRouteBlock(prefix, kebab string, guard *ResourceGuard) string {
	var meta string
	if guard != nil && guard.Auth {
		meta = "        meta: { \n          requiresAuth: true,\n"
		if len(guard.Roles) > 0 {
			meta += fmt.Sprintf("          roles: [%s]\n", formatRolesTS(guard.Roles))
		}
		meta += "        },\n"
	}

	return fmt.Sprintf("      { \n        path: '%s', \n        name: '%s', \n        component: () => import('@/views/%sView.vue'),\n%s      },\n",
		kebab, prefix, prefix, meta)
}

func registerResourceRouteString(content, prefix, kebab string, guard *ResourceGuard) (string, registrationStatus) {
	if strings.Contains(content, fmt.Sprintf("import('@/views/%sView.vue')", prefix)) {
		return content, regAlreadyExists
	}

	updated, ok := appendBeforeFirstMarker(content, "    ]\n  },", resourceRouteBlock(prefix, kebab, guard))
	if !ok {
		return content, regFailed
	}

	return updated, regInserted
}

func registerResourceMenuItem(menuPath, prefix, kebab string, guard *ResourceGuard) {
	content, err := readTextFile(menuPath)
	if err != nil {
		fmt.Printf("[warn] Skipped menu registration: cannot read %s\n", menuPath)
		return
	}

	updated, status := registerResourceMenuItemString(content, prefix, kebab, guard)
	switch status {
	case regAlreadyExists:
		fmt.Printf("[info] Menu item for %s already registered in useMenu.ts\n", prefix)
		return
	case regFailed:
		fmt.Printf("[warn] Skipped menu registration: unexpected structure in %s\n", menuPath)
		return
	}

	if err := os.WriteFile(menuPath, []byte(updated), 0644); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", menuPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", menuPath)
}

func resourceMenuBlock(prefix, kebab string, guard *ResourceGuard) string {
	var roles string
	if guard != nil && len(guard.Roles) > 0 {
		roles = fmt.Sprintf("    roles: [%s],\n", formatRolesTS(guard.Roles))
	}

	return fmt.Sprintf("\n  {    path: '/%s',\n    label: '%s',\n%s  },\n", kebab, prefix, roles)
}

func registerResourceMenuItemString(content, prefix, kebab string, guard *ResourceGuard) (string, registrationStatus) {
	if strings.Contains(content, fmt.Sprintf("path: '/%s'", kebab)) {
		return content, regAlreadyExists
	}

	updated, ok := appendToLastMarker(content, "\n]", resourceMenuBlock(prefix, kebab, guard))
	if !ok {
		return content, regFailed
	}

	return updated, regInserted
}

func registerReactResourceRoute(appPath, prefix, kebab string, guard *ResourceGuard) {
	content, err := readTextFile(appPath)
	if err != nil {
		fmt.Printf("[warn] Skipped route registration: cannot read %s\n", appPath)
		return
	}

	if strings.Contains(content, fmt.Sprintf("path=\"/%s\"", kebab)) {
		fmt.Printf("[info] Route for %s already registered in App.tsx\n", prefix)
		return
	}

	importBlock := fmt.Sprintf("\nimport %sView from '@/views/%sView'", prefix, prefix)
	importMarker := "\n\nfunction RequireAuth"
	if !strings.Contains(content, importMarker) {
		importMarker = "\n\nexport default function App"
	}

	updated, ok := appendBeforeFirstMarker(content, importMarker, importBlock)
	if !ok {
		fmt.Printf("[warn] Could not add the view import automatically. Add it manually to App.tsx.\n")
	} else {
		content = updated
	}

	updated, ok = appendBeforeFirstMarker(content, "\n      </Route>", reactResourceRouteBlock(prefix, kebab, guard))
	if !ok {
		fmt.Printf("[warn] Skipped route registration: unexpected structure in %s\n", appPath)
		return
	}

	if err := os.WriteFile(appPath, []byte(updated), 0644); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", appPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", appPath)
}

func reactResourceRouteBlock(prefix, kebab string, guard *ResourceGuard) string {
	if guard != nil && guard.Auth {
		roles := ""
		if len(guard.Roles) > 0 {
			roles = fmt.Sprintf(" roles={[%s]}", formatRolesTS(guard.Roles))
		}

		return fmt.Sprintf(`
        <Route
          path="/%s"
          element={
            <RequireAuth%s>
              <%sView />
            </RequireAuth>
          }
        />`, kebab, roles, prefix)
	}

	return fmt.Sprintf("\n        <Route path=\"/%s\" element={<%sView />} />", kebab, prefix)
}

func appendMigrationToDefault(baseDir, prefix string) {
	defaultPath := filepath.Join(baseDir, "default.go")
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

func registerAngularResourceRoute(routesPath, prefix, kebab string, guard *ResourceGuard) {
	content, err := readTextFile(routesPath)
	if err != nil {
		fmt.Printf("[warn] Skipped route registration: cannot read %s\n", routesPath)
		return
	}

	if strings.Contains(content, fmt.Sprintf("path: '%s'", kebab)) {
		fmt.Printf("[info] Route for %s already registered in app.routes.ts\n", prefix)
		return
	}

	importBlock := fmt.Sprintf("\nimport { %sView } from './views/%s.view'", prefix, kebab)
	updated, ok := appendBeforeFirstMarker(content, "\n\nexport const routes", importBlock)
	if !ok {
		fmt.Printf("[warn] Could not add the view import automatically. Add it manually to app.routes.ts.\n")
	} else {
		content = updated
	}

	updated, ok = appendBeforeFirstMarker(content, "\n    ],", angularResourceRouteBlock(prefix, kebab, guard))
	if !ok {
		fmt.Printf("[warn] Skipped route registration: unexpected structure in %s\n", routesPath)
		return
	}

	if err := os.WriteFile(routesPath, []byte(updated), 0644); err != nil {
		fmt.Printf("[error] Failed to update %s: %v\n", routesPath, err)
		return
	}
	fmt.Printf("[info] Updated: %s\n", routesPath)
}

func angularResourceRouteBlock(prefix, kebab string, guard *ResourceGuard) string {
	var meta string
	if guard != nil && guard.Auth {
		meta = "        canActivate: [AuthGuard],\n"
		if len(guard.Roles) > 0 {
			meta += fmt.Sprintf("        data: { roles: [%s] },\n", formatRolesTS(guard.Roles))
		}
	}

	return fmt.Sprintf("\n      {\n        path: '%s',\n        component: %sView,\n%s      },", kebab, prefix, meta)
}

func registerAstroResourcePage(frontendBase, kebab string) {
	pagePath := filepath.Join(frontendBase, "pages", kebab+".astro")
	if _, err := os.Stat(pagePath); err == nil {
		fmt.Printf("[info] Page for %s already exists in src/pages\n", kebab)
		return
	}

	page := fmt.Sprintf("---\n// Entry SPA: routing ditangani App (react-router) di sisi client.\nimport Layout from '@/layouts/Layout.astro'\n---\n\n<Layout />\n")
	if err := os.WriteFile(pagePath, []byte(page), 0644); err != nil {
		fmt.Printf("[error] Failed to create file %s: %v\n", pagePath, err)
		return
	}
	fmt.Printf("[info] Created: %s\n", pagePath)
}

func appendSeederToDefault(baseDir, prefix string) {
	defaultPath := filepath.Join(baseDir, "default.go")
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
