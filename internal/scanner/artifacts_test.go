package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractArtifactsPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{
  "name": "my-app",
  "version": "1.0.0",
  "dependencies": {
    "express": "^4.18.0",
    "lodash": "^4.17.21"
  },
  "devDependencies": {
    "typescript": "^5.0.0"
  },
  "scripts": {
    "build": "tsc",
    "start": "node dist/index.js"
  }
}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644); err != nil {
		t.Fatalf("writing package.json: %v", err)
	}

	artifacts, err := ExtractArtifacts(dir)
	if err != nil {
		t.Fatalf("ExtractArtifacts failed: %v", err)
	}

	if artifacts.PackageName != "my-app" {
		t.Errorf("expected name my-app, got %s", artifacts.PackageName)
	}
	if artifacts.PackageVersion != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", artifacts.PackageVersion)
	}
	if len(artifacts.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(artifacts.Dependencies))
	}
	if artifacts.Dependencies["express"] != "^4.18.0" {
		t.Errorf("expected express ^4.18.0, got %s", artifacts.Dependencies["express"])
	}
	if len(artifacts.DevDependencies) != 1 {
		t.Errorf("expected 1 dev dependency, got %d", len(artifacts.DevDependencies))
	}
	if len(artifacts.Scripts) != 2 {
		t.Errorf("expected 2 scripts, got %d", len(artifacts.Scripts))
	}
}

func TestExtractArtifactsGoMod(t *testing.T) {
	dir := t.TempDir()
	gomod := `module github.com/example/myapp

go 1.22

require (
	github.com/spf13/cobra v1.10.2
	github.com/charmbracelet/bubbletea v1.3.6
)
`
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	artifacts, err := ExtractArtifacts(dir)
	if err != nil {
		t.Fatalf("ExtractArtifacts failed: %v", err)
	}

	if artifacts.GoModule != "github.com/example/myapp" {
		t.Errorf("expected module github.com/example/myapp, got %s", artifacts.GoModule)
	}
	if len(artifacts.Dependencies) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(artifacts.Dependencies))
	}
	if artifacts.Dependencies["github.com/spf13/cobra"] != "v1.10.2" {
		t.Errorf("unexpected cobra version: %s", artifacts.Dependencies["github.com/spf13/cobra"])
	}
}

func TestExtractArtifactsEmpty(t *testing.T) {
	dir := t.TempDir()

	artifacts, err := ExtractArtifacts(dir)
	if err != nil {
		t.Fatalf("ExtractArtifacts failed: %v", err)
	}

	if artifacts.PackageName != "" {
		t.Errorf("expected empty package name, got %s", artifacts.PackageName)
	}
	if artifacts.GoModule != "" {
		t.Errorf("expected empty go module, got %s", artifacts.GoModule)
	}
}
