package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/repomap/repomap/internal/analyzer"
	"github.com/repomap/repomap/internal/config"
	"github.com/repomap/repomap/internal/planner"
	"github.com/repomap/repomap/internal/planner/ui"
	"github.com/repomap/repomap/internal/renderer"
	"github.com/repomap/repomap/internal/scanner"
	"github.com/spf13/cobra"
)

var runFlags struct {
	model         string
	provider      string
	concurrency   int
	output        string
	skipSynthesis bool
	deep          bool
}

func init() {
	runCmd := &cobra.Command{
		Use:   "analyze [path]",
		Short: "Analyze a repository and produce an HTML report",
		Args:  cobra.ExactArgs(1),
		RunE:  runAnalyze,
	}

	runCmd.Flags().StringVar(&runFlags.model, "model", "", "Override model selection")
	runCmd.Flags().StringVar(&runFlags.provider, "provider", "", "Override provider selection")
	runCmd.Flags().IntVar(&runFlags.concurrency, "concurrency", 10, "Number of concurrent LLM calls")
	runCmd.Flags().StringVarP(&runFlags.output, "output", "o", "repomap-report.html", "Output file path")
	runCmd.Flags().BoolVar(&runFlags.skipSynthesis, "skip-synthesis", false, "Skip architecture synthesis stage")
	runCmd.Flags().BoolVar(&runFlags.deep, "deep", false, "Enable deep flow tracing")

	// Also accept `repomap [path]` as the default command
	rootCmd.AddCommand(runCmd)
	rootCmd.Args = cobra.ArbitraryArgs
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) == 1 {
			return runAnalyze(cmd, args)
		}
		return cmd.Help()
	}
}

func runAnalyze(cmd *cobra.Command, args []string) error {
	startTime := time.Now()

	repoPath, err := filepath.Abs(filepath.Clean(args[0]))
	if err != nil {
		return fmt.Errorf("resolving path %s: %w", args[0], err)
	}

	// Derive output filename from repo name if not explicitly set
	if !cmd.Flags().Changed("output") {
		repoName := filepath.Base(repoPath)
		runFlags.output = repoName + "-repomap.html"
	}

	info, err := os.Stat(repoPath)
	if err != nil {
		return fmt.Errorf("accessing %s: %w", repoPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", repoPath)
	}

	// Check for .git directory
	gitDir := filepath.Join(repoPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("%s is not a git repository (no .git directory)", repoPath)
	}

	// Check for config
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "  ✗  No providers configured.")
		fmt.Fprintln(os.Stderr, "     Run `repomap config` to connect an AI provider.")
		return fmt.Errorf("loading config: %w", err)
	}

	if len(cfg.Providers) == 0 {
		fmt.Fprintln(os.Stderr, "  ✗  No providers configured.")
		fmt.Fprintln(os.Stderr, "     Run `repomap config` to connect an AI provider.")
		return fmt.Errorf("no providers configured")
	}

	// Step 1: Scan repository
	ctx := context.Background()
	fmt.Println("\n  Scanning repository...")

	walkResult, err := scanner.Walk(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("scanning repository: %w", err)
	}

	// Parse files with Tree-sitter
	var parsedFiles []*scanner.ParsedFile
	var chunks []scanner.Chunk

	for i, entry := range walkResult.Files {
		if !scanner.SupportedForParsing(entry.Language) {
			continue
		}

		content, readErr := os.ReadFile(entry.Path)
		if readErr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠  Could not read %s: %v\n", entry.RelPath, readErr)
			continue
		}

		// Classify the file for analysis filtering
		scanner.ClassifyFile(&walkResult.Files[i], content)

		pf, parseErr := scanner.Parse(ctx, entry, content)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠  Could not parse %s: %v\n", entry.RelPath, parseErr)
			continue
		}

		parsedFiles = append(parsedFiles, pf)

		// Skip chunking for files that don't need LLM analysis
		if walkResult.Files[i].SkipAnalysis {
			continue
		}

		fileChunks, chunkErr := scanner.ChunkFile(pf, content)
		if chunkErr != nil {
			fmt.Fprintf(os.Stderr, "  ⚠  Could not chunk %s: %v\n", entry.RelPath, chunkErr)
			continue
		}
		chunks = append(chunks, fileChunks...)
	}

	// Build dependency graph
	graph, err := scanner.BuildGraph(ctx, repoPath, walkResult, parsedFiles)
	if err != nil {
		return fmt.Errorf("building dependency graph: %w", err)
	}

	// Calculate token budget
	budget := scanner.CalculateBudget(chunks, nil)

	// Extract git metadata
	gitMeta, err := scanner.ExtractGitMetadata(ctx, repoPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠  Could not extract git metadata: %v\n", err)
	}

	// Extract project artifacts
	artifacts, _ := scanner.ExtractArtifacts(repoPath)

	// Count skipped files by reason
	skipCounts := map[string]int{}
	for _, entry := range walkResult.Files {
		if entry.SkipAnalysis {
			skipCounts[entry.SkipReason]++
		}
	}
	totalSkipped := 0
	for _, c := range skipCounts {
		totalSkipped += c
	}

	fmt.Printf("\n  ✓  Scan complete — %d files, %d parsed, %d languages\n\n",
		walkResult.TotalFiles, len(parsedFiles), len(walkResult.LanguageStats))

	for lang, count := range walkResult.LanguageStats {
		fmt.Printf("    %-15s %d files\n", lang, count)
	}

	if totalSkipped > 0 {
		parts := []string{}
		if n := skipCounts["test"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d test", n))
		}
		if n := skipCounts["generated"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d generated", n))
		}
		if n := skipCounts["barrel"]; n > 0 {
			parts = append(parts, fmt.Sprintf("%d barrel", n))
		}
		fmt.Printf("\n  Skipped from LLM analysis: %s\n", strings.Join(parts, ", "))
	}

	fmt.Printf("\n  Graph: %d nodes, %d edges, %d modules\n",
		len(graph.Nodes), len(graph.Edges), len(graph.Modules))
	fmt.Printf("  Token budget: ~%d input, ~%d output\n",
		budget.TotalInput, budget.EstimatedOutput)

	if gitMeta != nil {
		fmt.Printf("  Git: %d files with history\n", len(gitMeta.Churn))
	}
	if artifacts != nil && artifacts.GoModule != "" {
		fmt.Printf("  Go module: %s\n", artifacts.GoModule)
	}
	if artifacts != nil && artifacts.PackageName != "" {
		fmt.Printf("  Package: %s@%s\n", artifacts.PackageName, artifacts.PackageVersion)
	}

	// Step 2: Cost analysis — load models, estimate costs, show interactive table
	catalog, err := planner.LoadModels()
	if err != nil {
		return fmt.Errorf("loading model catalog: %w", err)
	}

	estimates := planner.EstimateAllModels(catalog, budget)
	scored := planner.ScoreModels(estimates, cfg.ConnectedProviders())

	costTable := ui.NewCostTable(scored)
	p := tea.NewProgram(costTable)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("running cost table: %w", err)
	}

	tableResult := finalModel.(ui.CostTableModel).Result()
	if tableResult.Quit || tableResult.Selected == nil {
		fmt.Println("\n  Cancelled.")
		return nil
	}

	selectedModel := tableResult.Selected

	// Step 3: Confirmation screen
	confirm := ui.NewConfirm(ui.ConfirmOptions{
		Model:         selectedModel.Model,
		Cost:          selectedModel.Total,
		Output:        runFlags.output,
		Budget:        budget,
		ModuleCount:   len(graph.Modules),
		SkipSynthesis: runFlags.skipSynthesis,
		Deep:          runFlags.deep,
	})

	p2 := tea.NewProgram(confirm)
	finalConfirm, err := p2.Run()
	if err != nil {
		return fmt.Errorf("running confirmation: %w", err)
	}

	confirmResult := finalConfirm.(ui.ConfirmModel).Result()
	if !confirmResult.Confirmed {
		fmt.Println("\n  Cancelled.")
		return nil
	}

	fmt.Printf("\n  Selected: %s (est. $%.2f)\n", selectedModel.Model.DisplayName, selectedModel.Total)

	// Step 4: Create provider and run analysis pipeline
	provider, err := createProvider(selectedModel.Model.Provider, cfg)
	if err != nil {
		return fmt.Errorf("creating provider: %w", err)
	}

	pipeline, err := analyzer.NewPipeline(analyzer.PipelineConfig{
		ModelID:       selectedModel.Model.ID,
		Provider:      provider,
		Concurrency:   runFlags.concurrency,
		SkipSynthesis: runFlags.skipSynthesis,
		Graph:         graph,
		Chunks:        chunks,
		OnProgress: func(completed, total int, result analyzer.TaskResult) {
			fmt.Printf("\r  Analyzing... %d/%d", completed, total)
		},
	})
	if err != nil {
		return fmt.Errorf("creating pipeline: %w", err)
	}

	fmt.Println("\n  Running analysis pipeline...")
	pipelineResult, err := pipeline.Run(ctx)
	if err != nil {
		return fmt.Errorf("running analysis pipeline: %w", err)
	}

	fmt.Printf("\n\n  ✓  Analysis complete — %d/%d modules summarized",
		pipelineResult.Stats.SucceededTasks, pipelineResult.Stats.TotalTasks)
	if pipelineResult.Stats.FailedTasks > 0 {
		fmt.Printf(" (%d failed)", pipelineResult.Stats.FailedTasks)
	}
	fmt.Printf("\n  Total cost: $%.4f\n", pipelineResult.Stats.TotalCost)

	// Step 5: Render HTML report
	fmt.Println("  Generating report...")

	reportData := &renderer.ReportData{
		Summaries:    pipelineResult.Summaries,
		Architecture: pipelineResult.Architecture,
		DocWarnings:  pipelineResult.DocWarnings,
		Stats:        pipelineResult.Stats,
		Graph:        graph,
		Metadata:     &graph.Metadata,
		GitMeta:      gitMeta,
	}

	if err := renderer.Render(reportData, runFlags.output); err != nil {
		return fmt.Errorf("rendering report: %w", err)
	}

	fmt.Printf("\n  ✓  Report written to %s\n", runFlags.output)

	// Log usage
	usageErr := analyzer.LogUsage(analyzer.UsageEntry{
		Timestamp:    time.Now(),
		RepoName:     graph.Metadata.Name,
		Model:        selectedModel.Model.ID,
		Provider:     selectedModel.Model.Provider,
		InputTokens:  budget.TotalInput,
		OutputTokens: budget.EstimatedOutput,
		Cost:         pipelineResult.Stats.TotalCost,
		Modules:      len(graph.Modules),
		Duration:     time.Since(startTime).String(),
	})
	if usageErr != nil {
		fmt.Fprintf(os.Stderr, "  ⚠  Could not log usage: %v\n", usageErr)
	}

	_ = artifacts

	return nil
}

func createProvider(providerName string, cfg *config.Config) (analyzer.Provider, error) {
	apiKey := cfg.ResolveCredential(providerName, "")
	if apiKey == "" {
		return nil, fmt.Errorf("no credentials found for provider %s", providerName)
	}

	switch providerName {
	case "anthropic":
		return analyzer.NewAnthropicProvider(apiKey), nil
	case "openai":
		return analyzer.NewOpenAIProvider(apiKey), nil
	case "google":
		return analyzer.NewGoogleProvider(apiKey), nil
	case "groq":
		return analyzer.NewGroqProvider(apiKey), nil
	case "kimi":
		return analyzer.NewKimiProvider(apiKey), nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}
}
