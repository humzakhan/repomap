package cmd

import (
	"github.com/spf13/cobra"
)

// version is set at build time via ldflags.
var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "repomap [path]",
	Short: "Analyze a Git repository and produce an interactive HTML visualization",
	Long: `Repomap scans a Git repository using static analysis and LLM-generated
annotations, then produces a single self-contained HTML file that visualizes
the codebase — dependency graphs, module summaries, data model relationships,
and call flows.`,
	Version:       version,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}
