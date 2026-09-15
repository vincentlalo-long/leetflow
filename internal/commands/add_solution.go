package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func AddSolution(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Add New Solution ---\n")

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}

	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	files := template.GetAllSolutionFiles(cfg.BaseDir, exts)
	if len(files) == 0 {
		ui.WriteOutput(MsgError, "No problem files found in workspace!")
		return fmt.Errorf("no problem files found in workspace")
	}

	fileNames := make([]string, len(files))
	for i, f := range files {
		fileNames[i] = filepath.Base(f)
	}

	selected := ui.PromptSelect("Select problem file to add solution", fileNames)
	if selected == "" {
		return nil
	}

	var filePath string
	for i, n := range fileNames {
		if n == selected {
			filePath = files[i]
			break
		}
	}
	if filePath == "" {
		return nil
	}

	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to read file: %v", err)
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	solNum := template.CountSolutions(content) + 1
	ui.WriteOutput(MsgInfo, "Adding Solution %d to %s", solNum, filepath.Base(filePath))

	method := ui.PromptText("Method / Approach (e.g., Two Pointers)")
	if method == "" {
		method = "Alternative Approach"
	}
	timeComp := ui.PromptText("Time complexity (suggested: O(N))")
	if timeComp == "" {
		timeComp = "O(N)"
	}
	spaceComp := ui.PromptText("Space complexity (suggested: O(1))")
	if spaceComp == "" {
		spaceComp = "O(1)"
	}

	ext := filepath.Ext(filePath)
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		langKey = cfg.DefaultLanguage
	}

	placeholderCode := "// Write or paste your solution code here...\n"
	if langKey == "python" {
		placeholderCode = "# Write or paste your solution code here...\n"
	}

	block := template.BuildSolutionBlock(langKey, solNum, method, timeComp, spaceComp, placeholderCode)

	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to open file: %v", err)
		return fmt.Errorf("failed to open file: %w", err)
	}
	if _, err := f.WriteString(block); err != nil {
		f.Close()
		ui.WriteOutput(MsgError, "Failed to write solution block: %v", err)
		return fmt.Errorf("failed to write solution block: %w", err)
	}
	if err := f.Close(); err != nil {
		ui.WriteOutput(MsgError, "Failed to close file: %v", err)
		return fmt.Errorf("failed to close file: %w", err)
	}

	ui.WriteOutput(MsgSuccess, fmt.Sprintf("Added Solution %d template to %s", solNum, filepath.Base(filePath)))

	editor := cfg.GetEditor()
	if err := openStandardEditor(editor, filePath, ui); err != nil {
		ui.WriteOutput(MsgInfo, "File path: %s", filePath)
		return err
	}

	ui.WriteOutput(MsgSuccess, "Opened file in editor! You can now write or paste your solution code there.")
	return nil
}
