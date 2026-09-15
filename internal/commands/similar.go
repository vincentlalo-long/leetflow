package commands

import (
	"encoding/json"
	"fmt"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

type SimilarProblem struct {
	Title       string `json:"title"`
	TitleSlug   string `json:"titleSlug"`
	Difficulty  string `json:"difficulty"`
}

func Similar(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Find Similar Problems ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	ui.WriteOutput(MsgInfo, "Looking up problem...")
	problemData, err := api.GetProblemByID(problemNum)
	if err != nil {
		ui.WriteOutput(MsgError, "Could not find problem with ID %s", problemNum)
		return fmt.Errorf("could not find problem with ID %s", problemNum)
	}

	ui.WriteOutput(MsgSuccess, "Found: %s", problemData.Title)
	ui.WriteOutput(MsgInfo, "Fetching similar problems...")

	details, err := api.GetProblemDetails(problemData.Slug)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to fetch problem details from LeetCode.")
		return fmt.Errorf("failed to fetch problem details: %w", err)
	}

	if details.SimilarQuestions == "" {
		ui.WriteOutput(MsgInfo, "No similar problems available for this problem.")
		return nil
	}

	var similar []SimilarProblem
	if err := json.Unmarshal([]byte(details.SimilarQuestions), &similar); err != nil {
		ui.WriteOutput(MsgError, "Failed to parse similar problems data.")
		return fmt.Errorf("failed to parse similar problems data: %w", err)
	}

	if len(similar) == 0 {
		ui.WriteOutput(MsgInfo, "No similar problems found for this problem.")
		return nil
	}

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}
	solvedMap := template.GetSolvedMap(cfg.BaseDir, exts)

	ui.WriteOutput(MsgPlain, "\nSimilar Problems to %s (%d total):", problemData.Title, len(similar))

	for i, sp := range similar {
		difficulty := sp.Difficulty
		badge := "[○ Unsolved]"
		if solvedMap[sp.TitleSlug] || solvedMap[api.Slugify(sp.Title)] {
			badge = "[✔ Solved]"
		}
		ui.WriteOutput(MsgPlain, "  %d. %-35s (%s) %s", i+1, sp.Title, difficulty, badge)
	}

	ui.WriteOutput(MsgSuccess, "Finished showing similar problems.")
	return nil
}
