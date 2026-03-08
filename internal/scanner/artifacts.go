package scanner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ProjectArtifacts contains metadata extracted from config files in the repo.
type ProjectArtifacts struct {
	PackageName     string            `json:"package_name,omitempty"`
	PackageVersion  string            `json:"package_version,omitempty"`
	Dependencies    map[string]string `json:"dependencies,omitempty"`
	DevDependencies map[string]string `json:"dev_dependencies,omitempty"`
	Scripts         map[string]string `json:"scripts,omitempty"`
	GoModule        string            `json:"go_module,omitempty"`
}

// ExtractArtifacts reads and parses project config files from the repo root.
func ExtractArtifacts(repoRoot string) (*ProjectArtifacts, error) {
	artifacts := &ProjectArtifacts{
		Dependencies:    make(map[string]string),
		DevDependencies: make(map[string]string),
		Scripts:         make(map[string]string),
	}

	// Try package.json
	if err := parsePackageJSON(repoRoot, artifacts); err != nil {
		// Non-fatal: file might not exist
		_ = err
	}

	// Try go.mod
	if err := parseGoMod(repoRoot, artifacts); err != nil {
		_ = err
	}

	return artifacts, nil
}

func parsePackageJSON(repoRoot string, artifacts *ProjectArtifacts) error {
	path := filepath.Join(repoRoot, "package.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading package.json: %w", err)
	}

	var pkg struct {
		Name            string            `json:"name"`
		Version         string            `json:"version"`
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
		Scripts         map[string]string `json:"scripts"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return fmt.Errorf("parsing package.json: %w", err)
	}

	artifacts.PackageName = pkg.Name
	artifacts.PackageVersion = pkg.Version
	for k, v := range pkg.Dependencies {
		artifacts.Dependencies[k] = v
	}
	for k, v := range pkg.DevDependencies {
		artifacts.DevDependencies[k] = v
	}
	for k, v := range pkg.Scripts {
		artifacts.Scripts[k] = v
	}

	return nil
}

func parseGoMod(repoRoot string, artifacts *ProjectArtifacts) error {
	path := filepath.Join(repoRoot, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("reading go.mod: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "module ") {
			artifacts.GoModule = strings.TrimPrefix(line, "module ")
			continue
		}

		if line == "require (" {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				artifacts.Dependencies[parts[0]] = parts[1]
			}
		}

		// Single-line require
		if strings.HasPrefix(line, "require ") && !strings.Contains(line, "(") {
			parts := strings.Fields(line)
			if len(parts) >= 3 {
				artifacts.Dependencies[parts[1]] = parts[2]
			}
		}
	}

	return nil
}
