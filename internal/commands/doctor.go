package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

// DoctorCommand is the handler for `leet doctor`. It validates the workspace
// configuration and directory structure, reporting issues that need attention.
func DoctorCommand(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Workspace Health Check ---\n")

	problems := 0
	fixable := 0

	// 1. Config file status.
	configPath := cfg.GetPath()
	if cfg.HasLocalConfig() {
		ui.WriteOutput(MsgSuccess, "Local config active: %s", filepath.Base(configPath))
	} else {
		ui.WriteOutput(MsgInfo, "Using base config (run 'leet init' to create machine-specific %s)", filepath.Base(configPath))
	}

	// 2. Base directory.
	expandedBase := config.ExpandHome(cfg.BaseDir)
	if cfg.BaseDir == "" {
		ui.WriteOutput(MsgError, "base_dir is not configured (run 'leet init')")
		problems++
		fixable++
	} else if _, err := os.Stat(expandedBase); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "base_dir does not exist: %s (run 'leet init' to create)", expandedBase)
		problems++
		fixable++
	} else {
		ui.WriteOutput(MsgSuccess, "base_dir exists: %s", expandedBase)
	}

	// 3. Default language.
	if cfg.DefaultLanguage == "" {
		ui.WriteOutput(MsgError, "default_language is not set")
		problems++
		fixable++
	} else {
		ui.WriteOutput(MsgSuccess, "default_language: %s", cfg.DefaultLanguage)
	}

	// 4. Editor.
	if cfg.GetEditor() == "" {
		ui.WriteOutput(MsgError, "editor is not configured")
		problems++
		fixable++
	} else {
		ui.WriteOutput(MsgSuccess, "editor: %s", cfg.GetEditor())
	}

	// 5. Data structures.
	ds := cfg.GetDataStructures()
	if len(ds) == 0 {
		ui.WriteOutput(MsgError, "No data structures configured")
		problems++
		fixable++
	} else {
		ui.WriteOutput(MsgSuccess, "%d data structure(s) configured", len(ds))
	}

	// 6. LeetCode cookies.
	if cfg.GetLeetcodeSession() == "" {
		ui.WriteOutput(MsgError, "LeetCode session cookie is not set (submit/sync require it)")
		ui.WriteOutput(MsgInfo, "Set via config menu, config.local.json, or LEETCODE_SESSION env var")
		problems++
	} else {
		ui.WriteOutput(MsgSuccess, "LeetCode session cookie is set")
	}
	if cfg.GetLeetcodeCsrf() == "" {
		ui.WriteOutput(MsgError, "LeetCode CSRF token is not set")
		ui.WriteOutput(MsgInfo, "Set via config menu, config.local.json, or LEETCODE_CSRF env var")
		problems++
	} else {
		ui.WriteOutput(MsgSuccess, "LeetCode CSRF token is set")
	}

	// 7. Languages with extensions.
	langs := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		langs[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range langs {
		exts = append(exts, info.Ext)
	}
	if len(exts) == 0 {
		ui.WriteOutput(MsgError, "No language extensions configured")
		problems++
	} else {
		ui.WriteOutput(MsgSuccess, "%d language(s) configured", len(langs))
	}

	if expandedBase == "" {
		ui.WriteOutput(MsgInfo, "Skipping workspace scan (base_dir not set)")
		return fmt.Errorf("base_dir not configured")
	}

	// 8. Scan workspace for issues.
	ui.WriteOutput(MsgPlain, "\nScanning workspace...")
	scanIssues := scanWorkspace(expandedBase, ds, exts, ui)
	problems += scanIssues

	// 9. Check .gitignore has config.local.json.
	ui.WriteOutput(MsgPlain, "")
	leetcodercPath := filepath.Join(expandedBase, ".gitignore")
	if _, err := os.Stat(leetcodercPath); err == nil {
		if data, err := os.ReadFile(leetcodercPath); err == nil {
			if strings.Contains(string(data), "config.local.json") {
				ui.WriteOutput(MsgSuccess, ".gitignore contains config.local.json")
			} else {
				ui.WriteOutput(MsgError, ".gitignore should contain config.local.json to protect secrets")
				fixable++
			}
		}
	}

	ui.WriteOutput(MsgPlain, "")
	if problems == 0 {
		ui.WriteOutput(MsgSuccess, "All checks passed!")
	} else {
		ui.WriteOutput(MsgError, "Found %d issue(s) (%d fixable)", problems, fixable)
		ui.WriteOutput(MsgInfo, "Run 'leet config' to fix config issues, or manually edit files.")
	}

	if problems > 0 {
		return fmt.Errorf("found %d issue(s)", problems)
	}
	return nil
}

// scanWorkspace walks the workspace and reports structural issues.
func scanWorkspace(baseDir string, ds map[string]string, exts []string, ui UI) int {
	clean := filepath.Clean(baseDir)
	home, _ := os.UserHomeDir()
	isHome := home != "" && strings.EqualFold(clean, filepath.Clean(home))
	vol := filepath.VolumeName(clean)
	isDriveRoot := clean == "/" || clean == "\\" || clean == vol || clean == vol+"\\" || clean == vol+"/"
	if isHome || isDriveRoot {
		ui.WriteOutput(MsgError, "base_dir is set to an unsafe root directory (%s)!", baseDir)
		ui.WriteOutput(MsgInfo, "Scanning was blocked to prevent traversing your entire filesystem.")
		if runtime.GOOS == "windows" {
			ui.WriteOutput(MsgInfo, "Please change base_dir to a dedicated folder like 'D:\\leetcode' or 'C:\\leetcode' via 'leet config'.")
		} else {
			ui.WriteOutput(MsgInfo, "Please change base_dir to a dedicated folder like '~/leetcode' via 'leet config'.")
		}
		return 1
	}

	var orphanFiles []string
	var missingReadme []string
	var badNames []string
	totalSolutions := 0

	// Collect data_structure folders for orphan detection.
	dsFolders := make(map[string]bool)
	for _, folder := range ds {
		dsFolders[strings.ToLower(folder)] = true
	}

	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path != baseDir {
				// Skip any hidden directory (.git, .cache, .local, etc.)
				if strings.HasPrefix(info.Name(), ".") || template.IsIgnoredDir(info.Name()) {
					return filepath.SkipDir
				}
				rel, _ := filepath.Rel(baseDir, path)
				parts := strings.Split(filepath.ToSlash(rel), "/")
				// Only scan into configured category folders or "uncategorized"
				if len(parts) == 1 {
					top := strings.ToLower(parts[0])
					if !dsFolders[top] && top != "uncategorized" {
						return filepath.SkipDir
					}
				}
				// Don't recurse deeper than category / problem-folder
				if len(parts) > 2 {
					return filepath.SkipDir
				}
			}
			return nil
		}

		rel, _ := filepath.Rel(baseDir, path)
		parent := filepath.Dir(rel)
		ext := strings.TrimLeft(filepath.Ext(path), ".")

		// Check if it's a known solution file.
		isSolution := false
		for _, e := range exts {
			if strings.EqualFold(ext, strings.TrimLeft(e, ".")) {
				isSolution = true
				break
			}
		}

		if isSolution {
			totalSolutions++
			parentDir := strings.ToLower(filepath.Base(parent))
			if !dsFolders[parentDir] && parent != "." && parent != "" {
				// Check if grandparent is a DS folder (nested problem dir).
				grandparent := strings.ToLower(filepath.Base(filepath.Dir(parent)))
				if !dsFolders[grandparent] {
					orphanFiles = append(orphanFiles, rel)
				}
			}

			// Check for missing README.md in same directory.
			readmePath := filepath.Join(filepath.Dir(path), "README.md")
			if _, err := os.Stat(readmePath); os.IsNotExist(err) {
				missingReadme = append(missingReadme, rel)
			}

			// Check filename for suspicious characters.
			baseName := filepath.Base(path)
			if badNameRe.MatchString(baseName) {
				badNames = append(badNames, rel)
			}
		}

		return nil
	})

	issues := 0
	if len(orphanFiles) > 0 {
		ui.WriteOutput(MsgError, "%d solution file(s) not inside a data_structure folder:", len(orphanFiles))
		for _, f := range orphanFiles {
			ui.WriteOutput(MsgPlain, "    %s", f)
		}
		ui.WriteOutput(MsgInfo, "Move them into a configured folder (array/, tree/, etc.) or add a new category.")
		issues += len(orphanFiles)
	}
	if len(missingReadme) > 0 {
		ui.WriteOutput(MsgError, "%d solution file(s) missing README.md:", len(missingReadme))
		for _, f := range missingReadme {
			ui.WriteOutput(MsgPlain, "    %s", f)
		}
		ui.WriteOutput(MsgInfo, "Run 'leet add <num>' to create the README, or write one manually.")
		issues += len(missingReadme)
	}
	if len(badNames) > 0 {
		ui.WriteOutput(MsgError, "%d file(s) have suspicious filenames:", len(badNames))
		for _, f := range badNames {
			ui.WriteOutput(MsgPlain, "    %s", f)
		}
		ui.WriteOutput(MsgInfo, "Rename files to use only alphanumeric, hyphen, and underscore characters.")
		issues += len(badNames)
	}
	if len(orphanFiles) == 0 && len(missingReadme) == 0 && len(badNames) == 0 {
		ui.WriteOutput(MsgSuccess, "No workspace structural issues found (%d solution files scanned)", totalSolutions)
	}

	return issues
}

var badNameRe = regexp.MustCompile(`[^a-zA-Z0-9_\-.]`)
