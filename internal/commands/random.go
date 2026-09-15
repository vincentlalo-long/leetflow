package commands

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strings"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

func Random(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Random LeetCode Problem ---\n")

	_, flags := parseFlags(args)
	autoAdd := hasFlag(flags, "add") || hasFlag(flags, "yes") || hasFlag(flags, "y")

	difficultyChoices := []string{"Any", "Easy", "Medium", "Hard"}
	selectedDifficulty := ""
	if hasFlag(flags, "difficulty") || hasFlag(flags, "diff") {
		d := flags["difficulty"]
		if d == "" {
			d = flags["diff"]
		}
		selectedDifficulty = strings.Title(strings.ToLower(d))
		if selectedDifficulty != "Any" && !isDifficulty(selectedDifficulty) {
			ui.WriteOutput(MsgError, "Invalid difficulty: %s", selectedDifficulty)
			return fmt.Errorf("invalid difficulty: %s", selectedDifficulty)
		}
	}
	if selectedDifficulty == "" {
		selectedDifficulty = ui.PromptSelect("Select difficulty level", difficultyChoices)
	}
	if selectedDifficulty == "" {
		return nil
	}

	ui.WriteOutput(MsgInfo, "Fetching problems...")
	data, err := api.GetAllProblems()
	if err != nil {
		ui.WriteOutput(MsgError, "Could not fetch problems. Please check your connection.")
		return fmt.Errorf("could not fetch problems: %w", err)
	}

	type candidate struct {
		FrontendID int
		Title      string
		Slug       string
		Level      int
		PaidOnly   bool
	}
	var candidates []candidate

	diffLevel := 0
	if selectedDifficulty != "Any" {
		diffLevel = template.DiffToLevel[selectedDifficulty]
	}

	for _, p := range data.StatStatusPairs {
		if p.PaidOnly {
			continue
		}
		if diffLevel > 0 && p.Difficulty.Level != diffLevel {
			continue
		}
		candidates = append(candidates, candidate{
			FrontendID: p.Stat.FrontendQuestionID,
			Title:      p.Stat.QuestionTitle,
			Slug:       p.Stat.QuestionTitleSlug,
			Level:      p.Difficulty.Level,
		})
	}

	if len(candidates) == 0 {
		ui.WriteOutput(MsgError, "No free %s problems found.", selectedDifficulty)
		return fmt.Errorf("no free %s problems found", selectedDifficulty)
	}

	prob := candidates[rand.Intn(len(candidates))]
	problemNum := fmt.Sprintf("%d", prob.FrontendID)
	problemName := prob.Title
	slug := prob.Slug
	difficulty := template.LevelToName[prob.Level]
	link := fmt.Sprintf("https://leetcode.com/problems/%s/", slug)

	ui.WriteOutput(MsgPlain, "\nYour Random Problem:")
	ui.WriteOutput(MsgPlain, "  %s. %s", problemNum, problemName)
	ui.WriteOutput(MsgInfo, "Difficulty: %s", difficulty)
	ui.WriteOutput(MsgInfo, "Link: %s", link)

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

	languages := template.NormalizeLanguages(nil)
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

	ui.WriteOutput(MsgInfo, "Fetching full problem details...")
	details, err := api.GetProblemDetails(slug)
	if err != nil {
		ui.WriteOutput(MsgInfo, "Could not fetch problem description (using basic template).")
	}

	tagsStr := "None"
	if details != nil {
		tags := make([]string, len(details.TopicTags))
		for i, t := range details.TopicTags {
			tags[i] = t.Name
		}
		if len(tags) > 0 {
			tagsStr = strings.Join(tags, ", ")
		}
	}

	content := template.BuildProblemTemplate(langKey, problemNum, problemName, link,
		difficulty, tagsStr, selected)
	if err := os.WriteFile(problemFile, []byte(content), 0644); err != nil {
		ui.WriteOutput(MsgError, "Failed to write file: %v", err)
		return fmt.Errorf("failed to write file: %w", err)
	}

	if details != nil && details.Content != "" {
		readmePath := filepath.Join(problemDir, "README.md")
		mdContent := template.FormatDescriptionMarkdown(details.Content)
		cleanNum := strings.TrimLeft(problemNum, "0")
		if cleanNum == "" {
			cleanNum = problemNum
		}
		readmeContent := template.MakeReadmeContent(cleanNum, problemName, link,
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
