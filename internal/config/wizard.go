package config

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// Wizard guides the user through provider setup.
type Wizard struct {
	cfg *Config
}

// NewWizard creates a config wizard starting from the given config.
func NewWizard(cfg *Config) *Wizard {
	return &Wizard{cfg: cfg}
}

type providerInfo struct {
	name     string
	label    string
	keyURL   string
	envVar   string
	validate func(key string) error
}

var supportedProviders = []providerInfo{
	{
		name:   "anthropic",
		label:  "Anthropic",
		keyURL: "https://console.anthropic.com/keys",
		envVar: "ANTHROPIC_API_KEY",
	},
	{
		name:   "openai",
		label:  "OpenAI",
		keyURL: "https://platform.openai.com/api-keys",
		envVar: "OPENAI_API_KEY",
	},
	{
		name:   "google",
		label:  "Google AI",
		keyURL: "https://aistudio.google.com/apikey",
		envVar: "GOOGLE_API_KEY",
	},
	{
		name:   "groq",
		label:  "Groq",
		keyURL: "https://console.groq.com/keys",
		envVar: "GROQ_API_KEY",
	},
	{
		name:   "kimi",
		label:  "Kimi (Moonshot)",
		keyURL: "https://platform.moonshot.ai/console/api-keys",
		envVar: "MOONSHOT_API_KEY",
	},
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("12")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("10"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9"))
)

// Run executes the config wizard and returns the updated config.
func (w *Wizard) Run() (*Config, error) {
	fmt.Println(titleStyle.Render("\n  Repomap — Provider Setup"))

	for {
		provider, err := w.selectProvider()
		if err != nil {
			return nil, fmt.Errorf("selecting provider: %w", err)
		}
		if provider == nil {
			break // user chose "Skip"
		}

		if err := w.connectProvider(provider); err != nil {
			fmt.Println(errorStyle.Render(fmt.Sprintf("  ✗  Failed: %v", err)))
			fmt.Println()
			continue
		}

		var addAnother bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Add another provider?").
					Value(&addAnother),
			),
		)
		if err := form.Run(); err != nil {
			return nil, fmt.Errorf("prompting for another provider: %w", err)
		}
		if !addAnother {
			break
		}
	}

	return w.cfg, nil
}

func (w *Wizard) selectProvider() (*providerInfo, error) {
	options := make([]huh.Option[string], 0, len(supportedProviders)+1)
	for _, p := range supportedProviders {
		label := p.label
		if _, connected := w.cfg.Providers[p.name]; connected {
			label += " (connected)"
		}
		options = append(options, huh.NewOption(label, p.name))
	}
	options = append(options, huh.NewOption("Skip for now", "skip"))

	var selected string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a provider to connect").
				Options(options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	if selected == "skip" {
		return nil, nil
	}

	for i := range supportedProviders {
		if supportedProviders[i].name == selected {
			return &supportedProviders[i], nil
		}
	}

	return nil, fmt.Errorf("unknown provider: %s", selected)
}

func (w *Wizard) connectProvider(p *providerInfo) error {
	fmt.Printf("\n  %s — API Key Setup\n", p.label)
	fmt.Printf("  Get your API key at: %s\n\n", p.keyURL)

	var apiKey string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Paste API key").
				EchoMode(huh.EchoModePassword).
				Value(&apiKey).
				Validate(func(s string) error {
					s = strings.TrimSpace(s)
					if s == "" {
						return fmt.Errorf("API key cannot be empty")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return fmt.Errorf("reading API key: %w", err)
	}

	apiKey = strings.TrimSpace(apiKey)

	fmt.Print("  Verifying... ")
	if err := verifyProviderKey(p.name, apiKey); err != nil {
		fmt.Println(errorStyle.Render("✗ " + err.Error()))
		return fmt.Errorf("verification failed: %w", err)
	}
	fmt.Println(successStyle.Render("✓ Connected"))

	// Attempt keychain storage on macOS
	keyStorage := "config"
	if err := keychainSet(p.name, apiKey); err == nil {
		keyStorage = "keychain"
		apiKey = "keychain" // store reference, not the actual key
	}

	w.cfg.Providers[p.name] = ProviderConfig{
		APIKey:      apiKey,
		KeyStorage:  keyStorage,
		ConnectedAt: time.Now().UTC(),
		Verified:    true,
	}

	// Set as default if first provider
	if w.cfg.DefaultModel == "" {
		var setDefault bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Set %s as default provider?", p.label)).
					Value(&setDefault),
			),
		)
		if err := form.Run(); err == nil && setDefault {
			w.cfg.DefaultModel = defaultModelFor(p.name)
		}
	}

	return nil
}

// verifyProviderKey makes a minimal API call to verify the key is valid.
func verifyProviderKey(provider, apiKey string) error {
	var url string
	var authHeader string

	switch provider {
	case "anthropic":
		url = "https://api.anthropic.com/v1/messages"
		authHeader = "x-api-key"
	case "openai":
		url = "https://api.openai.com/v1/models"
		authHeader = "Authorization"
		apiKey = "Bearer " + apiKey
	case "google":
		url = fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models?key=%s", apiKey)
		authHeader = ""
	case "groq":
		url = "https://api.groq.com/openai/v1/models"
		authHeader = "Authorization"
		apiKey = "Bearer " + apiKey
	case "kimi":
		url = "https://api.moonshot.cn/v1/models"
		authHeader = "Authorization"
		apiKey = "Bearer " + apiKey
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	if authHeader != "" {
		req.Header.Set(authHeader, apiKey)
	}

	// Anthropic requires specific headers even for verification
	if provider == "anthropic" {
		req.Header.Set("anthropic-version", "2023-06-01")
		req.Header.Set("content-type", "application/json")
		// GET on messages endpoint returns 405 with valid key, 401 with invalid
		// Use a different approach: just check auth header is accepted
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connecting to %s API: %w", provider, err)
	}
	defer resp.Body.Close()

	// For Anthropic, 405 (Method Not Allowed) means the key was accepted
	if provider == "anthropic" && resp.StatusCode == http.StatusMethodNotAllowed {
		return nil
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("invalid API key (HTTP %d)", resp.StatusCode)
	}

	// 2xx or non-auth errors mean the key is valid
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// For other errors (rate limit, server error), assume key is valid
	// but the service has a transient issue
	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil
	}

	return fmt.Errorf("unexpected response from %s API (HTTP %d)", provider, resp.StatusCode)
}

// defaultModelFor returns the sensible default model for a provider.
func defaultModelFor(provider string) string {
	defaults := map[string]string{
		"anthropic": "claude-haiku-3-5",
		"openai":    "gpt-4o-mini",
		"google":    "gemini-2.5-flash",
		"groq":      "llama-3.1-70b",
		"kimi":      "kimi-k2",
	}
	if m, ok := defaults[provider]; ok {
		return m
	}
	return ""
}
