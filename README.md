# Repomap

Repomap helps you understand an unfamiliar codebase in minutes instead of days. Instead of reading through files line by line trying to piece together how things connect, you get an interactive visual map of the entire repository — architecture, dependencies, data models, and call flows — with clear starting points for where to begin learning.

Point it at any Git repo. Get back a single `.html` file you can open in any browser, share with your team, or archive. No servers, no accounts, no ongoing costs beyond the one-time LLM analysis.

![Repomap Demo](https://i.imgur.com/h7fCew9.png)

## Quick start

### Install

**Homebrew (macOS):**

```bash
brew install humzakhan/tap/repomap
```

**From source:**

```bash
git clone https://github.com/repomap/repomap.git
cd repomap
make all    # builds frontend + Go binary
```

**Run locally after building:**

```bash
./repomap ./path/to/your-repo        # analyze a repo
./repomap config                      # connect an AI provider
./repomap --version                   # verify the build
```

### Connect an AI provider

Repomap needs access to an LLM to generate summaries and architecture analysis. Run the config wizard to connect a provider:

```bash
repomap config
```

This walks you through adding an API key for any supported provider. Keys are stored in `~/.repomap/config.json` (file permissions `0600`) with optional macOS Keychain integration.

You can also set a key via environment variable instead:

```bash
export ANTHROPIC_API_KEY=sk-ant-...
```

### Analyze a repository

```bash
repomap ./path/to/your-repo
```

This scans the codebase, sends chunks to the LLM for analysis, and writes `repomap-report.html` to the current directory. Open it in any browser — it works fully offline with no external dependencies.

## Usage

```
repomap [path]              Analyze a repository and generate a report
repomap config              Connect AI providers and manage credentials
repomap --version           Print version
```

### Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--model` | Override model selection (e.g. `claude-haiku-3-5`) | Auto-recommended |
| `--provider` | Override provider (e.g. `anthropic`, `openai`) | Auto-detected |
| `--concurrency` | Number of concurrent LLM calls | `10` |
| `-o, --output` | Output file path | `repomap-report.html` |
| `--skip-synthesis` | Skip architecture synthesis stage | `false` |
| `--deep` | Enable deep flow tracing | `false` |

### Examples

Analyze a repo with a specific model:

```bash
repomap --model claude-haiku-3-5 ./my-project
```

Write output to a custom path:

```bash
repomap -o docs/architecture.html ./my-project
```

Use a budget-friendly model with higher concurrency:

```bash
repomap --model gpt-4o-mini --concurrency 20 ./my-project
```

## What's in the report

The generated HTML file is a single-page interactive application containing:

- **Architecture map** — High-level system visualization showing layers, boundaries, and critical paths
- **Dependency graph** — Interactive node graph of imports and relationships across modules
- **Module summaries** — LLM-generated descriptions of each file's responsibilities, patterns, and role in the system
- **Data model canvas** — Entity relationships and data structures
- **Flow explorer** — Call flows and critical execution paths
- **Documentation audit** — Flags discrepancies between existing docs and actual code behavior
- **Fuzzy search** — Find any symbol, file, or concept across the entire analysis

The report requires no internet connection, no JavaScript CDN, and no backend. Everything is embedded.

## Supported languages

Repomap uses [Tree-sitter](https://tree-sitter.github.io/tree-sitter/) for parsing. The following languages get full structural analysis (symbols, imports, call graphs):

- TypeScript / TSX
- JavaScript / JSX
- Python
- Go
- Rust
- Ruby
- Java

Other file types (JSON, YAML, Markdown, SQL, shell scripts, etc.) are included in token counts and language statistics but are not parsed for structure.

## Supported providers and models

| Provider | Models | Env variable |
|----------|--------|--------------|
| **Anthropic** | Claude Haiku 3.5, Claude Sonnet 4 | `ANTHROPIC_API_KEY` |
| **OpenAI** | GPT-4o Mini, GPT-4o | `OPENAI_API_KEY` |
| **Google** | Gemini 2.5 Flash, Gemini 2.5 Pro | `GOOGLE_API_KEY` |
| **Groq** | Llama 3.1 70B | `GROQ_API_KEY` |
| **Kimi** | Kimi K2, Kimi K2.5 | `MOONSHOT_API_KEY` |

Repomap automatically recommends the cheapest model that fits your codebase within its context window at `balanced` quality or above. You can override this with `--model` or `--provider`.

### Credential resolution order

Credentials are resolved in this order — the first match wins:

1. `--model` / `--provider` CLI flags
2. `REPOMAP_API_KEY` environment variable (works with any provider)
3. Provider-specific environment variable (e.g. `ANTHROPIC_API_KEY`)
4. `~/.repomap/config.json` (set via `repomap config`)

## How it works

Repomap runs a three-stage pipeline:

1. **Scan** — Walks the repository, respects `.gitignore`, parses source files with Tree-sitter, extracts symbols and imports, counts tokens, and builds a dependency graph.

2. **Analyze** — Sends code chunks to the selected LLM in parallel via a worker pool. Three passes:
   - *Module summarization* — per-file analysis of responsibilities, patterns, and dependencies
   - *Architecture synthesis* — system-level overview of layers, critical paths, and entry points
   - *Documentation ingestion* — validates existing docs against actual code behavior

3. **Render** — Injects all analysis results as JSON into an HTML template with an embedded TypeScript frontend (Cytoscape.js, D3, Mermaid, Tailwind). Outputs a single `.html` file.

## Configuration

Running `repomap config` launches an interactive terminal wizard where you can:

- Add or update API keys for any supported provider
- Choose between file-based or macOS Keychain storage
- Set a default model
- Configure preferences (budget limits, concurrency, auto-open browser)

Configuration is stored at `~/.repomap/config.json`.

## Building from source

**Prerequisites:**

- Go 1.23+
- Node.js 18+ and npm (for the report frontend)

```bash
# Clone
git clone https://github.com/repomap/repomap.git
cd repomap

# Build everything (frontend + binary)
make all

# Or build steps individually:
cd report && npm install && npm run build && cd ..
make build

# Run tests
make test

# Lint
make lint
```

### Development

To work on the report frontend with hot reload:

```bash
cd report
npm install
npm run dev
```

The dev server reads from fixture data at `report/src/fixtures/sample.json` so you can iterate on the UI without running a full analysis.

## License

MIT
