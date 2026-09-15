package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func AddNote(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Add Study Note / Takeaway ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	} else {
		problemNum = ui.PromptText("Enter problem number")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	baseDir := cfg.BaseDir
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
		ui.WriteOutput(MsgError, "Could not find problem file for %s in workspace.", problemNum)
		return fmt.Errorf("could not find problem file for %s in workspace", problemNum)
	}

	targetFile := matches[0]
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, f := range matches {
			names[i] = filepath.Base(f)
		}
		selected := ui.PromptSelect("Multiple matches. Select problem file", names)
		for i, n := range names {
			if n == selected {
				targetFile = matches[i]
				break
			}
		}
	}

	noteText := ui.PromptText("Enter your key note / takeaway for this problem")
	if strings.TrimSpace(noteText) == "" {
		ui.WriteOutput(MsgInfo, "No note entered. Aborted.")
		return nil
	}

	targetDir := filepath.Dir(targetFile)
	readmePath := filepath.Join(targetDir, "README.md")

	dateStr := time.Now().Format("2006-01-02")
	formattedNote := fmt.Sprintf("\n\n## 📝 Notes & Takeaways\n- **[%s]**: %s\n", dateStr, strings.TrimSpace(noteText))

	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		title := filepath.Base(targetDir)
		initContent := fmt.Sprintf("# %s\n", title) + formattedNote
		if err := os.WriteFile(readmePath, []byte(initContent), 0644); err != nil {
			ui.WriteOutput(MsgError, "Failed to create README.md: %v", err)
			return fmt.Errorf("failed to create README.md: %w", err)
		}
	} else {
		f, err := os.OpenFile(readmePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			ui.WriteOutput(MsgError, "Failed to open README.md: %v", err)
			return fmt.Errorf("failed to open README.md: %w", err)
		}
		if _, err := f.WriteString(formattedNote); err != nil {
			f.Close()
			ui.WriteOutput(MsgError, "Failed to write note to README.md: %v", err)
			return fmt.Errorf("failed to write note to README.md: %w", err)
		}
		if err := f.Close(); err != nil {
			ui.WriteOutput(MsgError, "Failed to close README.md: %v", err)
			return fmt.Errorf("failed to close README.md: %w", err)
		}
	}

	ui.WriteOutput(MsgSuccess, "Saved note to %s!", filepath.Base(readmePath))
	ui.WriteOutput(MsgInfo, "Note: %s", noteText)
	return nil
}
