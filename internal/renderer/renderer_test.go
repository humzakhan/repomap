package renderer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/repomap/repomap/internal/analyzer"
	"github.com/repomap/repomap/internal/scanner"
)

func TestRender(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "report.html")

	data := &ReportData{
		Summaries: []analyzer.ModuleSummary{
			{
				FilePath:         "main.go",
				Summary:          "Entry point for the application.",
				Responsibilities: []string{"Bootstrap", "CLI parsing"},
			},
		},
		Stats: analyzer.PipelineStats{
			TotalTasks:     1,
			SucceededTasks: 1,
		},
		Graph: &scanner.RepoGraph{
			Nodes: []scanner.Node{
				{ID: "main.go", Path: "main.go", Language: "Go", Layer: "api"},
			},
		},
		Metadata: &scanner.RepoMetadata{
			Name:       "test-repo",
			TotalFiles: 1,
		},
	}

	if err := Render(data, outputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("reading output: %v", err)
	}

	html := string(content)

	if !strings.Contains(html, "test-repo") {
		t.Error("expected report to contain repo name")
	}
	if !strings.Contains(html, "repomap-data") {
		t.Error("expected report to contain repomap-data script tag")
	}
	if !strings.Contains(html, "main.go") {
		t.Error("expected report to contain file data")
	}
}

func TestRenderCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "subdir", "nested", "report.html")

	data := &ReportData{
		Stats: analyzer.PipelineStats{},
		Graph: &scanner.RepoGraph{},
		Metadata: &scanner.RepoMetadata{
			Name: "test",
		},
	}

	if err := Render(data, outputPath); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Error("expected output file to exist")
	}
}
