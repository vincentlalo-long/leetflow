package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

// VerifyCommand is the handler for `leet verify`. It runs the local test
// harness on a file without any interactive prompts — suitable for CI/CD.
//
// Usage: leet verify <file_path> [--local]
func VerifyCommand(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Auto-Verify Solution ---\n")

	targetFile := ""
	localOnly := false

	// Parse arguments.
	for _, a := range args {
		if a == "--local" {
			localOnly = true
		} else if !strings.HasPrefix(a, "-") {
			targetFile = a
		}
	}

	if targetFile == "" {
		return fmt.Errorf("usage: leet verify <file_path> [--local]")
	}

	// Resolve to absolute path.
	absPath, err := filepath.Abs(targetFile)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("file not found: %s", targetFile)
	}

	// Infer problem number from filename.
	baseName := filepath.Base(absPath)
	problemNum := ""
	re := regexp.MustCompile(`^(\d+)`)
	if m := re.FindStringSubmatch(baseName); len(m) == 2 {
		problemNum = m[1]
	}
	if problemNum == "" {
		return fmt.Errorf("cannot infer problem number from filename: %s", baseName)
	}

	ui.WriteOutput(MsgInfo, "File: %s", baseName)
	ui.WriteOutput(MsgInfo, "Problem: #%s", problemNum)

	// Read file content.
	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	// Determine language.
	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	ext := filepath.Ext(absPath)
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		return fmt.Errorf("unsupported file extension: %s", ext)
	}

	// Extract method signature.
	sig, ok := template.ExtractMethodSignature(langKey, content)
	if !ok {
		return fmt.Errorf("could not parse entry method for %s", langKey)
	}

	// Infer slug from file path and content.
	slug := template.InferSlugFromPath(absPath, content)

	// Fetch examples from local README or LeetCode API.
	examples := template.ExtractExamplesMarkdown(readLocalReadme(absPath))
	exampleSource := "local README"
	if len(examples) == 0 && !localOnly {
		if slug != "" {
			ui.WriteOutput(MsgInfo, "No local examples, fetching from LeetCode API...")
			details, err := api.GetProblemDetails(slug)
			if err == nil {
				examples = template.ExtractExamples(details.Content)
				exampleSource = "LeetCode API"
			}
		}
	}
	if len(examples) == 0 {
		return fmt.Errorf("no example test cases found for problem %s", problemNum)
	}

	// Build test cases.
	testCases := template.BuildTestCases(sig, examples)
	if len(testCases) == 0 {
		return fmt.Errorf("could not translate examples into test cases")
	}

	ui.WriteOutput(MsgInfo, "%d test case(s) from %s", len(testCases), exampleSource)

	// Build local harness.
	harness, ok := template.BuildLocalHarness(langKey, content, sig, testCases)
	if !ok {
		return fmt.Errorf("local harness not supported for %s", langKey)
	}

	// Write harness + payload to temp dir.
	dir, err := os.MkdirTemp("", "leet_verify_")
	if err != nil {
		return fmt.Errorf("could not create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	payload := make([]map[string]interface{}, 0, len(testCases))
	for _, tc := range testCases {
		payload = append(payload, map[string]interface{}{"args": tc.Args, "expected": tc.Expected})
	}
	payloadPath := filepath.Join(dir, "cases.json")
	if err := writeJSONFile(payloadPath, payload); err != nil {
		return fmt.Errorf("could not write test payload: %w", err)
	}

	srcPath := filepath.Join(dir, harnessFileName(langKey))
	if err := os.WriteFile(srcPath, []byte(harness), 0644); err != nil {
		return fmt.Errorf("could not write harness: %w", err)
	}

	// Java needs separate Solution.java.
	if langKey == "java" {
		solPath := filepath.Join(dir, "Solution.java")
		if err := os.WriteFile(solPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("could not write solution: %w", err)
		}
	}

	ui.WriteOutput(MsgInfo, "Running %d test case(s) (%s)...", len(testCases), langKey)
	output, err := compileAndRun(langKey, dir, srcPath, payloadPath)
	if err != nil {
		ui.WriteOutput(MsgError, "Run failed: %v", err)
		if out := output; out != "" {
			ui.WriteOutput(MsgPlain, "%s", truncate(out, 4000))
		}
		return fmt.Errorf("local run failed: %w", err)
	}

	results := parseHarnessOutput(output, len(testCases))
	passCount := 0
	for i, line := range results {
		expected := testCases[i].Expected
		if line.ok && jsonEquals(line.value, expected) {
			passCount++
			ui.WriteOutput(MsgSuccess, "Case %d: PASS", i+1)
			if line.value != "" {
				ui.WriteOutput(MsgPlain, "  output: %s", line.value)
			}
		} else {
			if !line.ok {
				ui.WriteOutput(MsgError, "Case %d: FAIL — %s", i+1, line.err)
				ui.WriteOutput(MsgPlain, "  expected: %s", expected)
				continue
			}
			ui.WriteOutput(MsgError, "Case %d: FAIL", i+1)
			ui.WriteOutput(MsgPlain, "  expected: %s", expected)
			ui.WriteOutput(MsgPlain, "  got:      %s", line.value)
		}
	}

	ui.WriteOutput(MsgPlain, "")
	if passCount == len(testCases) {
		ui.WriteOutput(MsgSuccess, "All %d test case(s) passed.", len(testCases))
		return nil
	}
	ui.WriteOutput(MsgError, "%d/%d test case(s) passed.", passCount, len(testCases))
	return fmt.Errorf("%d/%d test cases failed", len(testCases)-passCount, len(testCases))
}
