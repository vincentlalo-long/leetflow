package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"leetcli/internal/config"
)

func CleanWorkspace(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Workspace Build Cleanup ---\n")

	baseDir := cfg.BaseDir
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory '%s' does not exist.", baseDir)
		return fmt.Errorf("base directory '%s' does not exist", baseDir)
	}

	ui.WriteOutput(MsgInfo, "Scanning '%s' for temporary build artifacts...", baseDir)

	cleanedCount := 0
	var totalBytes int64 = 0
	failed := 0

	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		base := filepath.Base(path)
		ext := strings.ToLower(filepath.Ext(path))

		isTemp := false
		if strings.HasPrefix(base, "leet_run_") ||
			strings.HasPrefix(base, "leet_harness") ||
			ext == ".class" ||
			ext == ".tmp" ||
			ext == ".o" ||
			base == "a.out" ||
			strings.HasSuffix(base, ".exe~") {
			isTemp = true
		}

		if isTemp {
			size := info.Size()
			if removeErr := os.Remove(path); removeErr == nil {
				cleanedCount++
				totalBytes += size
				ui.WriteOutput(MsgPlain, "  ✓ Removed: %s (%d bytes)", base, size)
			} else {
				failed++
				ui.WriteOutput(MsgError, "  ✘ Could not remove %s: %v", base, removeErr)
			}
		}
		return nil
	})

	if err != nil {
		ui.WriteOutput(MsgError, "Error scanning workspace: %v", err)
		return fmt.Errorf("error scanning workspace: %w", err)
	}

	if cleanedCount == 0 && failed == 0 {
		ui.WriteOutput(MsgSuccess, "Workspace is clean! No build artifacts found.")
		return nil
	}
	if cleanedCount > 0 {
		mb := float64(totalBytes) / (1024 * 1024)
		ui.WriteOutput(MsgSuccess, "Cleaned %d temporary file(s) (%.2f MB reclaimed).", cleanedCount, mb)
	}
	if failed > 0 {
		ui.WriteOutput(MsgError, "%d file(s) could not be removed (locked or in use).", failed)
		return fmt.Errorf("%d file(s) could not be removed", failed)
	}
	return nil
}
