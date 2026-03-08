package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/repomap/repomap/internal/config"
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
	repoPath, err := filepath.Abs(filepath.Clean(args[0]))
	if err != nil {
		return fmt.Errorf("resolving path %s: %w", args[0], err)
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
	result, err := scanner.Walk(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("scanning repository: %w", err)
	}

	fmt.Printf("\n  ✓  Scan complete — %d files across %d languages\n\n", result.TotalFiles, len(result.LanguageStats))
	for lang, count := range result.LanguageStats {
		fmt.Printf("    %-15s %d files\n", lang, count)
	}

	// Remaining steps (planner, analyzer, renderer) will be implemented in later phases.
	fmt.Println("\n  ℹ  Analysis pipeline not yet implemented. Scan results shown above.")

	return nil
}
