package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/repomap/repomap/internal/analyzer"
	"github.com/repomap/repomap/internal/renderer"
	"github.com/repomap/repomap/internal/scanner"
)

func testData() *renderer.ReportData {
	return &renderer.ReportData{
		Summaries: []analyzer.ModuleSummary{
			{
				FilePath:         "src/main.go",
				Summary:          "Main entry point",
				Responsibilities: []string{"startup"},
				Patterns:         []string{"cli"},
			},
		},
		Architecture: &analyzer.ArchitectureSynthesis{
			Narrative: "A simple Go application",
		},
		Stats: analyzer.PipelineStats{
			TotalTasks:     1,
			SucceededTasks: 1,
		},
		Graph: &scanner.RepoGraph{
			Nodes: []scanner.Node{
				{ID: "src/main.go", Path: "src/main.go", Language: "Go", Layer: "other"},
			},
			Edges: []scanner.Edge{},
			Metadata: scanner.RepoMetadata{
				Name: "test-repo",
			},
		},
		Metadata: &scanner.RepoMetadata{
			Name: "test-repo",
		},
	}
}

func TestHandleReport(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/report", nil)
	w := httptest.NewRecorder()

	s.handleReport(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("expected application/json, got %s", ct)
	}

	var data renderer.ReportData
	if err := json.NewDecoder(w.Body).Decode(&data); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(data.Summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(data.Summaries))
	}
	if data.Summaries[0].FilePath != "src/main.go" {
		t.Fatalf("expected src/main.go, got %s", data.Summaries[0].FilePath)
	}
}

func TestHandleGraph(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/graph", nil)
	w := httptest.NewRecorder()

	s.handleGraph(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var graph scanner.RepoGraph
	if err := json.NewDecoder(w.Body).Decode(&graph); err != nil {
		t.Fatalf("decoding response: %v", err)
	}

	if len(graph.Nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(graph.Nodes))
	}
}

func TestHandleArchitecture(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/architecture", nil)
	w := httptest.NewRecorder()

	s.handleArchitecture(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleArchitectureNotFound(t *testing.T) {
	data := testData()
	data.Architecture = nil
	s := New(data, ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/architecture", nil)
	w := httptest.NewRecorder()

	s.handleArchitecture(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleModule(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/modules/src/main.go", nil)
	w := httptest.NewRecorder()

	s.handleModule(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestHandleModuleNotFound(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/api/modules/nonexistent.go", nil)
	w := httptest.NewRecorder()

	s.handleModule(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleIndex(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	s.handleIndex(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "text/html; charset=utf-8" {
		t.Fatalf("expected text/html, got %s", ct)
	}

	body := w.Body.String()
	if !contains(body, "test-repo") {
		t.Fatal("expected repo name in HTML")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	s := New(testData(), ":0")

	req := httptest.NewRequest(http.MethodPost, "/api/report", nil)
	w := httptest.NewRecorder()

	s.handleReport(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", w.Code)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
