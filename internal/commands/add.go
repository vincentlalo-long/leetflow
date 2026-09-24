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

func AddProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Add New Problem ---\n")

	pos, flags := parseFlags(args)

	problemNum := ""
	if len(pos) > 0 {
		problemNum = pos[0]
	}
	if problemNum == "" && hasFlag(flags, "num") {
		problemNum = flags["num"]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Problem number")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	suggestedName := ""
	slug := ""
	var details *api.ProblemDetail

	ui.WriteOutput(MsgInfo, "Looking up problem details...")
	problemData, err := api.GetProblemByID(problemNum)
	if err == nil && problemData != nil {
		suggestedName = problemData.Title
		slug = problemData.Slug
	}

	if slug == "" && suggestedName != "" {
		slug = api.Slugify(suggestedName)
	}

	if slug != "" {
		details, _ = api.GetProblemDetails(slug)
		if details != nil && details.Title != "" {
			suggestedName = details.Title
		}
	}

	if suggestedName != "" {
		diffStr := ""
		if details != nil && details.Difficulty != "" {
			diffStr = fmt.Sprintf(" (%s)", details.Difficulty)
		}
		ui.WriteOutput(MsgSuccess, "Found: %s%s", suggestedName, diffStr)
	} else {
		ui.WriteOutput(MsgError, "Could not find problem with ID %s", problemNum)
	}

	skipPrompt := hasFlag(flags, "yes") || hasFlag(flags, "y") || (!isTerminalStdin() && isHeadlessUI(ui))

	problemName := ""
	if hasFlag(flags, "name") {
		problemName = flags["name"]
	}
	if problemName == "" && !skipPrompt {
		promptLabel := "Problem name"
		if suggestedName != "" {
			promptLabel = fmt.Sprintf("Problem name (Enter for '%s')", suggestedName)
		}
		entered := ui.PromptText(promptLabel)
		if entered != "" {
			problemName = entered
		}
	}
	if problemName == "" && suggestedName != "" {
		problemName = suggestedName
	}
	if problemName == "" {
		ui.WriteOutput(MsgError, "Problem name cannot be empty")
		return fmt.Errorf("problem name cannot be empty")
	}

	dataStructures := cfg.GetDataStructures()
	detectedCategory := ""
	if details != nil && len(details.TopicTags) > 0 {
		detectedCategory = autoDetectCategory(details.TopicTags, dataStructures)
	}

	selected := ""
	if hasFlag(flags, "ds") {
		selected = flags["ds"]
	}
	if selected == "" && skipPrompt {
		selected = detectedCategory
		if selected == "" {
			selected = "[Uncategorized]"
		}
	}
	if selected == "" {
		var dsChoices []string
		if detectedCategory != "" {
			dsChoices = append(dsChoices, detectedCategory)
		}
		for k := range dataStructures {
			if k != detectedCategory {
				dsChoices = append(dsChoices, k)
			}
		}
		dsChoices = append(dsChoices, "[Uncategorized]", "Add new data structure")

		promptLabel := "Select data structure"
		if detectedCategory != "" {
			promptLabel = fmt.Sprintf("Select data structure (suggested: %s)", detectedCategory)
		}
		selected = ui.PromptSelect(promptLabel, dsChoices)
	}
	if selected == "" {
		selected = detectedCategory
	}
	if selected == "" {
		selected = "[Uncategorized]"
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
		var dsChoices []string
		for k := range dataStructures {
			dsChoices = append(dsChoices, k)
		}
		dsChoices = append(dsChoices, "[Uncategorized]")
		selected = ui.PromptSelect("Select data structure", dsChoices)
		if selected == "" {
			selected = "[Uncategorized]"
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
	if langKey == "" && skipPrompt {
		langKey = cfg.DefaultLanguage
	}
	if langKey == "" {
		langChoices, langMapping := template.GetLanguageChoices(languages, cfg.DefaultLanguage)
		promptLabel := fmt.Sprintf("Select language (default: %s)", cfg.DefaultLanguage)
		langChoice := ui.PromptSelect(promptLabel, langChoices)
		langKey = langMapping[langChoice]
	}
	if langKey == "" {
		langKey = cfg.DefaultLanguage
	}
	if langKey == "" {
		langKey = "cpp"
	}
	langExt := languages[langKey].Ext

	if slug == "" {
		slug = api.Slugify(problemName)
	}
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

	if details == nil && slug != "" {
		ui.WriteOutput(MsgInfo, "Fetching problem details from LeetCode...")
		details, _ = api.GetProblemDetails(slug)
	}

	content := ""
	if details != nil {
		tags := make([]string, len(details.TopicTags))
		for i, t := range details.TopicTags {
			tags[i] = t.Name
		}
		tagsStr := "None"
		if len(tags) > 0 {
			tagsStr = strings.Join(tags, ", ")
		}
		link := fmt.Sprintf("https://leetcode.com/problems/%s/", slug)
		snippet := details.GetCodeSnippet(langKey)
		content = template.BuildProblemTemplateWithSnippet(langKey, problemNum, details.Title, link,
			details.Difficulty, tagsStr, selected, snippet)
	} else {
		content = template.BuildProblemTemplate(langKey, problemNum, problemName, "",
			"Unknown", "None", selected)
		ui.WriteOutput(MsgError, "Could not fetch problem details. Using basic template.")
	}

	err = os.WriteFile(problemFile, []byte(content), 0644)
	if err != nil {
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
		link := fmt.Sprintf("https://leetcode.com/problems/%s/", slug)
		readmeContent := template.MakeReadmeContent(cleanNum, details.Title, link,
			details.Difficulty, strings.Join(func() []string {
				tags := make([]string, len(details.TopicTags))
				for i, t := range details.TopicTags {
					tags[i] = t.Name
				}
				return tags
			}(), ", "), mdContent)
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

// autoDetectCategory matches problem topic tags against configured data structures.
func autoDetectCategory(tags []api.TopicTag, ds map[string]string) string {
	tagMap := map[string]string{
		"array":              "array",
		"hashtable":          "hash",
		"linkedlist":         "linkedlist",
		"string":             "string",
		"tree":               "tree",
		"binarytree":         "tree",
		"binarysearchtree":   "tree",
		"binarysearch":       "binary",
		"dynamicprogramming": "dp",
		"stack":              "stack",
		"queue":              "queue",
		"heap":               "heap",
		"priorityqueue":      "heap",
		"graph":              "graph",
		"twopointers":        "two-pointer",
		"slidingwindow":      "sliding",
		"backtracking":       "backtracking",
		"greedy":             "greedy",
		"math":               "math",
		"bitmanipulation":    "binary",
		"recursion":          "backtracking",
		"trie":               "trie",
		"unionfind":          "graph",
	}

	for _, t := range tags {
		norm := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(t.Name, " ", ""), "-", ""))
		if target, ok := tagMap[norm]; ok {
			if _, exists := ds[target]; exists {
				return target
			}
		}
		for k := range ds {
			kNorm := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(k, " ", ""), "-", ""))
			if kNorm == norm || strings.Contains(norm, kNorm) {
				return k
			}
		}
	}
	return ""
}

