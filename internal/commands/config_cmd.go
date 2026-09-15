package commands

import (
	"fmt"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func ManageConfig(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- CLI Configuration Manager ---\n")

	if len(args) > 0 {
		switch args[0] {
		case "show":
			showConfig(cfg, ui)
			return nil
		case "open":
			if err := openConfig(cfg, ui); err != nil {
				return err
			}
			return nil
		}
	}

	choices := []string{
		fmt.Sprintf("1. Base Workspace Directory (base_dir: %s)", cfg.BaseDir),
		fmt.Sprintf("2. Default Language (default_language: %s)", cfg.DefaultLanguage),
		fmt.Sprintf("3. Code / Text Editor (editor: %s)", cfg.GetEditor()),
		fmt.Sprintf("4. LeetCode Session Cookie (leetcode_session: %s)", maskSecret(cfg.LeetcodeSession)),
		fmt.Sprintf("5. LeetCode CSRF Token (leetcode_csrf: %s)", maskSecret(cfg.LeetcodeCsrf)),
		"6. Open config.json in Editor",
		"7. Show All Current Config Settings",
	}

	selected := ui.PromptSelect("Select config option to edit (or Esc/Ctrl+C to cancel):", choices)
	if selected == "" {
		return nil
	}

	switch {
	case strings.HasPrefix(selected, "1."):
		newDir := ui.PromptText(fmt.Sprintf("Enter new base directory (suggested: %s)", cfg.BaseDir))
		if newDir != "" {
			cfg.BaseDir = newDir
			if err := cfg.Save(); err != nil {
				ui.WriteOutput(MsgError, "Failed to save config: %v", err)
				return fmt.Errorf("failed to save config: %w", err)
			}
			ui.WriteOutput(MsgSuccess, "Updated base_dir to: %s", cfg.BaseDir)
		}

	case strings.HasPrefix(selected, "2."):
		languages := template.NormalizeLanguages(nil)
		for k, v := range cfg.Languages {
			languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
		}
		langChoices, langMapping := template.GetLanguageChoices(languages, cfg.DefaultLanguage)
		langChoice := ui.PromptSelect("Select default language", langChoices)
		if langChoice != "" {
			langKey := langMapping[langChoice]
			cfg.DefaultLanguage = langKey
			if err := cfg.Save(); err != nil {
				ui.WriteOutput(MsgError, "Failed to save config: %v", err)
				return fmt.Errorf("failed to save config: %w", err)
			}
			ui.WriteOutput(MsgSuccess, "Updated default_language to: %s", cfg.DefaultLanguage)
		}

	case strings.HasPrefix(selected, "3."):
		editorChoices := []string{
			"code (VS Code)",
			"vim",
			"nvim",
			"nano",
			"notepad",
			"Custom editor",
		}
		choice := ui.PromptSelect("Select code editor", editorChoices)
		if choice != "" {
			newEditor := "code"
			if strings.HasPrefix(choice, "Custom") {
				newEditor = ui.PromptText("Enter custom editor executable name")
			} else {
				parts := strings.Fields(choice)
				newEditor = parts[0]
			}
			if newEditor != "" {
				cfg.SetEditor(newEditor)
				if err := cfg.Save(); err != nil {
					ui.WriteOutput(MsgError, "Failed to save config: %v", err)
					return fmt.Errorf("failed to save config: %w", err)
				}
				ui.WriteOutput(MsgSuccess, "Updated editor to: %s", cfg.GetEditor())
			}
		}

	case strings.HasPrefix(selected, "4."):
		ui.WriteOutput(MsgInfo, "Tip: you can also use the LEETCODE_SESSION env var instead of storing the cookie.")
		session := ui.PromptText("Enter LEETCODE_SESSION cookie value")
		if session != "" {
			cfg.LeetcodeSession = strings.TrimSpace(session)
			if err := cfg.Save(); err != nil {
				ui.WriteOutput(MsgError, "Failed to save config: %v", err)
				return fmt.Errorf("failed to save config: %w", err)
			}
			ui.WriteOutput(MsgSuccess, "Updated leetcode_session cookie (stored in %s).", cfg.GetPath())
		}

	case strings.HasPrefix(selected, "5."):
		ui.WriteOutput(MsgInfo, "Tip: you can also use the LEETCODE_CSRF env var instead of storing the token.")
		csrf := ui.PromptText("Enter csrftoken cookie value")
		if csrf != "" {
			cfg.LeetcodeCsrf = strings.TrimSpace(csrf)
			if err := cfg.Save(); err != nil {
				ui.WriteOutput(MsgError, "Failed to save config: %v", err)
				return fmt.Errorf("failed to save config: %w", err)
			}
			ui.WriteOutput(MsgSuccess, "Updated leetcode_csrf token (stored in %s).", cfg.GetPath())
		}

	case strings.HasPrefix(selected, "6."):
		if err := openConfig(cfg, ui); err != nil {
			return err
		}

	case strings.HasPrefix(selected, "7."):
		showConfig(cfg, ui)
	}
	return nil
}

func showConfig(cfg *config.Config, ui UI) {
	ui.WriteOutput(MsgPlain, "=== Current Configuration ===")
	ui.WriteOutput(MsgPlain, "  Config File:      %s", cfg.GetPath())
	ui.WriteOutput(MsgPlain, "  Base Directory:   %s", cfg.BaseDir)
	ui.WriteOutput(MsgPlain, "  Default Language: %s", cfg.DefaultLanguage)
	ui.WriteOutput(MsgPlain, "  Code Editor:      %s", cfg.GetEditor())
	ui.WriteOutput(MsgPlain, "  Theme:            %s", cfg.GetTheme())
	ui.WriteOutput(MsgPlain, "  LeetCode Session: %s", maskSecret(cfg.LeetcodeSession))
	ui.WriteOutput(MsgPlain, "  LeetCode CSRF:    %s", maskSecret(cfg.LeetcodeCsrf))
	ui.WriteOutput(MsgPlain, "=============================\n")
}

func openConfig(cfg *config.Config, ui UI) error {
	editor := cfg.GetEditor()
	configPath := cfg.GetPath()
	if err := openStandardEditor(editor, configPath, ui); err != nil {
		ui.WriteOutput(MsgInfo, "Config file location: %s", configPath)
		return err
	}
	ui.WriteOutput(MsgSuccess, "Config file opened in editor!")
	return nil
}

func maskSecret(s string) string {
	if s == "" {
		return "(not set)"
	}
	if len(s) <= 8 {
		return "********"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
