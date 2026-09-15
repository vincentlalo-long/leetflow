package commands

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

func TestProblem(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- LeetCode Test ---\n")

	problemNum := ""
	if len(args) > 0 {
		problemNum = args[0]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number to test")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Invalid base directory in config")
		return fmt.Errorf("invalid base directory in config")
	}

	targetFile, content := resolveLocalSolution(baseDir, problemNum, cfg, ui)
	if targetFile == "" {
		return fmt.Errorf("could not find local file for problem %s", problemNum)
	}

	// Local (cookie-free) harness mode: leet test <num> --local
	pos, flags := parseFlags(args)
	if hasFlag(flags, "local") {
		passed := runLocalHarness(cfg, ui, targetFile, content, "")
		if passed {
			recordLocalPass(cfg, ui, targetFile, content)
			return nil
		}
		return fmt.Errorf("local test cases failed for problem %s", problemNum)
	}
	_ = pos

	ext := filepath.Ext(targetFile)
	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		ui.WriteOutput(MsgError, "Unsupported file extension for testing.")
		return fmt.Errorf("unsupported file extension for testing")
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
		ui.WriteOutput(MsgError, "LeetCode test requires leetcode_session and leetcode_csrf.")
		ui.WriteOutput(MsgInfo, "Set them in config.local.json or use LEETCODE_SESSION / LEETCODE_CSRF env vars.")
		return fmt.Errorf("leetcode_session and leetcode_csrf are required")
	}

	ui.WriteOutput(MsgInfo, "Fetching example testcases for slug: %s...", slug)
	testcases, err := api.GetProblemTestcases(slug)
	if err != nil || testcases == "" {
		ui.WriteOutput(MsgInfo, "Could not fetch sample testcases automatically.")
		testcases = ui.PromptText("Enter custom input testcases")
		if testcases == "" {
			return fmt.Errorf("no testcases provided")
		}
	} else {
		ui.WriteOutput(MsgSuccess, "Sample testcases fetched.")
	}

	ui.WriteOutput(MsgInfo, "Sending test run to LeetCode (%s)...", lcLang)
	interpretID, err := api.InterpretSolution(session, csrf, slug, details.QuestionID, lcLang, content, testcases)
	if err != nil {
		ui.WriteOutput(MsgError, "Test run request failed: %v", err)
		return fmt.Errorf("test run request failed: %w", err)
	}

	ui.WriteOutput(MsgInfo, "Test Run ID: %s. Polling for results...", interpretID)
	for i := 0; i < 15; i++ {
		time.Sleep(2 * time.Second)
		res, err := api.CheckSubmissionStatus(session, csrf, interpretID)
		if err != nil {
			ui.WriteOutput(MsgError, "Error checking status: %v", err)
			continue
		}
		if res.State == "PENDING" || res.State == "STARTED" {
			ui.WriteOutput(MsgInfo, "Status: %s...", res.State)
			continue
		}

		if res.StatusMsg == "Accepted" {
			ui.WriteOutput(MsgSuccess, "✔ TEST PASSED!")
			if res.StatusRuntime != "" {
				ui.WriteOutput(MsgPlain, "  Runtime: %s", res.StatusRuntime)
			}
			if res.StatusMemory != "" {
				ui.WriteOutput(MsgPlain, "  Memory:  %s", res.StatusMemory)
			}
			if len(res.CodeOutput) > 0 {
				ui.WriteOutput(MsgPlain, "  Your Output:     %s", strings.Join(res.CodeOutput, ", "))
			}
			if len(res.ExpectedOutput) > 0 {
				ui.WriteOutput(MsgPlain, "  Expected Output: %s", strings.Join(res.ExpectedOutput, ", "))
			}
			if len(res.StdOutput) > 0 {
				ui.WriteOutput(MsgInfo, "  Stdout Output:\n%s", strings.Join(res.StdOutput, "\n"))
			}
		} else {
			ui.WriteOutput(MsgError, "✘ TEST FAILED: %s", res.StatusMsg)
			if len(res.CodeOutput) > 0 {
				ui.WriteOutput(MsgPlain, "  Your Output:     %s", strings.Join(res.CodeOutput, ", "))
			}
			if len(res.ExpectedOutput) > 0 {
				ui.WriteOutput(MsgPlain, "  Expected Output: %s", strings.Join(res.ExpectedOutput, ", "))
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
			return fmt.Errorf("test failed: %s", res.StatusMsg)
		}
		return nil
	}

	ui.WriteOutput(MsgError, "Timed out waiting for test run result.")
	return fmt.Errorf("timed out waiting for test run result")
}
