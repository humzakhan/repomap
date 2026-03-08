package analyzer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// UsageEntry represents a single analysis run for usage tracking.
type UsageEntry struct {
	Timestamp   time.Time `json:"timestamp"`
	RepoName    string    `json:"repo_name"`
	Model       string    `json:"model"`
	Provider    string    `json:"provider"`
	InputTokens int       `json:"input_tokens"`
	OutputTokens int      `json:"output_tokens"`
	Cost        float64   `json:"cost"`
	Modules     int       `json:"modules"`
	Duration    string    `json:"duration"`
}

// LogUsage appends a usage entry to ~/.repomap/usage.log.
func LogUsage(entry UsageEntry) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home directory: %w", err)
	}

	dir := filepath.Join(home, ".repomap")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating repomap directory: %w", err)
	}

	logPath := filepath.Join(dir, "usage.log")
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("opening usage log: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshaling usage entry: %w", err)
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("writing usage entry: %w", err)
	}

	return nil
}
