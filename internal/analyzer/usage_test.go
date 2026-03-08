package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLogUsage(t *testing.T) {
	// Override home directory for test
	origHome := os.Getenv("HOME")
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	defer func() {
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
	}()

	entry := UsageEntry{
		Timestamp:    time.Now(),
		RepoName:     "test-repo",
		Model:        "claude-haiku-3-5",
		Provider:     "anthropic",
		InputTokens:  10000,
		OutputTokens: 1250,
		Cost:         0.013,
		Modules:      15,
		Duration:     "2m30s",
	}

	if err := LogUsage(entry); err != nil {
		t.Fatalf("LogUsage failed: %v", err)
	}

	// Verify file was written
	logPath := filepath.Join(tmpDir, ".repomap", "usage.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading usage log: %v", err)
	}

	var parsed UsageEntry
	if err := json.Unmarshal(data[:len(data)-1], &parsed); err != nil { // trim trailing newline
		t.Fatalf("parsing usage entry: %v", err)
	}

	if parsed.RepoName != "test-repo" {
		t.Errorf("expected repo_name test-repo, got %s", parsed.RepoName)
	}
	if parsed.Model != "claude-haiku-3-5" {
		t.Errorf("expected model claude-haiku-3-5, got %s", parsed.Model)
	}
	if parsed.Cost != 0.013 {
		t.Errorf("expected cost 0.013, got %f", parsed.Cost)
	}

	// Log a second entry and verify both exist
	entry.RepoName = "second-repo"
	if err := LogUsage(entry); err != nil {
		t.Fatalf("second LogUsage failed: %v", err)
	}

	data, _ = os.ReadFile(logPath)
	lines := 0
	for _, b := range data {
		if b == '\n' {
			lines++
		}
	}
	if lines != 2 {
		t.Errorf("expected 2 log lines, got %d", lines)
	}
}
