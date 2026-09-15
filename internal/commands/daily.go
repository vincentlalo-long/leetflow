package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

func Daily(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- LeetCode Daily Challenge ---\n")

	_, flags := parseFlags(args)
	autoAdd := hasFlag(flags, "add") || hasFlag(flags, "yes") || hasFlag(flags, "y")

	ui.WriteOutput(MsgInfo, "Fetching daily challenge...")
	detail, _, _, err := api.GetDailyChallenge()
	if err != nil {
		ui.WriteOutput(MsgError, "Could not fetch daily challenge: %v", err)
		return fmt.Errorf("could not fetch daily challenge: %w", err)
	}
	if detail == nil {
		ui.WriteOutput(MsgError, "Could not fetch daily challenge. Please check your connection.")
		return fmt.Errorf("could not fetch daily challenge")
	}

	problemNum := detail.QuestionFrontendID
	problemName := detail.Title
	slug := api.Slugify(problemName)
	difficulty := detail.Difficulty
	tags := make([]string, len(detail.TopicTags))
	for i, t := range detail.TopicTags {
		tags[i] = t.Name
	}
	tagsStr := "None"
	if len(tags) > 0 {
		tagsStr = strings.Join(tags, ", ")
	}
	leetLink := fmt.Sprintf("https://leetcode.com/problems/%s/", slug)

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}
	solvedMap := template.GetSolvedMap(cfg.BaseDir, exts)
	statusBadge := "[○ Unsolved]"
	if solvedMap[problemNum] || solvedMap[slug] {
		statusBadge = "[✔ Solved]"
	}

	ui.WriteOutput(MsgPlain, "\nToday's Problem: %s", statusBadge)
	ui.WriteOutput(MsgPlain, "  %s. %s", problemNum, problemName)
	ui.WriteOutput(MsgInfo, "Difficulty: %s", difficulty)
	ui.WriteOutput(MsgInfo, "Tags: %s", tagsStr)
	ui.WriteOutput(MsgInfo, "Link: %s", leetLink)

	addIt := autoAdd
	if !addIt {
		addIt = ui.PromptConfirm("Add this problem to your workspace?")
	}
	if !addIt {
		return nil
	}

	dataStructures := cfg.GetDataStructures()

	dsChoices := []string{"[Uncategorized]"}
	for k := range dataStructures {
		dsChoices = append(dsChoices, k)
	}
	dsChoices = append(dsChoices, "Add new data structure")

	selected := ""
	if hasFlag(flags, "ds") {
		selected = flags["ds"]
	}
	if selected == "" {
		selected = ui.PromptSelect("Select data structure", dsChoices)
	}
	if selected == "" {
		return nil
	}
	if selected == "Add new data structure" {
		name := ui.PromptText("Data structure name (e.g., tree)")
		folder := name
		if folder == "" {
			return nil
		}
		folderInput := ui.PromptText(fmt.Sprintf("Folder name (press Enter for '%s')", name))
		if folderInput != "" {
			folder = folderInput
		}
		if cfg.AddDataStructure(name, folder) {
			if err := cfg.Save(); err != nil {
				ui.WriteOutput(MsgError, "Failed to save config: %v", err)
				return fmt.Errorf("failed to save config: %w", err)
			}
			ui.WriteOutput(MsgSuccess, "Added data structure: %s -> %s", name, folder)
		}
		dataStructures = cfg.GetDataStructures()
		dsChoices = []string{"[Uncategorized]"}
		for k := range dataStructures {
			dsChoices = append(dsChoices, k)
		}
		selected = ui.PromptSelect("Select data structure", dsChoices)
		if selected == "" {
			return nil
		}
	}

	dsFolder := "uncategorized"
	if selected != "[Uncategorized]" {
		if f, ok := dataStructures[selected]; ok {
			dsFolder = f
		}
	}

	languages = template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	langKey := ""
	if hasFlag(flags, "lang") {
		langKey = resolveLangFlag(languages, flags["lang"])
	}
	if langKey == "" {
		langChoices, langMapping := template.GetLanguageChoices(languages, cfg.DefaultLanguage)
		langChoice := ui.PromptSelect("Select language", langChoices)
		langKey = langMapping[langChoice]
	}
	if langKey == "" {
		langKey = cfg.DefaultLanguage
	}
	langExt := languages[langKey].Ext

	folderName := fmt.Sprintf("%s-%s", problemNum, slug)
	problemDir, err := template.CreateProblemDirectory(cfg.BaseDir, dsFolder, folderName)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to create directory: %v", err)
		return fmt.Errorf("failed to create directory: %w", err)
	}

	safeName := template.SanitizeFileName(problemName)
	if safeName == "" {
		safeName = slug
	}
	problemFile := filepath.Join(problemDir, fmt.Sprintf("%s_%s.%s", problemNum, safeName, langExt))
	if _, err := os.Stat(problemFile); err == nil {
		ui.WriteOutput(MsgError, "Problem file already exists!")
		return fmt.Errorf("problem file already exists")
	}

	content := template.BuildProblemTemplate(langKey, problemNum, problemName, leetLink,
		difficulty, tagsStr, selected)
	if err := os.WriteFile(problemFile, []byte(content), 0644); err != nil {
		ui.WriteOutput(MsgError, "Failed to write file: %v", err)
		return fmt.Errorf("failed to write file: %w", err)
	}

	details, err := api.GetProblemDetails(slug)
	if err != nil {
		ui.WriteOutput(MsgInfo, "Could not fetch problem description (using basic template).")
	} else if details != nil && details.Content != "" {
		readmePath := filepath.Join(problemDir, "README.md")
		mdContent := template.FormatDescriptionMarkdown(details.Content)
		cleanNum := strings.TrimLeft(problemNum, "0")
		if cleanNum == "" {
			cleanNum = problemNum
		}
		readmeContent := template.MakeReadmeContent(cleanNum, problemName, leetLink,
			difficulty, tagsStr, mdContent)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			ui.WriteOutput(MsgError, "Failed to write README.md: %v", err)
			return fmt.Errorf("failed to write README.md: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Saved problem description to README.md")
	}

	ui.WriteOutput(MsgSuccess, "Created problem directory and files")
	ui.WriteOutput(MsgInfo, "Path: %s", problemFile)
	return nil
}
