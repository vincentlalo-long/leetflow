package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"leetcli/internal/config"
)

// InitCommand runs the interactive first-run setup wizard to configure
// machine-specific settings (base_dir, editor, terminal, open_mode)
// and saves them safely to config.local.json (git-ignored).
func InitCommand(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "============================================")
	ui.WriteOutput(MsgPlain, "       Leet CLI - First-Run Setup Wizard    ")
	ui.WriteOutput(MsgPlain, "============================================\n")
	ui.WriteOutput(MsgInfo, "This wizard sets up your workspace without touching git-tracked files.")
	ui.WriteOutput(MsgInfo, "Settings will be saved to config.local.json (machine-specific).\n")

	_, flags := parseFlags(args)

	// 1. Detect & Prompt Workspace Directory (base_dir)
	defaultBaseDir := "~/leetcode"
	if runtime.GOOS == "windows" {
		if _, err := os.Stat("D:\\"); err == nil {
			defaultBaseDir = "D:\\leetcode"
		} else {
			defaultBaseDir = "C:\\leetcode"
		}
	}
	if cfg.BaseDir != "" && cfg.BaseDir != "." {
		home, _ := os.UserHomeDir()
		if home == "" || !strings.EqualFold(filepath.Clean(config.ExpandHome(cfg.BaseDir)), filepath.Clean(home)) {
			defaultBaseDir = cfg.BaseDir
		}
	}

	baseDir := flags["dir"]
	if baseDir == "" {
		if ui.PromptText("Workspace directory (where solutions will be stored)") != "" {
			// Note: PromptText in HeadlessUI can return empty; we handle both
		}
	}

	// If interactive PromptText is available:
	promptedDir := ui.PromptText(fmt.Sprintf("Enter workspace directory [%s]", defaultBaseDir))
	if strings.TrimSpace(promptedDir) != "" {
		baseDir = strings.TrimSpace(promptedDir)
	} else if baseDir == "" {
		baseDir = defaultBaseDir
	}

	expandedDir := config.ExpandHome(baseDir)
	if err := os.MkdirAll(expandedDir, 0755); err != nil {
		ui.WriteOutput(MsgError, "Failed to create directory %s: %v", expandedDir, err)
		return fmt.Errorf("failed to create base directory: %w", err)
	}
	cfg.BaseDir = baseDir
	ui.WriteOutput(MsgSuccess, "Workspace directory: %s", expandedDir)

	// 2. Default Programming Language
	defaultLang := cfg.DefaultLanguage
	if defaultLang == "" {
		defaultLang = "cpp"
	}
	if langFlag := flags["lang"]; langFlag != "" {
		cfg.DefaultLanguage = langFlag
	} else {
		langChoices := []string{
			"cpp (C++)",
			"python (Python)",
			"go (Go)",
			"java (Java)",
			"rust (Rust)",
			"javascript (JavaScript)",
			"typescript (TypeScript)",
			"c (C)",
		}
		selectedLangChoice := ui.PromptSelect("Select default programming language", langChoices)
		if selectedLangChoice != "" {
			parts := strings.Fields(selectedLangChoice)
			if len(parts) > 0 {
				cfg.DefaultLanguage = parts[0]
			}
		}
	}
	ui.WriteOutput(MsgSuccess, "Default language: %s", cfg.DefaultLanguage)

	// 3. Preferred Code Editor
	detectedEditors := detectAvailableTools([]string{"nvim", "vim", "code", "helix", "hx", "nano", "subl"})
	defaultEditor := "nvim"
	if runtime.GOOS == "windows" {
		defaultEditor = "code"
	}
	if len(detectedEditors) > 0 {
		defaultEditor = detectedEditors[0]
	}

	if edFlag := flags["editor"]; edFlag != "" {
		cfg.Editor = edFlag
	} else {
		var editorChoices []string
		for _, ed := range detectedEditors {
			editorChoices = append(editorChoices, ed)
		}
		if len(editorChoices) == 0 {
			editorChoices = []string{"nvim", "vim", "code", "nano"}
		}
		editorChoices = append(editorChoices, "other (type custom command)")

		selectedEditor := ui.PromptSelect(fmt.Sprintf("Select code editor [detected: %s]", defaultEditor), editorChoices)
		if selectedEditor == "other (type custom command)" {
			customEditor := ui.PromptText("Enter editor command")
			if strings.TrimSpace(customEditor) != "" {
				cfg.Editor = strings.TrimSpace(customEditor)
			} else {
				cfg.Editor = defaultEditor
			}
		} else if selectedEditor != "" {
			cfg.Editor = selectedEditor
		} else {
			cfg.Editor = defaultEditor
		}
	}
	ui.WriteOutput(MsgSuccess, "Code editor: %s", cfg.Editor)

	// 4. Terminal Emulator (primarily on Linux)
	if runtime.GOOS != "windows" {
		detectedTerms := detectAvailableTools([]string{"kitty", "alacritty", "ghostty", "wezterm", "foot"})
		defaultTerm := "kitty"
		if len(detectedTerms) > 0 {
			defaultTerm = detectedTerms[0]
		}
		if termFlag := flags["terminal"]; termFlag != "" {
			cfg.Terminal = termFlag
		} else {
			var termChoices []string
			for _, t := range detectedTerms {
				termChoices = append(termChoices, t)
			}
			if len(termChoices) == 0 {
				termChoices = []string{"kitty", "alacritty", "ghostty", "wezterm", "xterm"}
			}
			termChoices = append(termChoices, "other (custom)")

			selectedTerm := ui.PromptSelect(fmt.Sprintf("Select terminal emulator [detected: %s]", defaultTerm), termChoices)
			if selectedTerm == "other (custom)" {
				customTerm := ui.PromptText("Enter terminal command")
				if strings.TrimSpace(customTerm) != "" {
					cfg.Terminal = strings.TrimSpace(customTerm)
				} else {
					cfg.Terminal = defaultTerm
				}
			} else if selectedTerm != "" {
				cfg.Terminal = selectedTerm
			} else {
				cfg.Terminal = defaultTerm
			}
		}
		ui.WriteOutput(MsgSuccess, "Terminal emulator: %s", cfg.Terminal)

		// 5. Open Workspace Mode
		if modeFlag := flags["open-mode"]; modeFlag != "" {
			cfg.OpenMode = modeFlag
		} else {
			modeChoices := []string{
				"auto (smart detection: Kitty splits / Hyprland tiling / Tmux)",
				"kitty (split-pane inside single Kitty window)",
				"wm (open separate windows for tiling window manager)",
				"tmux (split panes inside tmux session)",
				"simple (just open editor without tiling layout)",
			}
			selectedMode := ui.PromptSelect("Select workspace layout mode for 'leet open'", modeChoices)
			if selectedMode != "" {
				fields := strings.Fields(selectedMode)
				if len(fields) > 0 {
					cfg.OpenMode = fields[0]
				}
			} else {
				cfg.OpenMode = "auto"
			}
		}
		ui.WriteOutput(MsgSuccess, "Workspace open mode: %s", cfg.OpenMode)
	} else {
		// Windows default
		cfg.OpenMode = "simple"
	}

	// Save to config.local.json
	if err := cfg.Save(); err != nil {
		ui.WriteOutput(MsgError, "Failed to save configuration: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}

	ui.WriteOutput(MsgPlain, "")
	ui.WriteOutput(MsgSuccess, "Configuration successfully saved to %s", cfg.GetPath())
	ui.WriteOutput(MsgPlain, "\nReady to go! Try these commands:")
	ui.WriteOutput(MsgInfo, "  leet add 1              # Fetch Two Sum and create workspace")
	ui.WriteOutput(MsgInfo, "  leet open 1             # Open solution with workspace layout")
	ui.WriteOutput(MsgInfo, "  leet test 1 --local     # Run test cases offline")
	ui.WriteOutput(MsgInfo, "  leet doctor             # Verify workspace health")
	ui.WriteOutput(MsgPlain, "")

	return nil
}

// detectAvailableTools checks which binaries from the list exist in PATH.
func detectAvailableTools(tools []string) []string {
	var found []string
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err == nil {
			found = append(found, tool)
		}
	}
	return found
}
