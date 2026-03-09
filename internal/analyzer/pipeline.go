package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/repomap/repomap/internal/scanner"
)

// ModuleSummary is the JSON structure returned by the LLM for module summarization.
type ModuleSummary struct {
	FilePath         string   `json:"file_path"`
	Summary          string   `json:"summary"`
	Responsibilities []string `json:"responsibilities"`
	Patterns         []string `json:"patterns"`
	DependenciesOn   []string `json:"dependencies_on"`
	DependendOnByHint *string `json:"depended_on_by_hint"`
}

// ArchitectureSynthesis is the JSON structure returned by the LLM for architecture synthesis.
type ArchitectureSynthesis struct {
	Narrative      string          `json:"narrative"`
	Layers         []ArchLayer     `json:"layers"`
	CriticalPaths  []CriticalPath  `json:"critical_paths"`
	StartHere      []string        `json:"start_here"`
	SystemPatterns []string        `json:"system_patterns"`
}

// ArchLayer represents an architectural layer in the synthesis output.
type ArchLayer struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	KeyModules  []string `json:"key_modules"`
}

// CriticalPath represents a critical code path in the synthesis output.
type CriticalPath struct {
	Name  string   `json:"name"`
	Steps []string `json:"steps"`
}

// DocWarning is the JSON structure returned by the LLM for doc ingestion.
type DocWarning struct {
	DocSummary    string         `json:"doc_summary"`
	Discrepancies []Discrepancy  `json:"discrepancies"`
	MissingTopics []string       `json:"missing_topics"`
	AccuracyScore float64        `json:"accuracy_score"`
}

// Discrepancy represents a single issue found in documentation.
type Discrepancy struct {
	Location string `json:"location"`
	Issue    string `json:"issue"`
	Severity string `json:"severity"`
}

// PipelineResult contains the complete output of the analysis pipeline.
type PipelineResult struct {
	Summaries    []ModuleSummary        `json:"summaries"`
	Architecture *ArchitectureSynthesis `json:"architecture,omitempty"`
	DocWarnings  []DocWarning           `json:"doc_warnings,omitempty"`
	Stats        PipelineStats          `json:"stats"`
}

// PipelineStats tracks pipeline execution statistics.
type PipelineStats struct {
	TotalTasks     int     `json:"total_tasks"`
	SucceededTasks int     `json:"succeeded_tasks"`
	FailedTasks    int     `json:"failed_tasks"`
	TotalRetries   int     `json:"total_retries"`
	TotalCost      float64 `json:"total_cost"`
}

// PipelineConfig configures the analysis pipeline.
type PipelineConfig struct {
	ModelID       string
	Provider      Provider
	Concurrency   int
	SkipSynthesis bool
	Graph         *scanner.RepoGraph
	Chunks        []scanner.Chunk
	DocChunks     []scanner.Chunk
	OnProgress    ProgressFunc
}

// Pipeline orchestrates the 3-stage LLM analysis.
type Pipeline struct {
	config    PipelineConfig
	templates *PromptTemplates
	pool      *Pool
}

// NewPipeline creates a new analysis pipeline.
func NewPipeline(cfg PipelineConfig) (*Pipeline, error) {
	templates, err := LoadPromptTemplates()
	if err != nil {
		return nil, fmt.Errorf("loading prompt templates: %w", err)
	}

	pool := NewPool(cfg.Concurrency, cfg.Provider)

	return &Pipeline{
		config:    cfg,
		templates: templates,
		pool:      pool,
	}, nil
}

// Run executes the full analysis pipeline:
// 1. Module summarization (parallel, per chunk)
// 2. Architecture synthesis (single call)
// 3. Doc ingestion (parallel, per doc chunk)
func (p *Pipeline) Run(ctx context.Context) (*PipelineResult, error) {
	result := &PipelineResult{}

	// Stage 1: Module summarization
	summaries, stats, err := p.runSummarization(ctx)
	if err != nil {
		return nil, fmt.Errorf("summarization stage: %w", err)
	}
	result.Summaries = summaries
	result.Stats = stats

	// Stage 2: Architecture synthesis (unless skipped)
	if !p.config.SkipSynthesis {
		arch, synthStats, err := p.runSynthesis(ctx, summaries)
		if err != nil {
			return nil, fmt.Errorf("synthesis stage: %w", err)
		}
		result.Architecture = arch
		result.Stats.TotalTasks += synthStats.TotalTasks
		result.Stats.SucceededTasks += synthStats.SucceededTasks
		result.Stats.FailedTasks += synthStats.FailedTasks
		result.Stats.TotalRetries += synthStats.TotalRetries
		result.Stats.TotalCost += synthStats.TotalCost
	}

	// Stage 3: Doc ingestion
	if len(p.config.DocChunks) > 0 {
		warnings, docStats, err := p.runDocIngestion(ctx, summaries)
		if err != nil {
			return nil, fmt.Errorf("doc ingestion stage: %w", err)
		}
		result.DocWarnings = warnings
		result.Stats.TotalTasks += docStats.TotalTasks
		result.Stats.SucceededTasks += docStats.SucceededTasks
		result.Stats.FailedTasks += docStats.FailedTasks
		result.Stats.TotalRetries += docStats.TotalRetries
		result.Stats.TotalCost += docStats.TotalCost
	}

	return result, nil
}

func (p *Pipeline) runSummarization(ctx context.Context) ([]ModuleSummary, PipelineStats, error) {
	tasks := make([]Task, 0, len(p.config.Chunks))

	for i, chunk := range p.config.Chunks {
		// Find the corresponding node for exports/imports
		var exports []string
		var imports []ImportRef
		for _, node := range p.config.Graph.Nodes {
			if node.Path == chunk.FilePath {
				exports = node.Exports
				for _, imp := range node.Imports {
					imports = append(imports, ImportRef{Source: imp.Source})
				}
				break
			}
		}

		// Determine language from node
		language := ""
		for _, node := range p.config.Graph.Nodes {
			if node.Path == chunk.FilePath {
				language = node.Language
				break
			}
		}

		prompt, err := p.templates.RenderSummarize(SummarizeData{
			FilePath: chunk.FilePath,
			Language: language,
			Exports:  exports,
			Imports:  imports,
			Code:     chunk.Content,
		})
		if err != nil {
			return nil, PipelineStats{}, fmt.Errorf("rendering prompt for %s: %w", chunk.FilePath, err)
		}

		tasks = append(tasks, Task{
			ID: fmt.Sprintf("summarize-%d-%s", i, chunk.FilePath),
			Request: CompletionRequest{
				Model: p.config.ModelID,
				Messages: []Message{
					{Role: "user", Content: prompt},
				},
				Temperature: 0.1,
				MaxTokens:   2048,
				JSONMode:    true,
			},
		})
	}

	results := p.pool.Run(ctx, tasks, p.config.OnProgress)

	var summaries []ModuleSummary
	stats := PipelineStats{TotalTasks: len(tasks)}

	for i, r := range results {
		stats.TotalRetries += r.Retries
		if r.Response != nil {
			stats.TotalCost += p.config.Provider.EstimateCost(
				r.Response.InputTokens, r.Response.OutputTokens, p.config.ModelID)
		}

		if r.Error != nil {
			stats.FailedTasks++
			summaries = append(summaries, ModuleSummary{
				FilePath: p.config.Chunks[i].FilePath,
				Summary:  "summary_unavailable",
			})
			continue
		}

		var summary ModuleSummary
		if err := json.Unmarshal([]byte(r.Response.Content), &summary); err != nil {
			stats.FailedTasks++
			summaries = append(summaries, ModuleSummary{
				FilePath: p.config.Chunks[i].FilePath,
				Summary:  "summary_unavailable",
			})
			continue
		}
		summary.FilePath = p.config.Chunks[i].FilePath
		stats.SucceededTasks++
		summaries = append(summaries, summary)
	}

	return summaries, stats, nil
}

func (p *Pipeline) runSynthesis(ctx context.Context, summaries []ModuleSummary) (*ArchitectureSynthesis, PipelineStats, error) {
	// Build summaries text
	var summaryLines []string
	for _, s := range summaries {
		if s.Summary == "summary_unavailable" {
			continue
		}
		summaryLines = append(summaryLines, fmt.Sprintf("## %s\n%s", s.FilePath, s.Summary))
	}

	// Build edges text
	var edgeLines []string
	for _, e := range p.config.Graph.Edges {
		edgeLines = append(edgeLines, fmt.Sprintf("%s -> %s (%s)", e.Source, e.Target, e.Kind))
	}

	// Build language list
	var languages []string
	for lang := range p.config.Graph.Metadata.LanguageBreaks {
		languages = append(languages, lang)
	}

	prompt, err := p.templates.RenderSynthesize(SynthesizeData{
		Summaries:  strings.Join(summaryLines, "\n\n"),
		Edges:      strings.Join(edgeLines, "\n"),
		RepoName:   p.config.Graph.Metadata.Name,
		TotalFiles: p.config.Graph.Metadata.TotalFiles,
		Languages:  strings.Join(languages, ", "),
	})
	if err != nil {
		return nil, PipelineStats{}, fmt.Errorf("rendering synthesis prompt: %w", err)
	}

	tasks := []Task{{
		ID: "synthesize-architecture",
		Request: CompletionRequest{
			Model: p.config.ModelID,
			Messages: []Message{
				{Role: "user", Content: prompt},
			},
			Temperature: 0.2,
			MaxTokens:   8192,
			JSONMode:    true,
		},
	}}

	results := p.pool.Run(ctx, tasks, p.config.OnProgress)
	stats := PipelineStats{TotalTasks: 1}

	r := results[0]
	stats.TotalRetries += r.Retries
	if r.Response != nil {
		stats.TotalCost += p.config.Provider.EstimateCost(
			r.Response.InputTokens, r.Response.OutputTokens, p.config.ModelID)
	}

	if r.Error != nil {
		stats.FailedTasks++
		return nil, stats, fmt.Errorf("synthesis failed: %w", r.Error)
	}

	content := strings.TrimSpace(r.Response.Content)
	if content == "" {
		stats.FailedTasks++
		return nil, stats, fmt.Errorf("synthesis returned empty response (model may have refused or timed out)")
	}

	var arch ArchitectureSynthesis
	if err := json.Unmarshal([]byte(content), &arch); err != nil {
		stats.FailedTasks++
		// Truncated JSON is typically caused by hitting the max output token limit
		if strings.HasPrefix(content, "{") && !strings.HasSuffix(content, "}") {
			return nil, stats, fmt.Errorf("synthesis response was truncated (output likely exceeded max token limit): %w", err)
		}
		return nil, stats, fmt.Errorf("parsing synthesis response: %w", err)
	}

	stats.SucceededTasks++
	return &arch, stats, nil
}

func (p *Pipeline) runDocIngestion(ctx context.Context, summaries []ModuleSummary) ([]DocWarning, PipelineStats, error) {
	// Build code summaries text for reference
	var summaryLines []string
	for _, s := range summaries {
		if s.Summary == "summary_unavailable" {
			continue
		}
		summaryLines = append(summaryLines, fmt.Sprintf("- %s: %s", s.FilePath, s.Summary))
	}
	codeSummaries := strings.Join(summaryLines, "\n")

	tasks := make([]Task, 0, len(p.config.DocChunks))
	for i, chunk := range p.config.DocChunks {
		prompt, err := p.templates.RenderIngestDocs(IngestDocsData{
			DocContent:    chunk.Content,
			CodeSummaries: codeSummaries,
		})
		if err != nil {
			return nil, PipelineStats{}, fmt.Errorf("rendering doc ingestion prompt for %s: %w", chunk.FilePath, err)
		}

		tasks = append(tasks, Task{
			ID: fmt.Sprintf("ingest-doc-%d-%s", i, chunk.FilePath),
			Request: CompletionRequest{
				Model: p.config.ModelID,
				Messages: []Message{
					{Role: "user", Content: prompt},
				},
				Temperature: 0.1,
				MaxTokens:   2048,
				JSONMode:    true,
			},
		})
	}

	results := p.pool.Run(ctx, tasks, p.config.OnProgress)
	stats := PipelineStats{TotalTasks: len(tasks)}

	var warnings []DocWarning
	for _, r := range results {
		stats.TotalRetries += r.Retries
		if r.Response != nil {
			stats.TotalCost += p.config.Provider.EstimateCost(
				r.Response.InputTokens, r.Response.OutputTokens, p.config.ModelID)
		}

		if r.Error != nil {
			stats.FailedTasks++
			continue
		}

		var warning DocWarning
		if err := json.Unmarshal([]byte(r.Response.Content), &warning); err != nil {
			stats.FailedTasks++
			continue
		}
		stats.SucceededTasks++
		warnings = append(warnings, warning)
	}

	return warnings, stats, nil
}
