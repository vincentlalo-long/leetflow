package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func RunProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Run Code ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number to run")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Invalid base directory in config")
		return fmt.Errorf("invalid base directory in config")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist")
		return fmt.Errorf("base directory does not exist")
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
		return fmt.Errorf("could not find local file for problem %s", problemNum)
	}

	targetFile := matches[0]
	if len(matches) > 1 {
		names := make([]string, len(matches))
		for i, f := range matches {
			names[i] = filepath.Base(f)
		}
		selected := ui.PromptSelect("Multiple matches. Select file to run", names)
		for i, n := range names {
			if n == selected {
				targetFile = matches[i]
				break
			}
		}
	}

	ui.WriteOutput(MsgSuccess, "Found: %s", filepath.Base(targetFile))

	contentBytes, err := os.ReadFile(targetFile)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to read file: %v", err)
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	ext := filepath.Ext(targetFile)
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		ui.WriteOutput(MsgError, "Unsupported file extension.")
		return fmt.Errorf("unsupported file extension")
	}

	switch langKey {
	case "cpp", "c":
		if !strings.Contains(content, "main(") {
			ui.WriteOutput(MsgInfo, "No main() function detected. Will compile anyway for syntax check.")
		}

		compiler := "g++"
		stdFlag := "-std=c++17"
		if langKey == "c" {
			compiler = "gcc"
			stdFlag = "-std=c11"
		}

		tmpDir := os.TempDir()
		exeName := fmt.Sprintf("leet_run_%s", problemNum)
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}
		exePath := filepath.Join(tmpDir, exeName)

		ui.WriteOutput(MsgInfo, "Compiling with %s...", compiler)
		compileCmd := exec.Command(compiler, stdFlag, "-O2", "-Wall", targetFile, "-o", exePath)
		compileOut, err := compileCmd.CombinedOutput()
		if err != nil {
			ui.WriteOutput(MsgError, "Compilation failed!")
			ui.WriteOutput(MsgPlain, "%s", string(compileOut))
			return fmt.Errorf("compilation failed: %w", err)
		}
		if len(compileOut) > 0 {
			ui.WriteOutput(MsgInfo, "Compiled with warnings:\n%s", string(compileOut))
		} else {
			ui.WriteOutput(MsgSuccess, "Compiled successfully.")
		}

		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		runCmd := exec.Command(exePath)
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin
		runCmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")
		os.Remove(exePath)

	case "python":
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		cmd := exec.Command(detectPythonCmd(), targetFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")

	case "go":
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		cmd := exec.Command("go", "run", targetFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")

	case "java":
		ui.WriteOutput(MsgInfo, "Compiling Java solution...")
		tmpDir := os.TempDir()
		compileCmd := exec.Command("javac", "-d", tmpDir, targetFile)
		compileOut, err := compileCmd.CombinedOutput()
		if err != nil {
			ui.WriteOutput(MsgError, "Java compilation failed!\n%s", string(compileOut))
			return fmt.Errorf("java compilation failed: %w", err)
		}
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		runCmd := exec.Command("java", "-cp", tmpDir, "Solution")
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin
		runCmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")

	case "rust":
		tmpDir := os.TempDir()
		exeName := fmt.Sprintf("leet_run_rs_%s", problemNum)
		if runtime.GOOS == "windows" {
			exeName += ".exe"
		}
		exePath := filepath.Join(tmpDir, exeName)

		ui.WriteOutput(MsgInfo, "Compiling with rustc...")
		compileCmd := exec.Command("rustc", targetFile, "-o", exePath)
		compileOut, err := compileCmd.CombinedOutput()
		if err != nil {
			ui.WriteOutput(MsgError, "Rust compilation failed!\n%s", string(compileOut))
			return fmt.Errorf("rust compilation failed: %w", err)
		}
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		runCmd := exec.Command(exePath)
		runCmd.Stdout = os.Stdout
		runCmd.Stderr = os.Stderr
		runCmd.Stdin = os.Stdin
		runCmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")
		os.Remove(exePath)

	case "javascript":
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		cmd := exec.Command("node", targetFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")

	case "typescript":
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		cmd := exec.Command("npx", "tsx", targetFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		if err := cmd.Run(); err != nil {
			// Fallback to ts-node if npx tsx fails
			fallbackCmd := exec.Command("ts-node", targetFile)
			fallbackCmd.Stdout = os.Stdout
			fallbackCmd.Stderr = os.Stderr
			fallbackCmd.Stdin = os.Stdin
			fallbackCmd.Run()
		}
		ui.WriteOutput(MsgPlain, "\n---\n")

	case "csharp":
		ui.WriteOutput(MsgPlain, "\n--- Output ---\n")
		cmd := exec.Command("dotnet", "script", targetFile)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		cmd.Run()
		ui.WriteOutput(MsgPlain, "\n---\n")

	default:
		ui.WriteOutput(MsgError, "Running '%s' files is not supported yet.", langKey)
		return fmt.Errorf("running '%s' files is not supported yet", langKey)
	}
	return nil
}

// detectPythonCmd returns "python3" if available, falling back to "python".
func detectPythonCmd() string {
	if _, err := exec.LookPath("python3"); err == nil {
		return "python3"
	}
	return "python"
}
