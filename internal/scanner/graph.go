package scanner

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// RepoGraph is the fully serializable representation of a codebase's structure.
type RepoGraph struct {
	Nodes    []Node       `json:"nodes"`
	Edges    []Edge       `json:"edges"`
	Modules  []Module     `json:"modules"`
	Metadata RepoMetadata `json:"metadata"`
}

// Node represents a single file in the dependency graph.
type Node struct {
	ID         string       `json:"id"`
	Path       string       `json:"path"`
	Language   string       `json:"language"`
	Layer      string       `json:"layer"`
	Symbols    []Symbol     `json:"symbols,omitempty"`
	Imports    []ImportDecl `json:"imports,omitempty"`
	Exports    []string     `json:"exports,omitempty"`
	SkipReason string       `json:"skip_reason,omitempty"`
}

// Edge represents a dependency relationship between two nodes.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Kind   string `json:"kind"` // "imports", "extends", "implements"
}

// Module represents a logical grouping of files (typically a directory).
type Module struct {
	Path      string   `json:"path"`
	Files     []string `json:"files"`
	EntryFile string   `json:"entry_file,omitempty"`
}

// RepoMetadata contains high-level information about the repository.
type RepoMetadata struct {
	Name           string             `json:"name"`
	RootPath       string             `json:"root_path"`
	TotalFiles     int                `json:"total_files"`
	ParsedFiles    int                `json:"parsed_files"`
	LanguageBreaks map[string]float64 `json:"language_breaks"`
	DetectedAt     time.Time          `json:"detected_at"`
}

// Layer constants for node classification.
const (
	LayerAPI    = "api"
	LayerService = "service"
	LayerData   = "data"
	LayerUtil   = "util"
	LayerTest   = "test"
	LayerConfig = "config"
	LayerOther  = "other"
)

// layerPatterns maps path segments to their layer classification.
var layerPatterns = map[string]string{
	"api":         LayerAPI,
	"routes":      LayerAPI,
	"controllers": LayerAPI,
	"handlers":    LayerAPI,
	"endpoints":   LayerAPI,
	"views":       LayerAPI,
	"services":    LayerService,
	"service":     LayerService,
	"domain":      LayerService,
	"core":        LayerService,
	"usecases":    LayerService,
	"models":      LayerData,
	"model":       LayerData,
	"entities":    LayerData,
	"entity":      LayerData,
	"schema":      LayerData,
	"schemas":     LayerData,
	"migrations":  LayerData,
	"db":          LayerData,
	"database":    LayerData,
	"utils":       LayerUtil,
	"util":        LayerUtil,
	"helpers":     LayerUtil,
	"helper":      LayerUtil,
	"lib":         LayerUtil,
	"shared":      LayerUtil,
	"common":      LayerUtil,
	"pkg":         LayerUtil,
	"test":        LayerTest,
	"tests":       LayerTest,
	"spec":        LayerTest,
	"__tests__":   LayerTest,
	"config":      LayerConfig,
	"configs":     LayerConfig,
	"infra":       LayerConfig,
	"deploy":      LayerConfig,
	"scripts":     LayerConfig,
}

// BuildGraph constructs a RepoGraph from walk results and parsed files.
func BuildGraph(ctx context.Context, repoRoot string, walkResult *WalkResult, parsedFiles []*ParsedFile) (*RepoGraph, error) {
	parsedMap := make(map[string]*ParsedFile)
	for _, pf := range parsedFiles {
		parsedMap[pf.Path] = pf
	}

	graph := &RepoGraph{
		Metadata: RepoMetadata{
			Name:           filepath.Base(repoRoot),
			RootPath:       repoRoot,
			TotalFiles:     walkResult.TotalFiles,
			ParsedFiles:    len(parsedFiles),
			LanguageBreaks: make(map[string]float64),
			DetectedAt:     time.Now().UTC(),
		},
	}

	// Calculate language percentages
	for lang, count := range walkResult.LanguageStats {
		if walkResult.TotalFiles > 0 {
			graph.Metadata.LanguageBreaks[lang] = float64(count) / float64(walkResult.TotalFiles) * 100
		}
	}

	// Build nodes
	fileToNode := make(map[string]string) // relPath -> nodeID
	for _, entry := range walkResult.Files {
		nodeID := entry.RelPath
		fileToNode[entry.RelPath] = nodeID

		node := Node{
			ID:         nodeID,
			Path:       entry.RelPath,
			Language:   entry.Language,
			Layer:      classifyLayer(entry.RelPath, entry.Language),
			SkipReason: entry.SkipReason,
		}

		if pf, ok := parsedMap[entry.RelPath]; ok {
			node.Symbols = append(node.Symbols, pf.Functions...)
			node.Symbols = append(node.Symbols, pf.Classes...)
			node.Symbols = append(node.Symbols, pf.Interfaces...)
			node.Imports = pf.Imports
			node.Exports = pf.Exports
		}

		graph.Nodes = append(graph.Nodes, node)
	}

	// Build edges from imports
	edgeSet := make(map[string]bool)
	for _, entry := range walkResult.Files {
		pf, ok := parsedMap[entry.RelPath]
		if !ok {
			continue
		}

		for _, imp := range pf.Imports {
			target := resolveImport(entry.RelPath, imp.Source, entry.Language, fileToNode)
			if target == "" {
				continue // external dependency, skip
			}

			edgeKey := fmt.Sprintf("%s->%s", entry.RelPath, target)
			if edgeSet[edgeKey] || entry.RelPath == target {
				continue // no duplicates, no self-edges
			}
			edgeSet[edgeKey] = true

			graph.Edges = append(graph.Edges, Edge{
				Source: entry.RelPath,
				Target: target,
				Kind:   "imports",
			})
		}
	}

	// Build modules (group files by directory)
	moduleMap := make(map[string]*Module)
	for _, entry := range walkResult.Files {
		dir := filepath.Dir(entry.RelPath)
		if dir == "." {
			dir = "root"
		}

		mod, ok := moduleMap[dir]
		if !ok {
			mod = &Module{Path: dir}
			moduleMap[dir] = mod
		}
		mod.Files = append(mod.Files, entry.RelPath)
		if entry.IsEntryPoint && mod.EntryFile == "" {
			mod.EntryFile = entry.RelPath
		}
	}

	for _, mod := range moduleMap {
		graph.Modules = append(graph.Modules, *mod)
	}

	return graph, nil
}

// classifyLayer determines the architectural layer of a file based on its path.
func classifyLayer(relPath string, language string) string {
	// Check for test files by name pattern
	base := filepath.Base(relPath)
	if strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, ".test.ts") ||
		strings.HasSuffix(base, ".test.js") ||
		strings.HasSuffix(base, ".spec.ts") ||
		strings.HasSuffix(base, ".spec.js") ||
		strings.HasPrefix(base, "test_") {
		return LayerTest
	}

	// Check path segments against known patterns
	parts := strings.Split(filepath.Dir(relPath), string(filepath.Separator))
	for _, part := range parts {
		if layer, ok := layerPatterns[strings.ToLower(part)]; ok {
			return layer
		}
	}

	return LayerOther
}

// resolveImport attempts to resolve an import path to a file in the repository.
func resolveImport(fromFile string, importPath string, language string, fileIndex map[string]string) string {
	if importPath == "" {
		return ""
	}

	switch language {
	case "TypeScript", "JavaScript":
		return resolveJSImport(fromFile, importPath, fileIndex)
	case "Python":
		return resolvePythonImport(fromFile, importPath, fileIndex)
	case "Go":
		return resolveGoImport(importPath, fileIndex)
	default:
		// For other languages, try simple relative resolution
		return resolveRelative(fromFile, importPath, fileIndex)
	}
}

func resolveJSImport(fromFile string, importPath string, fileIndex map[string]string) string {
	// Skip external packages
	if !strings.HasPrefix(importPath, ".") && !strings.HasPrefix(importPath, "/") {
		return ""
	}

	dir := filepath.Dir(fromFile)
	resolved := filepath.Clean(filepath.Join(dir, importPath))

	// Try with common extensions
	extensions := []string{"", ".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.tsx", "/index.js", "/index.jsx"}
	for _, ext := range extensions {
		candidate := resolved + ext
		if _, ok := fileIndex[candidate]; ok {
			return candidate
		}
	}

	return ""
}

func resolvePythonImport(fromFile string, importPath string, fileIndex map[string]string) string {
	// Relative imports (starting with .)
	if strings.HasPrefix(importPath, ".") {
		dir := filepath.Dir(fromFile)
		dots := 0
		for _, c := range importPath {
			if c == '.' {
				dots++
			} else {
				break
			}
		}
		// Go up (dots-1) directories
		for i := 1; i < dots; i++ {
			dir = filepath.Dir(dir)
		}
		rest := importPath[dots:]
		if rest == "" {
			return ""
		}
		modPath := filepath.Join(dir, strings.ReplaceAll(rest, ".", "/"))
		candidates := []string{modPath + ".py", filepath.Join(modPath, "__init__.py")}
		for _, c := range candidates {
			if _, ok := fileIndex[c]; ok {
				return c
			}
		}
		return ""
	}

	// Absolute imports — try to match within the repo
	modPath := strings.ReplaceAll(importPath, ".", "/")
	candidates := []string{modPath + ".py", filepath.Join(modPath, "__init__.py")}
	for _, c := range candidates {
		if _, ok := fileIndex[c]; ok {
			return c
		}
	}

	return ""
}

func resolveGoImport(importPath string, fileIndex map[string]string) string {
	// Go imports are package paths — try to match the suffix against repo paths
	// This is a heuristic: if the import ends with a path that matches a directory in the repo
	for path := range fileIndex {
		dir := filepath.Dir(path)
		if strings.HasSuffix(importPath, dir) || importPath == dir {
			return path
		}
	}
	return ""
}

func resolveRelative(fromFile string, importPath string, fileIndex map[string]string) string {
	if !strings.HasPrefix(importPath, ".") {
		return ""
	}
	dir := filepath.Dir(fromFile)
	resolved := filepath.Clean(filepath.Join(dir, importPath))
	if _, ok := fileIndex[resolved]; ok {
		return resolved
	}
	return ""
}
