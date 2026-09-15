package commands

import (
	"fmt"
	"os"

	"leetcli/internal/config"
	"leetcli/internal/template"
	"leetcli/internal/tracker"
)

func Stats(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Problem Statistics ---\n")

	baseDir := cfg.BaseDir
	dataStructures := cfg.GetDataStructures()

	if baseDir == "" {
		ui.WriteOutput(MsgError, "Missing 'base_dir' in config")
		return fmt.Errorf("missing 'base_dir' in config")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist: %s", baseDir)
		return fmt.Errorf("base directory does not exist: %s", baseDir)
	}

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	files := template.GetAllSolutionFiles(baseDir, exts)
	totalProblems := len(files)
	totalSolutions := 0

	byStructure := make(map[string]int)
	for name := range dataStructures {
		byStructure[name] = 0
	}
	unmatched := 0

	for _, f := range files {
		structName := template.DetectStructure(f, dataStructures)
		if structName != "unmatched" {
			byStructure[structName]++
		} else {
			unmatched++
		}

		if content, err := os.ReadFile(f); err == nil {
			totalSolutions += template.CountSolutions(string(content))
		}
	}

	ui.WriteOutput(MsgPlain, "\nOverview:")
	ui.WriteOutput(MsgInfo, "Base directory: %s", baseDir)
	ui.WriteOutput(MsgInfo, "Total problems: %d", totalProblems)
	ui.WriteOutput(MsgInfo, "Total solutions: %d", totalSolutions)

	ui.WriteOutput(MsgPlain, "\nBy Data Structure:")
	if len(dataStructures) == 0 {
		ui.WriteOutput(MsgInfo, "No data structures configured")
	} else {
		for _, name := range sortedKeys(byStructure) {
			ui.WriteOutput(MsgPlain, "  %-20s %d", name, byStructure[name])
		}
	}

	if unmatched > 0 {
		ui.WriteOutput(MsgInfo, "Unmatched problems: %d", unmatched)
	}

	prog := tracker.Load(baseDir)
	if len(prog.Problems) > 0 {
		solved := 0
		byDiff := map[string]int{}
		due := prog.DueReviews()
		for _, e := range prog.Problems {
			if e.Status == "solved" {
				solved++
				byDiff[e.Difficulty]++
			}
		}
		ui.WriteOutput(MsgPlain, "\nProgress Tracker (.leet/progress.json):")
		ui.WriteOutput(MsgInfo, "Tracked problems: %d", len(prog.Problems))
		ui.WriteOutput(MsgInfo, "Solved (accepted): %d", solved)
		if len(byDiff) > 0 {
			ui.WriteOutput(MsgInfo, "  Easy: %d | Medium: %d | Hard: %d", byDiff["Easy"], byDiff["Medium"], byDiff["Hard"])
		}
		ui.WriteOutput(MsgInfo, "Due for review: %d", len(due))
	} else {
		ui.WriteOutput(MsgInfo, "No progress tracked yet. Submit solutions or use 'review --solve <num>'.")
	}
	return nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}
