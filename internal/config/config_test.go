package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if cfg.Preferences.Concurrency != 10 {
		t.Errorf("expected concurrency 10, got %d", cfg.Preferences.Concurrency)
	}
	if !cfg.Preferences.AutoOpenBrowser {
		t.Error("expected auto_open_browser to be true")
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(cfg.Providers))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	cfg.DefaultModel = "claude-haiku-3-5"
	cfg.Providers["anthropic"] = ProviderConfig{
		APIKey:      "sk-ant-test-key-1234",
		KeyStorage:  "config",
		ConnectedAt: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
		Verified:    true,
	}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	loaded, err := LoadFrom(path)
	if err != nil {
		t.Fatalf("LoadFrom failed: %v", err)
	}

	if loaded.DefaultModel != "claude-haiku-3-5" {
		t.Errorf("expected default model claude-haiku-3-5, got %s", loaded.DefaultModel)
	}
	if loaded.Version != 1 {
		t.Errorf("expected version 1, got %d", loaded.Version)
	}

	pc, ok := loaded.Providers["anthropic"]
	if !ok {
		t.Fatal("expected anthropic provider in loaded config")
	}
	if pc.APIKey != "sk-ant-test-key-1234" {
		t.Errorf("expected API key sk-ant-test-key-1234, got %s", pc.APIKey)
	}
	if !pc.Verified {
		t.Error("expected provider to be verified")
	}
}

func TestLoadInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	if err := os.WriteFile(path, []byte(`{invalid json}`), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error loading invalid JSON")
	}
}

func TestLoadUnknownFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := `{"version": 1, "unknown_field": "value", "providers": {}, "preferences": {"auto_open_browser": true, "skip_docs": false, "concurrency": 10, "budget_limit": null}}`
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadFrom(path)
	if err == nil {
		t.Fatal("expected error for unknown fields")
	}
}

func TestLoadMissingFile(t *testing.T) {
	_, err := LoadFrom("/nonexistent/path/config.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolveCredentialPriority(t *testing.T) {
	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{
		APIKey:     "config-key",
		KeyStorage: "config",
		Verified:   true,
	}

	// Priority 1: CLI override
	got := cfg.ResolveCredential("anthropic", "cli-key")
	if got != "cli-key" {
		t.Errorf("expected cli-key, got %s", got)
	}

	// Priority 4: Config file (no env vars set, no CLI override)
	got = cfg.ResolveCredential("anthropic", "")
	if got != "config-key" {
		t.Errorf("expected config-key, got %s", got)
	}
}

func TestResolveCredentialGenericEnv(t *testing.T) {
	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{
		APIKey:     "config-key",
		KeyStorage: "config",
		Verified:   true,
	}

	t.Setenv("REPOMAP_API_KEY", "generic-env-key")

	got := cfg.ResolveCredential("anthropic", "")
	if got != "generic-env-key" {
		t.Errorf("expected generic-env-key, got %s", got)
	}
}

func TestResolveCredentialProviderEnv(t *testing.T) {
	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{
		APIKey:     "config-key",
		KeyStorage: "config",
		Verified:   true,
	}

	t.Setenv("ANTHROPIC_API_KEY", "provider-env-key")

	got := cfg.ResolveCredential("anthropic", "")
	if got != "provider-env-key" {
		t.Errorf("expected provider-env-key, got %s", got)
	}
}

func TestResolveCredentialEmpty(t *testing.T) {
	cfg := Default()

	got := cfg.ResolveCredential("anthropic", "")
	if got != "" {
		t.Errorf("expected empty string, got %s", got)
	}
}

func TestConnectedProviders(t *testing.T) {
	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{APIKey: "key1", Verified: true}
	cfg.Providers["openai"] = ProviderConfig{APIKey: "key2", Verified: true}
	cfg.Providers["google"] = ProviderConfig{APIKey: "", Verified: false}

	connected := cfg.ConnectedProviders()
	if len(connected) != 2 {
		t.Errorf("expected 2 connected providers, got %d", len(connected))
	}
}

func TestStringMasksKeys(t *testing.T) {
	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{
		APIKey:     "sk-ant-very-secret-key-abcd",
		KeyStorage: "config",
		Verified:   true,
	}

	s := cfg.String()

	if strings.Contains(s, "sk-ant-very-secret-key-abcd") {
		t.Error("String() should not contain the full API key")
	}
	if !strings.Contains(s, "****abcd") {
		t.Errorf("String() should show masked key with last 4 chars, got: %s", s)
	}
}

func TestMaskKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "(not set)"},
		{"keychain", "(keychain)"},
		{"short", "****"},
		{"12345678", "****"},
		{"sk-ant-test-key-1234", "****1234"},
	}

	for _, tt := range tests {
		got := maskKey(tt.input)
		if got != tt.expected {
			t.Errorf("maskKey(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "deep", "config.json")

	cfg := Default()
	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo should create nested directories: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("config file should exist after SaveTo")
	}
}

func TestFilePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := Default()
	cfg.Providers["anthropic"] = ProviderConfig{APIKey: "secret", Verified: true}

	if err := cfg.SaveTo(path); err != nil {
		t.Fatalf("SaveTo failed: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat failed: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("config file should have 0600 permissions, got %o", perm)
	}
}
