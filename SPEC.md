# Repomap — Implementation Spec

> A CLI tool that analyzes any Git repository and produces an interactive, self-contained HTML visualization for onboarding engineers onto foreign codebases.

---

## Table of Contents

1. [Core Thesis](#1-core-thesis)
2. [Architecture Overview](#2-architecture-overview)
3. [Technology Stack](#3-technology-stack)
4. [CLI UX Flow](#4-cli-ux-flow)
5. [Module Breakdown](#5-module-breakdown)
6. [Provider & Credential System](#6-provider--credential-system)
7. [Token Estimation & Cost Engine](#7-token-estimation--cost-engine)
8. [LLM Pipeline](#8-llm-pipeline)
9. [HTML Export](#9-html-export)
10. [Best Practices](#10-best-practices)
11. [Project Structure](#11-project-structure)
12. [Milestones](#12-milestones)

---

## 1. Core Thesis

Most developer onboarding is painful because codebases have no navigable, high-level map. Documentation is stale. Architecture diagrams don't exist or don't match reality. The mental model lives in people's heads.

Repomap solves this by treating the codebase itself as the source of truth. It performs static analysis to extract the real structure of the system, uses an LLM to add semantic understanding, and renders the result as a single interactive HTML file that any engineer can open — no server, no install, no account.

**The three principles guiding every design decision:**

1. **Accuracy over aesthetics** — the tool must reflect what the code actually does, not what the docs say it does. Static analysis is ground truth. LLM output is annotation.
2. **Cost transparency** — the user always knows what they are about to spend before a single API call is made.
3. **Zero-friction distribution** — the output is one `.html` file. It works offline, can be emailed, committed to the repo, or posted to a wiki.

---

## 2. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        repomap CLI (Go)                         │
│                                                                 │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐   │
│  │  config  │  │  scanner │  │  planner │  │   analyzer   │   │
│  │  module  │  │  module  │  │  module  │  │   module     │   │
│  └──────────┘  └──────────┘  └──────────┘  └──────────────┘   │
│       │              │              │               │           │
│       ▼              ▼              ▼               ▼           │
│  ~/.repomap/    AST + import    token count    LLM provider     │
│  config.json    graph + meta   + cost table   abstraction       │
│                                                    │            │
│                                               ┌────▼─────┐     │
│                                               │ renderer │     │
│                                               └────┬─────┘     │
│                                                    │            │
│                                            report.html          │
└─────────────────────────────────────────────────────────────────┘
```

Each module is independently testable. The analyzer module depends on an interface — swapping providers requires no changes to the scanner or renderer.

---

## 3. Technology Stack

### Language: Go

Go is the primary language for the entire CLI. Rationale:

- **Single binary distribution.** `brew install repomap` or `curl | sh` — no runtime dependencies, no version conflicts.
- **Goroutine-based concurrency.** The scan phase walks thousands of files in parallel. The analysis phase sends hundreds of concurrent LLM API calls. Both are trivial to implement safely in Go with worker pools.
- **Fast startup.** The CLI starts in ~5ms. Python takes ~300ms before user code runs — noticeable for a frequently-run tool.
- **Tree-sitter Go bindings** (`go-tree-sitter`) are mature enough for production use across all target languages.

### Terminal UI: Charm / Bubble Tea

[Charm](https://charm.sh) is an open-source company that builds Go-native terminal tooling. Their ecosystem is the correct choice here:

| Library | Purpose |
|---|---|
| `bubbletea` | React-like TUI component framework. Drives all interactive screens. |
| `lipgloss` | CSS-like terminal styling. Colors, borders, padding, layout. |
| `bubbles` | Pre-built components: tables, spinners, progress bars, text inputs. |
| `huh` | Form and prompt library. Drives the config wizard and confirmation screens. |
| `glamour` | Renders Markdown in the terminal. Used for displaying AI summaries inline. |

Bubble Tea uses an Elm-inspired Model-Update-View architecture. Every interactive screen is a `tea.Model` with a `Update(msg)` and `View()` method — predictable, testable, composable.

### Static Analysis: Tree-sitter

Tree-sitter provides incremental, error-tolerant parsing for every major language. It produces a concrete syntax tree that can be queried with S-expression patterns.

```go
// Example: extract all function declarations from a TypeScript file
query := `(function_declaration name: (identifier) @fn-name)`
```

Supported languages at launch: TypeScript, JavaScript, Python, Go, Rust, Ruby, Java. Adding a language is registering its grammar — the rest of the pipeline is language-agnostic.

### Token Counting: tiktoken-go

`tiktoken-go` is the Go port of OpenAI's tokenizer. It is used as a universal approximation for token counts across all providers. The estimates are within ~5% for all major models, which is precise enough for cost estimation.

### HTML Report: Go `html/template` + embedded assets

The report is generated using Go's standard `html/template` package. All JS, CSS, and graph data is injected at build time into a single file using Go's `embed` package.

```go
//go:embed report/dist/bundle.js
var bundleJS []byte

//go:embed report/dist/styles.css
var stylesCSS []byte
```

The frontend of the report itself is written in **TypeScript + Vite**, compiled to a single bundle, and embedded into the Go binary at build time.

### Frontend (Report): Cytoscape.js + D3 + Tailwind

| Library | Purpose |
|---|---|
| Cytoscape.js | Primary graph renderer. Architecture map, dependency graph. |
| `cytoscape-dagre` | Hierarchical auto-layout plugin for Cytoscape. |
| D3.js | Custom charts: treemaps, heatmaps, flame charts. |
| Mermaid.js | Sequence and flow diagrams for the Flow Explorer view. |
| Shiki | VS Code-grade syntax highlighting in the code panel. |
| Fuse.js | Client-side fuzzy search over the symbol index. |
| interact.js | Drag and resize for panels and sidebar. |
| Tailwind CSS | Utility-first styling. Purged to ~30KB in the final bundle. |

---

## 4. CLI UX Flow

### First Run (no config)

```
$ repomap

  ✗  No providers configured.
     Run `repomap config` to connect an AI provider.
```

### Config Wizard

```
$ repomap config

  ┌─────────────────────────────────────────────┐
  │  Repomap — Provider Setup                   │
  └─────────────────────────────────────────────┘

  Select a provider to connect:

  ❯ Anthropic
    OpenAI
    Google AI
    Groq
    ─────────────
    Skip for now

  ──────────────────────────────────────────────
  [↑↓] navigate   [enter] select   [q] quit
```

After selecting a provider:

```
  Anthropic — API Key Setup

  Get your API key at:
  https://console.anthropic.com/keys

  [Press Enter to open in browser]

  Paste API key: sk-ant-▌

  Verifying... ✓ Connected as [org: Acme Corp]

  Set as default provider? [Y/n]  Y

  Add another provider? [y/N]  N

  ✓  Config saved to ~/.repomap/config.json

  You're ready. Try:
    repomap ./your-project
```

### Analysis Run

```
$ repomap ./my-project

  ┌──────────────────────────────────────────────┐
  │  Step 1 of 5 — Scanning Repository           │
  └──────────────────────────────────────────────┘

  ▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░   847 / 1,203 files
  Parsing: src/services/auth/session.ts

  Languages detected:
    TypeScript  71%  ████████████████████░░░░░░░░
    Python      18%  █████░░░░░░░░░░░░░░░░░░░░░░░
    YAML         8%  ██░░░░░░░░░░░░░░░░░░░░░░░░░░
    Other        3%

  ✓  Scan complete in 3.2s
```

```
  ┌──────────────────────────────────────────────┐
  │  Step 2 of 5 — Cost Analysis                 │
  └──────────────────────────────────────────────┘

  Token Budget
  ├── Module summarization       ~124,000 tokens
  ├── Architecture synthesis      ~18,000 tokens
  ├── Documentation ingestion     ~34,000 tokens
  └── Total input                ~176,000 tokens
      Estimated output            ~22,000 tokens

  ──────────────────────────────────────────────
  Model                     Context    Est. Cost   Status
  ──────────────────────────────────────────────
  CONNECTED PROVIDERS
  ──────────────────────────────────────────────
  ⭐ claude-haiku-3-5        200K       $0.23      ✓ connected
     claude-sonnet-4         200K       $0.88      ✓ connected
     gpt-4o-mini             128K       $0.06      ✓ connected  ⚠ small ctx
     gpt-4o                  128K       $0.74      ✓ connected  ⚠ small ctx

  ──────────────────────────────────────────────
  OTHER AVAILABLE PROVIDERS  (not connected)
  ──────────────────────────────────────────────
     gemini-1.5-flash        1M         $0.03      → connect
     gemini-1.5-pro          1M         $0.36      → connect
     llama-3.1-70b (Groq)    131K       $0.18      → connect

  ──────────────────────────────────────────────
  [↑↓] select model   [c] connect provider   [enter] confirm   [q] quit

  ⭐ Recommended: claude-haiku-3-5 — best balance of cost and quality
     for repositories of this size.
```

If the user presses `[c]` on an unconnected provider, the config wizard opens inline, connects the provider, and returns to this screen with it now listed under connected providers.

```
  ┌──────────────────────────────────────────────┐
  │  Step 3 of 5 — Confirm                       │
  └──────────────────────────────────────────────┘

  Model:        claude-haiku-3-5  (Anthropic)
  Est. cost:    ~$0.23
  Output:       ./repomap-report.html

  Stages:
    ✓  Module summaries       (124 modules)
    ✓  Architecture synthesis
    ✓  Documentation ingestion
    ✗  Deep flow tracing      (add --deep, ~+$0.08)

  Proceed? [Y/n]
```

```
  ┌──────────────────────────────────────────────┐
  │  Step 4 of 5 — Analysis                      │
  └──────────────────────────────────────────────┘

  Summarizing modules...
  ▓▓▓▓▓▓▓▓▓▓▓▓░░░░░░░░░░   62 / 124 complete

  src/services/payment/stripe.ts    ✓
  src/services/auth/session.ts      ✓
  src/api/routes/users.ts           ✓  (in progress)
  src/models/user.ts                ⋯

  Tokens used: 84,201  |  Cost so far: $0.11
```

```
  ┌──────────────────────────────────────────────┐
  │  Step 5 of 5 — Rendering Report              │
  └──────────────────────────────────────────────┘

  ✓  Analysis complete
  ✓  Graph data serialized  (2,847 nodes, 5,103 edges)
  ✓  Report written to ./repomap-report.html  (4.2 MB)

  Final cost: $0.21  (estimated $0.23)

  Opening in browser...
```

---

## 5. Module Breakdown

### `cmd/` — Command Entry Points

```
cmd/
├── root.go       # root command, version, help
├── config.go     # `repomap config` wizard
└── run.go        # `repomap [path]` main analysis command
```

Uses [Cobra](https://github.com/spf13/cobra) for command routing. Cobra is the standard for Go CLIs (used by kubectl, Hugo, GitHub CLI).

### `internal/scanner/` — Static Analysis

Responsible for everything that does not require an LLM.

- Walk the file tree (respects `.gitignore` via `go-gitignore`)
- Parse each file with Tree-sitter, extract:
  - Function/class/interface declarations
  - Import statements → builds the dependency graph
  - Inline docstrings and comments
  - Export surface (what is public vs. private)
- Detect entry points heuristically (`main`, `index`, `app`, `server`, `cli`)
- Read Git metadata: commit frequency per file, last-modified timestamps, primary authors per file
- Parse config artifacts: `package.json`, `pyproject.toml`, `go.mod`, `docker-compose.yml`, `.env.example`
- Count tokens per chunk using tiktoken-go

Output: a `RepoGraph` struct — a fully serializable representation of the codebase structure, with no LLM involvement.

### `internal/planner/` — Cost & Model Engine

Takes the `RepoGraph` and produces a `AnalysisPlan`:

- Calculates token budget per stage
- Loads `~/.repomap/config.json` to identify connected providers
- Loads `models.json` (bundled, refreshable) to get pricing
- Scores and ranks models
- Renders the interactive cost table (Bubble Tea component)
- Handles inline provider connection flow

### `internal/analyzer/` — LLM Pipeline

Executes the `AnalysisPlan` against the selected model.

```go
type Provider interface {
    Name() string
    Complete(ctx context.Context, req CompletionRequest) (string, error)
    Stream(ctx context.Context, req CompletionRequest) (<-chan string, error)
}
```

Concrete implementations: `AnthropicProvider`, `OpenAIProvider`, `GoogleProvider`, `GroqProvider`.

Pipeline stages run sequentially. Within each stage, requests are concurrent via a bounded worker pool (default: 10 concurrent calls, configurable with `--concurrency`).

### `internal/renderer/` — HTML Export

Takes the completed `AnalysisResult` and writes `report.html`:

- Serializes graph data to JSON
- Injects JSON blob, bundled JS, and CSS into the HTML template
- Writes the single file output

### `internal/config/` — Credential Management

- Read/write `~/.repomap/config.json`
- On macOS: optionally store keys in Keychain via `go-keychain`
- On Linux: optionally use Secret Service via `go-libsecret`
- The config file stores a reference (`"keychain"`) instead of the raw key when secure storage is used

---

## 6. Provider & Credential System

### Config File Schema

```json
{
  "version": 1,
  "default_model": "claude-haiku-3-5",
  "providers": {
    "anthropic": {
      "api_key": "sk-ant-xxxx...xxxx",
      "key_storage": "config",
      "connected_at": "2026-03-08T10:22:00Z",
      "verified": true
    },
    "openai": {
      "api_key": "keychain",
      "key_storage": "keychain",
      "connected_at": "2026-03-08T10:25:00Z",
      "verified": true
    }
  },
  "preferences": {
    "budget_limit": null,
    "skip_docs": false,
    "auto_open_browser": true,
    "concurrency": 10
  }
}
```

### Priority Order (highest to lowest)

```
1. --model / --provider flags       (per-run CLI override)
2. REPOMAP_API_KEY env var          (CI/CD pipelines)
3. Provider-specific env vars       (ANTHROPIC_API_KEY, OPENAI_API_KEY, etc.)
4. ~/.repomap/config.json           (personal persistent config)
```

This ensures the tool works both as a personal dev tool (config file) and in automated pipelines (env vars), without either path blocking the other.

### Verification

On connect, the tool makes a minimal API call (e.g., list models, or a 1-token completion) to verify the key is valid before saving. Failed verification shows the error inline and lets the user re-enter.

---

## 7. Token Estimation & Cost Engine

### Token Counting Strategy

Each file is chunked using the same strategy that will be used during analysis — ensuring the estimate precisely matches actual usage:

```go
type Chunk struct {
    FilePath    string
    Stage       Stage   // Summarization | Synthesis | DocIngestion
    Tokens      int
    Content     string
}
```

Chunking rules:
- Files under 200 lines → single chunk
- Files over 200 lines → split at function/class boundaries (Tree-sitter provides these), never mid-function
- Synthesis stage → concatenated summaries, not raw code

### models.json

Bundled with the binary. Updated with `repomap config refresh-models`.

```json
{
  "updated_at": "2026-03-01",
  "models": [
    {
      "id": "claude-haiku-3-5",
      "display_name": "Claude Haiku 3.5",
      "provider": "anthropic",
      "context_window": 200000,
      "input_cost_per_million": 0.80,
      "output_cost_per_million": 4.00,
      "quality_tier": "balanced",
      "recommended_for": ["summarization", "large-repos"],
      "notes": null
    }
  ]
}
```

### Cost Calculation

```go
func EstimateCost(model ModelConfig, budget TokenBudget) CostEstimate {
    inputCost  := float64(budget.InputTokens)  / 1_000_000 * model.InputCostPerMillion
    outputCost := float64(budget.OutputTokens) / 1_000_000 * model.OutputCostPerMillion
    return CostEstimate{
        InputCost:  inputCost,
        OutputCost: outputCost,
        Total:      inputCost + outputCost,
        Warning:    budget.InputTokens > model.ContextWindow,
    }
}
```

Output tokens are estimated at ~12.5% of input tokens based on empirical testing across summarization tasks. This ratio is configurable internally.

---

## 8. LLM Pipeline

### Stage 1 — Module Summarization (parallelized)

Prompt per module:

```
You are analyzing a software module for developer documentation.

File: {filepath}
Language: {language}
Exports: {export_list}
Imports: {import_list}

Code:
{code_chunk}

Respond with JSON:
{
  "summary": "One paragraph plain-English description of what this module does.",
  "responsibilities": ["3-5 bullet points of key responsibilities"],
  "patterns": ["design patterns observed, if any"],
  "dependencies_on": ["modules this file critically depends on"],
  "depended_on_by_hint": null
}
```

Temperature: 0. Structured output. No markdown, raw JSON.

### Stage 2 — Architecture Synthesis (single call, frontier model)

Takes all module summaries + the dependency graph as input. Produces:

- System narrative (3-5 paragraphs)
- Layer identification (API / Service / Data / Util)
- Critical paths (the 3-5 most important call chains)
- "Start here" recommendations for a new engineer
- Identified design patterns at system level

This is the most expensive call and uses the best available model regardless of what was selected for Stage 1. It can be disabled with `--skip-synthesis`.

### Stage 3 — Documentation Ingestion

README, CHANGELOG, OpenAPI specs, and inline JSDoc/docstrings are passed separately. The model is asked to reconcile them against the code summaries and flag discrepancies — outdated docs are surfaced in the report as warnings.

### Retry & Error Handling

- Exponential backoff with jitter on rate limit (429) responses
- Failed chunks are retried up to 3 times, then marked as `summary_unavailable` — analysis continues, not aborted
- Token limit exceeded on a chunk → re-chunk at half size and retry automatically
- Real-time cost tracking: actual token usage from API response headers updates the running total displayed in the terminal

---

## 9. HTML Export

### Report Structure

```
report.html  (single file, ~3-6MB typical)
├── <style>          Tailwind purged + custom CSS vars
├── <script>         Full JS bundle (Cytoscape, D3, Mermaid, Shiki, Fuse, interact)
└── <script id="repomap-data" type="application/json">
    {
      "meta": { "repo": "my-project", "generated_at": "...", "model": "..." },
      "graph": { "nodes": [...], "edges": [...] },
      "summaries": { "module_path": { "summary": "...", "responsibilities": [...] } },
      "models": { "User": { "fields": [...], "used_by": [...] } },
      "flows": [ { "name": "POST /auth/login", "steps": [...] } ],
      "git": { "churn": { "filepath": 42 }, "authors": {...} },
      "search_index": [ { "id": "...", "label": "...", "keywords": "..." } ]
    }
</script>
```

### UI Layout

```
┌────────────────────────────────────────────────────────────────┐
│  🗺 repomap  │  my-project  │  [Search...]  │  ⚙  Generated by │
├──────┬───────────────────────────────────────┬─────────────────┤
│      │  [Architecture] [Flows] [Models] [Git]│                 │
│ Nav  │                                       │  Detail Panel   │
│ Tree │         Main Canvas                   │                 │
│      │    (Cytoscape / Mermaid / D3)         │  Summary        │
│      │                                       │  Source code    │
│      │         drag • pan • zoom             │  Git blame      │
│      │                                       │  Used by        │
├──────┴───────────────────────────────────────┴─────────────────┤
│  2,847 nodes  │  5,103 edges  │  124 modules  │  claude-haiku  │
└────────────────────────────────────────────────────────────────┘
```

### Color System

```css
:root {
  --bg:            #0a0a0f;
  --surface:       #111118;
  --border:        #1e1e2e;
  --text:          #f1f5f9;
  --muted:         #64748b;

  /* Node layers */
  --layer-api:     #7c3aed;   /* purple  — routes, controllers */
  --layer-service: #06b6d4;   /* cyan    — business logic      */
  --layer-data:    #f59e0b;   /* amber   — models, migrations  */
  --layer-util:    #64748b;   /* slate   — helpers, shared     */
  --layer-test:    #10b981;   /* green   — test files          */
  --layer-config:  #ec4899;   /* pink    — config, env, infra  */
}
```

---

## 10. Best Practices

### Go

- All internal packages under `internal/` — nothing is importable as a library (this is a CLI, not an SDK)
- Errors are wrapped with context using `fmt.Errorf("scanning %s: %w", path, err)` — never silently swallowed
- Concurrency via worker pools with explicit cancellation through `context.Context`. Every goroutine respects context cancellation.
- Secrets never logged. The `config` struct implements `fmt.Stringer` with masked keys.
- All file I/O uses `filepath.Clean` and validated against the target repo root — no path traversal
- `golangci-lint` in CI with `errcheck`, `staticcheck`, `gosec` enabled

### LLM Calls

- Always set a timeout per request (default: 60s) — passed via `context.WithTimeout`
- Prompts are loaded from embedded `.txt` template files, not hardcoded strings. This allows prompt iteration without recompiling.
- Output is always structured JSON. Never parse free-form LLM text. Use `encoding/json` with strict unmarshaling.
- Log actual token usage per call to a local `~/.repomap/usage.log` for the user's own auditing

### Security

- API keys are never written to stdout, never included in error messages, never logged
- On macOS, default to Keychain storage after the first successful connection. Prompt the user to opt in.
- The generated HTML report contains no API keys — only the analysis output
- `repomap config reset` performs a secure wipe of the config file and Keychain entries

### Testing

- Scanner module: table-driven tests against fixture repos (small, committed test repos in `testdata/`)
- Cost engine: unit tests for every pricing calculation, parametrized across all models
- Provider implementations: integration tests behind a `--integration` build tag, skipped in CI unless `REPOMAP_INTEGRATION=1`
- TUI components: Bubble Tea provides a `teatest` package for headless testing of interactive screens

---

## 11. Project Structure

```
repomap/
├── cmd/
│   ├── root.go
│   ├── config.go
│   └── run.go
│
├── internal/
│   ├── scanner/
│   │   ├── walker.go         # file tree traversal
│   │   ├── parser.go         # tree-sitter AST extraction
│   │   ├── graph.go          # dependency graph builder
│   │   ├── git.go            # git metadata (churn, authors)
│   │   └── tokenizer.go      # token counting + chunking
│   │
│   ├── planner/
│   │   ├── models.go         # models.json loader
│   │   ├── estimator.go      # cost calculation
│   │   ├── scorer.go         # model recommendation
│   │   └── ui/
│   │       ├── cost_table.go # Bubble Tea cost table component
│   │       └── confirm.go    # Bubble Tea confirmation screen
│   │
│   ├── analyzer/
│   │   ├── provider.go       # Provider interface
│   │   ├── anthropic.go
│   │   ├── openai.go
│   │   ├── google.go
│   │   ├── groq.go
│   │   ├── pool.go           # concurrent worker pool
│   │   └── pipeline.go       # stage orchestration
│   │
│   ├── renderer/
│   │   ├── renderer.go       # HTML template injection
│   │   └── templates/
│   │       └── report.html   # base HTML shell
│   │
│   └── config/
│       ├── config.go         # read/write config.json
│       ├── keychain_darwin.go
│       ├── keychain_linux.go
│       └── keychain_windows.go
│
├── report/                   # Frontend source (TypeScript + Vite)
│   ├── src/
│   │   ├── main.ts
│   │   ├── views/
│   │   │   ├── ArchitectureMap.ts
│   │   │   ├── FlowExplorer.ts
│   │   │   └── ModelCanvas.ts
│   │   └── components/
│   │       ├── DetailPanel.ts
│   │       ├── SearchBar.ts
│   │       └── Toolbar.ts
│   ├── package.json
│   └── vite.config.ts
│
├── assets/
│   └── models.json           # bundled model pricing data
│
├── testdata/
│   └── fixtures/             # small test repos for scanner tests
│
├── prompts/
│   ├── summarize_module.txt
│   ├── synthesize_architecture.txt
│   └── ingest_docs.txt
│
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 12. Milestones

### Phase 1 — Foundation (Weeks 1–3)
- [ ] Go project scaffold with Cobra
- [ ] `repomap config` wizard with Bubble Tea + huh
- [ ] Config file read/write with Keychain integration
- [ ] File tree walker with `.gitignore` support

### Phase 2 — Scanner (Weeks 3–5)
- [ ] Tree-sitter integration for TypeScript + Python
- [ ] Import graph extraction
- [ ] Token counting + chunking pipeline
- [ ] Git metadata extraction

### Phase 3 — Planner (Weeks 5–6)
- [ ] `models.json` + cost calculation engine
- [ ] Model recommendation scoring
- [ ] Interactive cost table UI (Bubble Tea)
- [ ] Inline provider connect flow from cost screen

### Phase 4 — Analyzer (Weeks 6–9)
- [ ] Provider interface + Anthropic implementation
- [ ] Worker pool for concurrent LLM calls
- [ ] Module summarization stage
- [ ] Architecture synthesis stage
- [ ] Add OpenAI, Google, Groq providers

### Phase 5 — Report (Weeks 9–12)
- [ ] Vite frontend scaffold
- [ ] Cytoscape architecture map
- [ ] Detail panel + search
- [ ] Mermaid flow explorer
- [ ] D3 treemap / git heatmap
- [ ] Go renderer: JSON injection + single file output

### Phase 6 — Polish & Distribution (Weeks 12–14)
- [ ] `goreleaser` for cross-platform binary releases
- [ ] Homebrew formula
- [ ] `--refresh-models` remote pricing update
- [ ] Integration test suite
- [ ] README + demo GIF

---

*Generated by Repomap spec process — March 2026*