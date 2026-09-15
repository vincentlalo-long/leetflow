package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

func OpenWebProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Open LeetCode in Browser ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	} else {
		problemNum = ui.PromptText("Enter problem number (or title slug)")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	slug := api.Slugify(problemNum)

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	allFiles := template.GetAllSolutionFiles(cfg.BaseDir, exts)
	foundLocal := false
	for _, f := range allFiles {
		base := filepath.Base(f)
		if template.MatchesProblemNumber(base, problemNum) {
			content, err := os.ReadFile(f)
			if err == nil {
				inferred := template.InferSlugFromPath(f, string(content))
				if inferred != "" {
					slug = inferred
					foundLocal = true
					break
				}
			}
		}
	}

	problemData, err := api.GetProblemByID(problemNum)
	if err == nil && problemData != nil {
		slug = problemData.Slug
	} else if !foundLocal && isNumeric(problemNum) {
		ui.WriteOutput(MsgError, "Could not resolve problem '%s' from local files or the LeetCode API.", problemNum)
		return fmt.Errorf("could not resolve problem '%s'", problemNum)
	}

	url := fmt.Sprintf("https://leetcode.com/problems/%s/", slug)
	ui.WriteOutput(MsgInfo, "Opening in default browser: %s", url)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// rundll32 passes the URL directly to the Win32 API without shell
		// re-parsing, avoiding cmd metacharacter injection from user input.
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}

	if err := cmd.Start(); err != nil {
		ui.WriteOutput(MsgError, "Failed to open browser: %v", err)
		ui.WriteOutput(MsgInfo, "URL: %s", url)
		return fmt.Errorf("failed to open browser: %w", err)
	}

	ui.WriteOutput(MsgSuccess, "Opened problem page in browser!")
	return nil
}

var numericRe = regexp.MustCompile(`^\d+$`)

func isNumeric(s string) bool {
	return numericRe.MatchString(s)
}
