package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseTypeScript(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "simple-ts", "src", "index.ts"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	entry := FileEntry{
		RelPath:  "src/index.ts",
		Language: "TypeScript",
	}

	pf, err := Parse(context.Background(), entry, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check functions
	funcNames := symbolNames(pf.Functions)
	assertContains(t, funcNames, "createApp", "expected function createApp")
	assertContains(t, funcNames, "start", "expected method start")

	// Check classes
	classNames := symbolNames(pf.Classes)
	assertContains(t, classNames, "App", "expected class App")

	// Check interfaces
	ifaceNames := symbolNames(pf.Interfaces)
	assertContains(t, ifaceNames, "AppConfig", "expected interface AppConfig")

	// Check imports
	if len(pf.Imports) < 2 {
		t.Errorf("expected at least 2 imports, got %d", len(pf.Imports))
	}

	importSources := make(map[string]bool)
	for _, imp := range pf.Imports {
		importSources[imp.Source] = true
	}
	if !importSources["./services/user"] {
		t.Error("expected import from ./services/user")
	}
	if !importSources["./utils/logger"] {
		t.Error("expected import from ./utils/logger")
	}

	// Check exports
	if len(pf.Exports) < 2 {
		t.Errorf("expected at least 2 exports, got %d", len(pf.Exports))
	}
}

func TestParsePython(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "simple-py", "app.py"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	entry := FileEntry{
		RelPath:  "app.py",
		Language: "Python",
	}

	pf, err := Parse(context.Background(), entry, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Check functions
	funcNames := symbolNames(pf.Functions)
	assertContains(t, funcNames, "create_app", "expected function create_app")
	assertContains(t, funcNames, "_internal_helper", "expected function _internal_helper")

	// Check classes
	classNames := symbolNames(pf.Classes)
	assertContains(t, classNames, "Application", "expected class Application")

	// Check exports (public names only)
	exportSet := make(map[string]bool)
	for _, e := range pf.Exports {
		exportSet[e] = true
	}
	if !exportSet["create_app"] {
		t.Error("expected create_app in exports")
	}
	if exportSet["_internal_helper"] {
		t.Error("_internal_helper should not be exported")
	}

	// Check imports
	if len(pf.Imports) < 2 {
		t.Errorf("expected at least 2 imports, got %d", len(pf.Imports))
	}
}

func TestParseGo(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "simple-go", "main.go"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	entry := FileEntry{
		RelPath:  "main.go",
		Language: "Go",
	}

	pf, err := Parse(context.Background(), entry, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	funcNames := symbolNames(pf.Functions)
	assertContains(t, funcNames, "main", "expected function main")

	// Check imports
	importSources := make(map[string]bool)
	for _, imp := range pf.Imports {
		importSources[imp.Source] = true
	}
	if !importSources["fmt"] {
		t.Error("expected import of fmt")
	}
}

func TestParseGoExported(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "simple-go", "lib", "helper.go"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	entry := FileEntry{
		RelPath:  "lib/helper.go",
		Language: "Go",
	}

	pf, err := Parse(context.Background(), entry, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	funcNames := symbolNames(pf.Functions)
	assertContains(t, funcNames, "Helper", "expected function Helper")

	// Helper should be exported (uppercase)
	for _, fn := range pf.Functions {
		if fn.Name == "Helper" && !fn.IsExported {
			t.Error("Helper should be marked as exported")
		}
	}
}

func TestParsePythonModel(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fixtures", "simple-py", "models", "user.py"))
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	entry := FileEntry{
		RelPath:  "models/user.py",
		Language: "Python",
	}

	pf, err := Parse(context.Background(), entry, content)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	classNames := symbolNames(pf.Classes)
	assertContains(t, classNames, "User", "expected class User")

	funcNames := symbolNames(pf.Functions)
	assertContains(t, funcNames, "__init__", "expected __init__ method")
	assertContains(t, funcNames, "to_dict", "expected to_dict method")
}

func TestParseUnsupportedLanguage(t *testing.T) {
	entry := FileEntry{
		RelPath:  "data.json",
		Language: "JSON",
	}

	_, err := Parse(context.Background(), entry, []byte(`{"key": "value"}`))
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestSupportedForParsing(t *testing.T) {
	supported := []string{"TypeScript", "JavaScript", "Python", "Go", "Rust", "Ruby", "Java"}
	for _, lang := range supported {
		if !SupportedForParsing(lang) {
			t.Errorf("%s should be supported for parsing", lang)
		}
	}

	unsupported := []string{"JSON", "YAML", "Markdown", "HTML", "Other"}
	for _, lang := range unsupported {
		if SupportedForParsing(lang) {
			t.Errorf("%s should not be supported for parsing", lang)
		}
	}
}

// --- helpers ---

func symbolNames(symbols []Symbol) []string {
	var names []string
	for _, s := range symbols {
		names = append(names, s.Name)
	}
	return names
}

func assertContains(t *testing.T, slice []string, item string, msg string) {
	t.Helper()
	for _, s := range slice {
		if s == item {
			return
		}
	}
	t.Errorf("%s (got: %v)", msg, slice)
}
