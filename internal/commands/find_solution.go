package commands

import (
	"os"
	"path/filepath"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

// resolveLocalSolution finds the local solution file matching a problem number
// and returns its path and content. Returns ("", "") when nothing is found.
func resolveLocalSolution(baseDir, problemNum string, cfg *config.Config, ui UI) (string, string) {
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist: %s", baseDir)
		return "", ""
	}

	ui.WriteOutput(MsgInfo, "Searching for problem...")
	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	allFiles := template.GetAllSolutionFiles(baseDir, exts)
	var matches []string
	for _, f := range allFiles {
		base := filepath.Base(f)
		if template.MatchesProblemNumber(base, problemNum) {
			matches = append(matches, f)
		}
	}

	if len(matches) == 0 {
		ui.WriteOutput(MsgError, "Could not find local file for problem %s.", problemNum)
		return "", ""
	}

	targetFile := matches[0]
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, f := range matches {
			names[i] = filepath.Base(f)
		}
		selected := ui.PromptSelect("Multiple matches. Select file to use", names)
		for i, n := range names {
			if n == selected {
				targetFile = matches[i]
				break
			}
		}
	}

	contentBytes, err := os.ReadFile(targetFile)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to read file: %v", err)
		return "", ""
	}
	return targetFile, string(contentBytes)
}