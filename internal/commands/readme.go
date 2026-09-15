package commands

import (
	"fmt"
	"os"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func BuildReadme(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Generating Root Workspace README ---\n")

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Base directory not configured.")
		return fmt.Errorf("base directory not configured")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory '%s' does not exist.", baseDir)
		return fmt.Errorf("base directory '%s' does not exist", baseDir)
	}

	ui.WriteOutput(MsgInfo, "Scanning workspace for problems...")
	readmePath, count, err := template.GenerateRootReadme(baseDir, cfg)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to generate README: %v", err)
		return fmt.Errorf("failed to generate README: %w", err)
	}

	ui.WriteOutput(MsgSuccess, "Successfully generated root README.md with %d problem(s)!", count)
	ui.WriteOutput(MsgInfo, "File saved to: %s", readmePath)
	return nil
}
