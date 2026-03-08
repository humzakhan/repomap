# CLAUDE.md — Repomap

This file provides context and conventions for AI assistants working on this codebase.
Read this before making any changes.

---

## What This Project Does

Repomap is a CLI tool written in Go. You point it at a Git repository and it produces
a single self-contained HTML file that visualizes the codebase — dependency graphs,
module summaries, data model relationships, and call flows — powered by static analysis
and LLM-generated annotations.

The two core user-facing commands are:

```bash
repomap config        # connect AI providers, manage credentials
repomap ./some-repo   # analyze a repo and produce report.html
```

---

## Repository Layout

```
repomap/
├── cmd/                  # Cobra command entry points (root, config, run)
├── internal/
│   ├── scanner/          # Static analysis: file walking, Tree-sitter parsing, token counting
│   ├── planner/          # Token estimation, cost calculation, model recommendation, TUI screens
│   ├── analyzer/         # LLM provider interface + concrete implementations + worker pool
│   ├── renderer/         # HTML report generation, JSON injection into template
│   └── config/           # Config file read/write, Keychain integration
├── report/               # Frontend source — TypeScript + Vite (compiled into Go binary)
├── assets/               # models.json (bundled pricing data)
├── prompts/              # LLM prompt templates as .txt files
└── testdata/fixtures/    # Small committed repos used in scanner tests
```

Everything under `internal/` is private to this binary. Nothing is designed to be
imported as a library.

---

## Technology Decisions — Don't Revisit Without Good Reason

**Language: Go.** Not Python, not TypeScript. The binary distribution story (single
static binary, no runtime deps) is central to the product. Do not suggest rewriting
modules in another language.

**TUI: Charm ecosystem.** Bubble Tea for interactive screens, lipgloss for styling,
huh for forms and prompts, glamour for Markdown rendering in the terminal. Do not
introduce other TUI libraries.

**Parsing: Tree-sitter** via `go-tree-sitter`. All language parsing goes through
Tree-sitter. Do not use regex or string matching to extract code structure.

**Token counting: tiktoken-go.** Used as a universal approximation across all
providers. The estimate is within ~5% for all supported models — good enough.

**Report frontend: Cytoscape.js + D3 + Mermaid + Tailwind**, compiled by Vite,
embedded into the Go binary via `//go:embed`. The report is a single `.html` file
with no external dependencies. It must work fully offline.

---

## Code Conventions

### Go

- **Error wrapping:** Always wrap errors with context.
  ```go
  // correct
  return fmt.Errorf("scanning %s: %w", path, err)

  // wrong — context-free errors are useless up the call stack
  return err
  ```

- **Never swallow errors silently.** If a chunk fails during analysis, mark it
  `summary_unavailable` and continue. Log the error. Do not silently skip it.

- **Context propagation:** Every function that does I/O or makes network calls must
  accept a `context.Context` as its first argument. Goroutines must respect
  cancellation.
  ```go
  func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (string, error)
  ```

- **Concurrency via worker pools**, not unbounded goroutines. The `pool.go` in
  `internal/analyzer/` is the shared implementation. Use it.

- **Secrets must never appear in:**
  - Log output
  - Error messages
  - `fmt.Sprintf` or `fmt.Println` calls
  - The generated HTML report

  The `Config` struct implements `fmt.Stringer` with masked keys. Always use that
  when printing config state.

- **File paths:** Always use `filepath.Clean` and validate paths are within the
  target repo root before reading. No path traversal.

- **Linting:** `golangci-lint` runs in CI with `errcheck`, `staticcheck`, and
  `gosec` enabled. Fix lint errors rather than suppressing them.

### Prompts

- Prompts live in `prompts/` as `.txt` template files.
- Never hardcode prompt strings in Go source.
- Prompt templates use Go's `text/template` syntax.
- LLM output must always be **structured JSON**. Never parse free-form text.
  Use `encoding/json` with strict unmarshaling — unknown fields should error,
  not be silently ignored.

### models.json

- This file is the source of truth for model pricing and capabilities.
- Do not hardcode model IDs, context windows, or pricing anywhere in Go source.
  Always load from `models.json`.
- When adding a new model, add it to `models.json` only — no Go changes needed
  unless the provider itself is new.

---

## Provider Interface

All LLM providers implement this interface. Adding a new provider means creating a
new file in `internal/analyzer/` — nothing else changes.

```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (string, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan string, error)
    EstimateCost(inputTokens, outputTokens int, modelID string) float64
}
```

Providers are registered at startup. Only providers with valid credentials in config
(or environment variables) are registered.

---

## Credential Priority Order

When resolving credentials, this order is always followed:

```
1. --model / --provider CLI flags         (per-run override)
2. REPOMAP_API_KEY environment variable   (generic override)
3. Provider-specific env vars             (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
4. ~/.repomap/config.json                 (persistent personal config)
```

The config module handles this resolution. Do not implement credential lookup
anywhere else.

---

## The Cost Table Screen

This is the most complex TUI screen. Key behaviour to preserve:

- **Connected providers** appear first, above a visual divider.
- **Unconnected providers** appear below the divider with a `→ connect` affordance.
- Pressing `[c]` on an unconnected row launches the config wizard inline, without
  leaving the screen. On completion, the screen refreshes and the provider appears
  in the connected section.
- The `⭐` recommendation badge goes to the cheapest model that: (a) fits the full
  token budget within its context window without chunking penalty, and (b) has a
  `quality_tier` of `balanced` or above.
- Models whose context window is smaller than the estimated input show a `⚠ small ctx`
  warning. They are still selectable.

---

## Testing Conventions

- **Scanner tests:** Table-driven, against fixture repos in `testdata/fixtures/`.
  Each fixture is a minimal committed repo demonstrating a specific language or pattern.

- **Cost engine tests:** Parametrized unit tests covering every model in `models.json`.
  If you add a model to `models.json`, add a corresponding test case.

- **Provider integration tests:** Behind a `//go:build integration` build tag.
  Run with `go test -tags integration ./...` and require `REPOMAP_INTEGRATION=1`
  plus valid API keys. Never run in CI by default.

- **TUI tests:** Use `github.com/charmbracelet/x/exp/teatest` for headless Bubble Tea
  component testing.

- Do not mock the file system. Use real temporary directories via `t.TempDir()`.

---

## Running Locally

```bash
# Build
make build

# Run against a local repo
./repomap ./path/to/some-repo

# Run config wizard
./repomap config

# Run tests (unit only)
make test

# Run with integration tests (requires API keys)
REPOMAP_INTEGRATION=1 make test-integration

# Lint
make lint

# Build the report frontend
cd report && npm install && npm run build

# Full build (frontend + Go binary)
make all
```

---

## Building the Report Frontend

The report frontend lives in `report/` and is a TypeScript + Vite project.

```bash
cd report
npm install
npm run build       # outputs to report/dist/
npm run dev         # dev server with hot reload against fixture data
```

After building, run `make build` from the repo root — the Go binary embeds
`report/dist/bundle.js` and `report/dist/styles.css` at compile time via `//go:embed`.

When working on the report frontend, use `npm run dev`. It reads graph data from
`report/src/fixtures/sample.json` so you can develop without running a full analysis.

---

## What Not to Do

- **Do not add runtime dependencies to the Go binary** that require the user to
  install anything. The binary must be fully self-contained.
- **Do not write to stdout during analysis** except through the Bubble Tea renderer.
  Raw `fmt.Println` calls in the hot path will corrupt the TUI.
- **Do not store or log API keys.** If you are unsure whether something might contain
  a key, mask it.
- **Do not change the Provider interface** without updating all four concrete
  implementations and their tests.
- **Do not edit `models.json` pricing** without verifying against the official
  provider pricing pages. Stale pricing misleads users about their actual spend.
- **Do not add new LLM prompt logic inline in Go.** Add a template file to `prompts/`
  and load it. Prompts in source code are not reviewable as prompts.

---

## Useful References

- [Bubble Tea docs](https://github.com/charmbracelet/bubbletea)
- [Huh (forms)](https://github.com/charmbracelet/huh)
- [go-tree-sitter](https://github.com/smacker/go-tree-sitter)
- [tiktoken-go](https://github.com/pkoukk/tiktoken-go)
- [Cytoscape.js](https://js.cytoscape.org)
- [Anthropic Go SDK](https://github.com/anthropics/anthropic-sdk-go)
- [Cobra (CLI routing)](https://github.com/spf13/cobra)
- [goreleaser (binary releases)](https://goreleaser.com)