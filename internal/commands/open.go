package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func OpenProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Open Problem ---\n")

	positional, flags := parseFlags(args)

	problemNum := ""
	if len(positional) > 0 {
		problemNum = positional[0]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number to open")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Invalid base directory in config")
		ui.WriteOutput(MsgInfo, "Run 'leet init' to configure your workspace.")
		return fmt.Errorf("invalid base directory in config")
	}
	baseDir = config.ExpandHome(baseDir)
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist: %s", baseDir)
		return fmt.Errorf("base directory does not exist: %s", baseDir)
	}

	ui.WriteOutput(MsgInfo, "Searching for problem...")
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
		ui.WriteOutput(MsgError, "Could not find local file for problem %s.", problemNum)
		ui.WriteOutput(MsgInfo, "Try running 'leet add %s' first.", problemNum)
		return fmt.Errorf("could not find local file for problem %s", problemNum)
	}

	targetFile := matches[0]
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, f := range matches {
			names[i] = filepath.Base(f)
		}
		selected := ui.PromptSelect("Multiple matches. Select file to open", names)
		for i, n := range names {
			if n == selected {
				targetFile = matches[i]
				break
			}
		}
	}

	targetDir := filepath.Dir(targetFile)
	ui.WriteOutput(MsgSuccess, "Found problem: %s", filepath.Base(targetFile))

	readmePath := filepath.Join(targetDir, "README.md")
	hasReadme := false
	if _, err := os.Stat(readmePath); err == nil {
		hasReadme = true
	}

	editor := cfg.GetEditor()
	openMode := cfg.GetOpenMode()
	if modeFlag := flags["mode"]; modeFlag != "" {
		openMode = modeFlag
	}
	if hasFlag(flags, "simple") {
		openMode = "simple"
	}

	// On Linux / macOS, support workspace split layout (Hyprland tiling / Tmux / Kitty splits)
	if runtime.GOOS != "windows" && openMode != "simple" {
		launched, err := openWorkspaceLayout(cfg, targetFile, targetDir, readmePath, hasReadme, problemNum, openMode, ui)
		if err == nil && launched {
			ui.WriteOutput(MsgSuccess, "Workspace opened successfully in split layout!")
			return nil
		}
		if err != nil {
			ui.WriteOutput(MsgInfo, "Falling back to standard editor: %v", err)
		}
	}

	// Standard editor fallback
	return openStandardEditor(editor, targetFile, ui)
}

// openWorkspaceLayout launches a 3-part layout (Editor, Problem README, Terminal Tester).
// On tiling WMs (like Hyprland), spawning 3 windows causes the WM to naturally tile them.
func openWorkspaceLayout(cfg *config.Config, targetFile, targetDir, readmePath string, hasReadme bool, problemNum, mode string, ui UI) (bool, error) {
	terminal := cfg.GetTerminal()
	editor := cfg.GetEditor()

	// Pick markdown viewer
	readViewer := detectMarkdownViewer()

	isHyprland := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE") != ""
	isSwayOrI3 := os.Getenv("SWAYSOCK") != "" || os.Getenv("I3SOCK") != ""
	inTmux := os.Getenv("TMUX") != ""

	// Mode resolution: if auto, choose best available strategy
	if mode == "auto" {
		if inTmux {
			mode = "tmux"
		} else if isHyprland || isSwayOrI3 {
			mode = "wm"
		} else if _, err := exec.LookPath("tmux"); err == nil {
			mode = "tmux"
		} else {
			mode = "wm"
		}
	}

	switch mode {
	case "wm":
		ui.WriteOutput(MsgInfo, "Launching 3-window workspace layout (Tiling WM / %s)...", terminal)

		// 1. Launch Editor
		targetBase := filepath.Base(targetFile)
		if err := spawnTerminalWindow(terminal, targetDir, "LeetCode: "+targetBase, editor, targetBase); err != nil {
			return false, fmt.Errorf("failed to spawn editor window: %w", err)
		}
		time.Sleep(120 * time.Millisecond)

		// 2. Launch Problem Description (README)
		if hasReadme {
			viewCmd := fmt.Sprintf("%s README.md; exec bash", readViewer)
			_ = spawnTerminalWindow(terminal, targetDir, "LeetCode: Problem Description", "sh", "-c", viewCmd)
			time.Sleep(120 * time.Millisecond)
		}

		// 3. Launch Terminal Tester
		testPrompt := fmt.Sprintf("echo '=== LeetCode Tester: #%s ==='; echo 'Commands:'; echo '  leet test %s --local'; echo '  leet run %s'; echo ''; exec bash", problemNum, problemNum, problemNum)
		_ = spawnTerminalWindow(terminal, targetDir, "LeetCode: Terminal", "sh", "-c", testPrompt)

		return true, nil

	case "tmux":
		if inTmux {
			ui.WriteOutput(MsgInfo, "Splitting current Tmux window into workspace...")
			readCmd := fmt.Sprintf("%s README.md; echo ''; exec bash", readViewer)
			exec.Command("tmux", "split-window", "-h", "-c", targetDir, readCmd).Run()
			exec.Command("tmux", "split-window", "-v", "-c", targetDir, "bash").Run()
			exec.Command("tmux", "select-pane", "-L").Run()
			return true, openStandardEditor(editor, targetFile, ui)
		}

		sessionName := fmt.Sprintf("leet_%s", problemNum)
		ui.WriteOutput(MsgInfo, "Creating Tmux workspace session '%s'...", sessionName)
		exec.Command("tmux", "new-session", "-d", "-s", sessionName, "-c", targetDir, fmt.Sprintf("%s '%s'", editor, filepath.Base(targetFile))).Run()
		readCmd := fmt.Sprintf("%s README.md; exec bash", readViewer)
		exec.Command("tmux", "split-window", "-h", "-t", sessionName, "-c", targetDir, readCmd).Run()
		exec.Command("tmux", "split-window", "-v", "-t", sessionName, "-c", targetDir, "bash").Run()

		attachCmd := exec.Command("tmux", "attach", "-t", sessionName)
		attachCmd.Stdin = os.Stdin
		attachCmd.Stdout = os.Stdout
		attachCmd.Stderr = os.Stderr
		return true, attachCmd.Run()
	}

	return false, nil
}

// spawnTerminalWindow executes a terminal emulator with working directory and command.
func spawnTerminalWindow(term, dir, title string, commandArgs ...string) error {
	termBase := strings.ToLower(filepath.Base(term))

	var cmd *exec.Cmd
	switch termBase {
	case "kitty":
		args := []string{"-d", dir, "-T", title}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)

	case "alacritty":
		args := []string{"--working-directory", dir, "--title", title, "-e"}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)

	case "ghostty":
		args := []string{fmt.Sprintf("--working-directory=%s", dir), fmt.Sprintf("--title=%s", title), "-e"}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)

	case "wezterm":
		args := []string{"start", "--cwd", dir}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)

	case "foot":
		args := []string{"-D", dir, "-T", title}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)

	default:
		// Generic X11 / Wayland terminal emulator fallback
		args := []string{"-e"}
		args = append(args, commandArgs...)
		cmd = exec.Command(term, args...)
		cmd.Dir = dir
	}

	return cmd.Start()
}

func detectMarkdownViewer() string {
	if _, err := exec.LookPath("bat"); err == nil {
		return "bat --paging=always"
	}
	if _, err := exec.LookPath("glow"); err == nil {
		return "glow -p"
	}
	return "less -R"
}

func isTerminalEditor(editor string) bool {
	base := strings.ToLower(filepath.Base(editor))
	switch base {
	case "nvim", "vim", "vi", "nano", "helix", "hx", "micro", "emacs":
		return true
	}
	return false
}

func openStandardEditor(editor, targetFile string, ui UI) error {
	ui.WriteOutput(MsgInfo, "Opening with %s...", editor)

	cmd := exec.Command(editor, targetFile)
	if isTerminalEditor(editor) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			ui.WriteOutput(MsgError, "Failed to run editor '%s': %v", editor, err)
			return fmt.Errorf("editor failed: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Editor closed.")
		return nil
	}

	if err := cmd.Start(); err != nil {
		ui.WriteOutput(MsgError, "Failed to open editor '%s': %v", editor, err)
		ui.WriteOutput(MsgInfo, "You can manually open: %s", targetFile)
		return fmt.Errorf("failed to open editor '%s': %w", editor, err)
	}
	ui.WriteOutput(MsgSuccess, "Problem opened in editor!")
	return nil
}
