package scanner

import (
	"context"
	"testing"
)

func TestBuildGraphBasic(t *testing.T) {
	walkResult := &WalkResult{
		Files: []FileEntry{
			{RelPath: "src/index.ts", Language: "TypeScript"},
			{RelPath: "src/services/user.ts", Language: "TypeScript"},
			{RelPath: "src/utils/logger.ts", Language: "TypeScript"},
		},
		LanguageStats: map[string]int{"TypeScript": 3},
		TotalFiles:    3,
	}

	parsedFiles := []*ParsedFile{
		{
			Path:     "src/index.ts",
			Language: "TypeScript",
			Imports: []ImportDecl{
				{Source: "./services/user", Names: []string{"UserService"}},
				{Source: "./utils/logger", Names: []string{"logger"}},
			},
			Functions: []Symbol{{Name: "createApp", Kind: "function"}},
			Classes:   []Symbol{{Name: "App", Kind: "class"}},
		},
		{
			Path:     "src/services/user.ts",
			Language: "TypeScript",
			Classes:  []Symbol{{Name: "UserService", Kind: "class"}},
		},
		{
			Path:     "src/utils/logger.ts",
			Language: "TypeScript",
			Exports:  []string{"logger"},
		},
	}

	graph, err := BuildGraph(context.Background(), "/test/repo", walkResult, parsedFiles)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	// Verify nodes
	if len(graph.Nodes) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(graph.Nodes))
	}

	// Verify edges (index.ts imports services/user.ts and utils/logger.ts)
	if len(graph.Edges) != 2 {
		t.Errorf("expected 2 edges, got %d", len(graph.Edges))
		for _, e := range graph.Edges {
			t.Logf("  %s -> %s (%s)", e.Source, e.Target, e.Kind)
		}
	}

	// Verify no self-edges
	for _, e := range graph.Edges {
		if e.Source == e.Target {
			t.Errorf("found self-edge: %s -> %s", e.Source, e.Target)
		}
	}

	// Verify metadata
	if graph.Metadata.Name != "repo" {
		t.Errorf("expected repo name 'repo', got %s", graph.Metadata.Name)
	}
	if graph.Metadata.TotalFiles != 3 {
		t.Errorf("expected 3 total files, got %d", graph.Metadata.TotalFiles)
	}
}

func TestBuildGraphNoDuplicateEdges(t *testing.T) {
	walkResult := &WalkResult{
		Files: []FileEntry{
			{RelPath: "a.ts", Language: "TypeScript"},
			{RelPath: "b.ts", Language: "TypeScript"},
		},
		LanguageStats: map[string]int{"TypeScript": 2},
		TotalFiles:    2,
	}

	parsedFiles := []*ParsedFile{
		{
			Path:     "a.ts",
			Language: "TypeScript",
			Imports: []ImportDecl{
				{Source: "./b", Names: []string{"foo"}},
				{Source: "./b", Names: []string{"bar"}}, // duplicate import source
			},
		},
		{Path: "b.ts", Language: "TypeScript"},
	}

	graph, err := BuildGraph(context.Background(), "/test/repo", walkResult, parsedFiles)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	if len(graph.Edges) != 1 {
		t.Errorf("expected 1 edge (no duplicates), got %d", len(graph.Edges))
	}
}

func TestClassifyLayer(t *testing.T) {
	tests := []struct {
		path     string
		language string
		expected string
	}{
		{"src/api/routes.ts", "TypeScript", LayerAPI},
		{"controllers/user.go", "Go", LayerAPI},
		{"services/auth.py", "Python", LayerService},
		{"models/user.ts", "TypeScript", LayerData},
		{"db/migrations/001.sql", "SQL", LayerData},
		{"utils/helpers.go", "Go", LayerUtil},
		{"lib/shared.ts", "TypeScript", LayerUtil},
		{"config/app.yaml", "YAML", LayerConfig},
		{"src/main.ts", "TypeScript", LayerOther},
		{"user_test.go", "Go", LayerTest},
		{"src/app.test.ts", "TypeScript", LayerTest},
		{"src/app.spec.js", "JavaScript", LayerTest},
	}

	for _, tt := range tests {
		got := classifyLayer(tt.path, tt.language)
		if got != tt.expected {
			t.Errorf("classifyLayer(%q, %q) = %q, want %q", tt.path, tt.language, got, tt.expected)
		}
	}
}

func TestBuildGraphModules(t *testing.T) {
	walkResult := &WalkResult{
		Files: []FileEntry{
			{RelPath: "src/index.ts", Language: "TypeScript", IsEntryPoint: true},
			{RelPath: "src/app.ts", Language: "TypeScript"},
			{RelPath: "lib/utils.ts", Language: "TypeScript"},
		},
		LanguageStats: map[string]int{"TypeScript": 3},
		TotalFiles:    3,
	}

	graph, err := BuildGraph(context.Background(), "/test/repo", walkResult, nil)
	if err != nil {
		t.Fatalf("BuildGraph failed: %v", err)
	}

	if len(graph.Modules) < 2 {
		t.Errorf("expected at least 2 modules, got %d", len(graph.Modules))
	}

	// Check that src module has entry file
	for _, mod := range graph.Modules {
		if mod.Path == "src" && mod.EntryFile != "src/index.ts" {
			t.Errorf("expected src module entry file to be src/index.ts, got %s", mod.EntryFile)
		}
	}
}
