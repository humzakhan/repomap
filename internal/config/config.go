package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Config represents the persistent user configuration.
type Config struct {
	Version      int                       `json:"version"`
	DefaultModel string                    `json:"default_model"`
	Providers    map[string]ProviderConfig `json:"providers"`
	Preferences  Preferences               `json:"preferences"`
}

// ProviderConfig stores credentials and connection state for a single provider.
type ProviderConfig struct {
	APIKey      string    `json:"api_key"`
	KeyStorage  string    `json:"key_storage"`
	ConnectedAt time.Time `json:"connected_at"`
	Verified    bool      `json:"verified"`
}

// Preferences holds user-configurable behavior.
type Preferences struct {
	BudgetLimit     *float64 `json:"budget_limit"`
	SkipDocs        bool     `json:"skip_docs"`
	AutoOpenBrowser bool     `json:"auto_open_browser"`
	Concurrency     int      `json:"concurrency"`
}

// providerEnvVars maps provider names to their expected environment variable names.
var providerEnvVars = map[string]string{
	"anthropic": "ANTHROPIC_API_KEY",
	"openai":    "OPENAI_API_KEY",
	"google":    "GOOGLE_API_KEY",
	"groq":      "GROQ_API_KEY",
}

// Default returns a new Config with sensible defaults.
func Default() *Config {
	return &Config{
		Version:   1,
		Providers: make(map[string]ProviderConfig),
		Preferences: Preferences{
			AutoOpenBrowser: true,
			Concurrency:     10,
		},
	}
}

// configDir returns the path to ~/.repomap/.
func configDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".repomap")
	}
	return filepath.Join(home, ".repomap")
}

// FilePath returns the full path to the config file.
func FilePath() string {
	return filepath.Join(configDir(), "config.json")
}

// Load reads the config from disk. Returns an error if the file doesn't exist
// or is malformed.
func Load() (*Config, error) {
	data, err := os.ReadFile(FilePath())
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var cfg Config
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	return &cfg, nil
}

// LoadFrom reads the config from a specific path.
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	var cfg Config
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	if cfg.Providers == nil {
		cfg.Providers = make(map[string]ProviderConfig)
	}

	return &cfg, nil
}

// Save writes the config to disk, creating the directory if needed.
func (c *Config) Save() error {
	return c.SaveTo(FilePath())
}

// SaveTo writes the config to a specific path.
func (c *Config) SaveTo(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing config file %s: %w", path, err)
	}

	return nil
}

// ResolveCredential returns the API key for a provider using the priority chain:
// 1. explicit override (passed as parameter, e.g. from CLI flags)
// 2. REPOMAP_API_KEY env var
// 3. Provider-specific env var (e.g. ANTHROPIC_API_KEY)
// 4. Config file value (with keychain resolution)
func (c *Config) ResolveCredential(provider string, cliOverride string) string {
	// 1. CLI override
	if cliOverride != "" {
		return cliOverride
	}

	// 2. Generic env var
	if key := os.Getenv("REPOMAP_API_KEY"); key != "" {
		return key
	}

	// 3. Provider-specific env var
	if envVar, ok := providerEnvVars[provider]; ok {
		if key := os.Getenv(envVar); key != "" {
			return key
		}
	}

	// 4. Config file
	if pc, ok := c.Providers[provider]; ok {
		if pc.KeyStorage == "keychain" {
			if key, err := keychainGet(provider); err == nil && key != "" {
				return key
			}
		}
		return pc.APIKey
	}

	return ""
}

// ConnectedProviders returns the names of all providers that have valid credentials.
func (c *Config) ConnectedProviders() []string {
	var connected []string
	for name, pc := range c.Providers {
		if pc.Verified && pc.APIKey != "" {
			connected = append(connected, name)
		}
	}
	return connected
}

// String returns a human-readable representation with masked API keys.
func (c *Config) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Config v%d\n", c.Version))
	if c.DefaultModel != "" {
		b.WriteString(fmt.Sprintf("  Default model: %s\n", c.DefaultModel))
	}
	for name, pc := range c.Providers {
		b.WriteString(fmt.Sprintf("  Provider: %s\n", name))
		b.WriteString(fmt.Sprintf("    API Key: %s\n", maskKey(pc.APIKey)))
		b.WriteString(fmt.Sprintf("    Storage: %s\n", pc.KeyStorage))
		b.WriteString(fmt.Sprintf("    Verified: %v\n", pc.Verified))
	}
	return b.String()
}

// maskKey returns a masked version of an API key, showing only the last 4 characters.
func maskKey(key string) string {
	if key == "" {
		return "(not set)"
	}
	if key == "keychain" {
		return "(keychain)"
	}
	if len(key) <= 8 {
		return "****"
	}
	return "****" + key[len(key)-4:]
}
