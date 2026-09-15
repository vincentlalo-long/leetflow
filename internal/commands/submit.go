package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
	"leetcli/internal/tracker"
)

func SubmitProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- LeetCode Submit ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number to submit")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Invalid base directory in config")
		return fmt.Errorf("invalid base directory in config")
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
		selected := ui.PromptSelect("Multiple matches. Select file to submit", names)
		for i, n := range names {
			if n == selected {
				targetFile = matches[i]
				break
			}
		}
	}

	contentBytes, err := os.ReadFile(targetFile)
	if err != nil {
		ui.WriteOutput(MsgError, "Failed to read file: %v", err)
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	ext := filepath.Ext(targetFile)
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		ui.WriteOutput(MsgError, "Unsupported file extension for submission.")
		return fmt.Errorf("unsupported file extension for submission")
	}

	lcLang, ok := template.LeetCodeLangMap[langKey]
	if !ok {
		ui.WriteOutput(MsgError, "Language '%s' is not supported by LeetCode API.", langKey)
		return fmt.Errorf("language '%s' is not supported by LeetCode API", langKey)
	}

	slug := template.InferSlugFromPath(targetFile, content)
	if slug == "" {
		slug = ui.PromptText("Enter LeetCode problem slug (e.g., two-sum)")
		if slug == "" {
			return fmt.Errorf("problem slug is required")
		}
	}

	details, err := api.GetProblemDetails(slug)
	if err != nil {
		ui.WriteOutput(MsgError, "Could not fetch problem details from LeetCode: %v", err)
		return fmt.Errorf("could not fetch problem details: %w", err)
	}

	session := cfg.GetLeetcodeSession()
	csrf := cfg.GetLeetcodeCsrf()
	if session == "" || csrf == "" {
		ui.WriteOutput(MsgError, "LeetCode submit requires leetcode_session and leetcode_csrf.")
		ui.WriteOutput(MsgInfo, "Set them in config.local.json or use LEETCODE_SESSION / LEETCODE_CSRF env vars.")
		return fmt.Errorf("leetcode_session and leetcode_csrf are required")
	}

	ui.WriteOutput(MsgInfo, "Submitting %s solution to LeetCode (slug: %s)...", lcLang, slug)
	subID, err := api.SubmitSolution(session, csrf, slug, details.QuestionID, lcLang, content)
	if err != nil {
		ui.WriteOutput(MsgError, "Submission failed: %v", err)
		return fmt.Errorf("submission failed: %w", err)
	}

	ui.WriteOutput(MsgInfo, "Submission ID: %s. Polling for results...", subID)
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		res, err := api.CheckSubmissionStatus(session, csrf, subID)
		if err != nil {
			ui.WriteOutput(MsgError, "Error checking status: %v", err)
			continue
		}
		if res.State == "PENDING" || res.State == "STARTED" {
			ui.WriteOutput(MsgInfo, "Status: %s...", res.State)
			continue
		}

		if res.StatusMsg == "Accepted" {
			ui.WriteOutput(MsgSuccess, "✔ ACCEPTED!")
			runtime := ""
			memory := ""
			if res.StatusRuntime != "" {
				runtime = res.StatusRuntime
				ui.WriteOutput(MsgPlain, "  Runtime: %s (Beats %.2f%%)", runtime, res.RuntimePercentile)
			}
			if res.StatusMemory != "" {
				memory = res.StatusMemory
				ui.WriteOutput(MsgPlain, "  Memory:  %s (Beats %.2f%%)", memory, res.MemoryPercentile)
			}
			if res.TotalTestcases > 0 {
				ui.WriteOutput(MsgPlain, "  Testcases: %d / %d", res.TotalCorrect, res.TotalTestcases)
			}
			recordSubmission(cfg, targetFile, slug, details, runtime, memory, true, ui)
			return nil
		} else {
			ui.WriteOutput(MsgError, "✘ Result: %s", res.StatusMsg)
			if res.TotalTestcases > 0 {
				ui.WriteOutput(MsgPlain, "  Passed: %d / %d testcases", res.TotalCorrect, res.TotalTestcases)
			}
			if res.CompileError != "" || res.FullCompileError != "" {
				errMsg := res.CompileError
				if errMsg == "" {
					errMsg = res.FullCompileError
				}
				ui.WriteOutput(MsgError, "Compile Error:\n%s", errMsg)
			}
			if res.RuntimeError != "" || res.FullRuntimeError != "" {
				errMsg := res.RuntimeError
				if errMsg == "" {
					errMsg = res.FullRuntimeError
				}
				ui.WriteOutput(MsgError, "Runtime Error:\n%s", errMsg)
			}
			recordSubmission(cfg, targetFile, slug, details, "", "", false, ui)
			return fmt.Errorf("submission not accepted: %s", res.StatusMsg)
		}
	}

	ui.WriteOutput(MsgError, "Timed out waiting for submission result.")
	return fmt.Errorf("timed out waiting for submission result")
}

// recordLocalPass marks a problem solved in the tracker when all local example
// test cases pass, without needing LeetCode cookies.
func recordLocalPass(cfg *config.Config, ui UI, targetFile, content string) {
	if cfg.BaseDir == "" {
		return
	}
	slug := template.InferSlugFromPath(targetFile, content)
	if slug == "" {
		ui.WriteOutput(MsgInfo, "Progress not saved: could not infer problem slug.")
		return
	}
	details, err := api.GetProblemDetails(slug)
	if err != nil {
		ui.WriteOutput(MsgInfo, "Progress not saved: could not fetch problem details (%v).", err)
		return
	}

	prog := tracker.Load(cfg.BaseDir)
	number := ""
	title := details.Title
	difficulty := details.Difficulty
	category := template.DetectStructure(targetFile, cfg.GetDataStructures())
	tags := make([]string, len(details.TopicTags))
	for i, t := range details.TopicTags {
		tags[i] = t.Name
	}

	base := filepath.Base(targetFile)
	re := regexp.MustCompile(`^(\d+)`)
	if m := re.FindStringSubmatch(base); len(m) == 2 {
		number = m[1]
	}
	if number == "" {
		number = api.Slugify(title)
	}

	prog.Upsert(cfg.BaseDir, number, title, slug, difficulty, category, tags, "solved", "", "")
	if err := prog.Save(cfg.BaseDir); err != nil {
		ui.WriteOutput(MsgError, "Warning: could not save progress: %v", err)
		return
	}
	ui.WriteOutput(MsgSuccess, "Progress saved: %s marked as solved.", number)
}

// recordSubmission persists the result of a submit/test run into the progress tracker.
func recordSubmission(cfg *config.Config, targetFile, slug string, details *api.ProblemDetail, runtime, memory string, accepted bool, ui UI) {
	if cfg.BaseDir == "" {
		return
	}
	prog := tracker.Load(cfg.BaseDir)
	number := ""
	title := details.Title
	difficulty := details.Difficulty
	category := template.DetectStructure(targetFile, cfg.GetDataStructures())
	tags := make([]string, len(details.TopicTags))
	for i, t := range details.TopicTags {
		tags[i] = t.Name
	}

	base := filepath.Base(targetFile)
	re := regexp.MustCompile(`^(\d+)`)
	if m := re.FindStringSubmatch(base); len(m) == 2 {
		number = m[1]
	}
	if number == "" {
		number = api.Slugify(title)
	}

	status := "unsolved"
	if accepted {
		status = "solved"
	}
	prog.Upsert(cfg.BaseDir, number, title, slug, difficulty, category, tags, status, runtime, memory)
	if err := prog.Save(cfg.BaseDir); err != nil {
		ui.WriteOutput(MsgError, "Warning: could not save progress: %v", err)
		return
	}
	if accepted {
		ui.WriteOutput(MsgSuccess, "Progress saved: %s marked as solved.", number)
	} else {
		ui.WriteOutput(MsgInfo, "Submission attempt recorded for %s.", number)
	}
}
