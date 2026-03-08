package cmd

import (
	"fmt"

	"github.com/repomap/repomap/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Connect AI providers and manage credentials",
	RunE:  runConfig,
}

func init() {
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		cfg = config.Default()
	}

	wizard := config.NewWizard(cfg)
	updatedCfg, err := wizard.Run()
	if err != nil {
		return fmt.Errorf("running config wizard: %w", err)
	}

	if err := updatedCfg.Save(); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Println("\n  ✓  Config saved to", config.FilePath())
	fmt.Println("\n  You're ready. Try:")
	fmt.Println("    repomap ./your-project")
	return nil
}
