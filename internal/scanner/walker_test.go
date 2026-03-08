package scanner

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func setupFixtureRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git directory to mark as a repo
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatalf("creating .git: %v", err)
	}

	// Create source files
	files := map[string]string{
		"main.go":              "package main\n\nfunc main() {}\n",
		"lib/helper.go":        "package lib\n\nfunc Helper() {}\n",
		"lib/utils.go":         "package lib\n\nfunc Utils() {}\n",
		"src/index.ts":         "export const hello = 'world';\n",
		"src/app.ts":           "import { hello } from './index';\n",
		"src/components/btn.tsx": "export function Button() { return <button/>; }\n",
		"tests/main_test.go":   "package tests\n\nfunc TestMain() {}\n",
		"README.md":            "# Test Repo\n",
		"config.yaml":          "key: value\n",
	}

	for path, content := range files {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("creating dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	return dir
}

func setupFixtureWithGitignore(t *testing.T) string {
	t.Helper()
	dir := setupFixtureRepo(t)

	// Add .gitignore
	gitignore := "*.log\nbuild/\ntmp/\n"
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatalf("writing .gitignore: %v", err)
	}

	// Create files that should be ignored
	ignoredFiles := map[string]string{
		"debug.log":        "log data",
		"build/output.js":  "compiled",
		"tmp/cache.json":   "cached data",
	}
	for path, content := range ignoredFiles {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("creating dir for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("writing %s: %v", path, err)
		}
	}

	return dir
}

func TestWalkBasic(t *testing.T) {
	dir := setupFixtureRepo(t)
	ctx := context.Background()

	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if result.TotalFiles != 9 {
		t.Errorf("expected 9 files, got %d", result.TotalFiles)
		for _, f := range result.Files {
			t.Logf("  %s (%s)", f.RelPath, f.Language)
		}
	}
}

func TestWalkLanguageDetection(t *testing.T) {
	dir := setupFixtureRepo(t)
	ctx := context.Background()

	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	expected := map[string]int{
		"Go":         4,
		"TypeScript": 3,
		"Markdown":   1,
		"YAML":       1,
	}

	for lang, count := range expected {
		if result.LanguageStats[lang] != count {
			t.Errorf("expected %d %s files, got %d", count, lang, result.LanguageStats[lang])
		}
	}
}

func TestWalkEntryPoints(t *testing.T) {
	dir := setupFixtureRepo(t)
	ctx := context.Background()

	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	entryPoints := make(map[string]bool)
	for _, f := range result.Files {
		if f.IsEntryPoint {
			entryPoints[f.RelPath] = true
		}
	}

	expectedEntries := []string{"main.go", "src/index.ts", "src/app.ts"}
	for _, path := range expectedEntries {
		if !entryPoints[path] {
			t.Errorf("expected %s to be an entry point", path)
		}
	}
}

func TestWalkGitignore(t *testing.T) {
	dir := setupFixtureWithGitignore(t)
	ctx := context.Background()

	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	// Should not contain ignored files
	for _, f := range result.Files {
		if f.RelPath == "debug.log" || f.RelPath == "build/output.js" || f.RelPath == "tmp/cache.json" {
			t.Errorf("gitignored file %s should not be in results", f.RelPath)
		}
	}

	if result.SkippedFiles < 1 {
		t.Errorf("expected at least 1 skipped file, got %d", result.SkippedFiles)
	}
}

func TestWalkSkipsNodeModules(t *testing.T) {
	dir := setupFixtureRepo(t)

	// Create node_modules
	nmPath := filepath.Join(dir, "node_modules", "some-pkg", "index.js")
	if err := os.MkdirAll(filepath.Dir(nmPath), 0755); err != nil {
		t.Fatalf("creating node_modules: %v", err)
	}
	if err := os.WriteFile(nmPath, []byte("module.exports = {}"), 0644); err != nil {
		t.Fatalf("writing node_modules file: %v", err)
	}

	ctx := context.Background()
	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	for _, f := range result.Files {
		if filepath.Base(filepath.Dir(f.RelPath)) == "node_modules" || f.RelPath == "node_modules" {
			t.Errorf("node_modules file %s should not be in results", f.RelPath)
		}
	}
}

func TestWalkContextCancellation(t *testing.T) {
	dir := setupFixtureRepo(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := Walk(ctx, dir)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestWalkNonexistentDir(t *testing.T) {
	_, err := Walk(context.Background(), "/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for nonexistent directory")
	}
}

func TestWalkNotADirectory(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(filePath, []byte("hello"), 0644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	_, err := Walk(context.Background(), filePath)
	if err == nil {
		t.Fatal("expected error when walking a file instead of directory")
	}
}

func TestWalkEmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx := context.Background()

	result, err := Walk(ctx, dir)
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}

	if result.TotalFiles != 0 {
		t.Errorf("expected 0 files, got %d", result.TotalFiles)
	}
}
