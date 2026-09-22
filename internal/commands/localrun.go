package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

// runLocalHarness runs the solution against the example test cases locally,
// without requiring LeetCode cookies. Supported flow:
//
//  1. parse the entry method signature from the solution file
//  2. extract Input/Output examples from the problem description
//  3. generate a self-contained harness for the solution's language
//  4. compile & execute it, then compare each result with the expected JSON
//
// Returns ok=false when the language/harness is unsupported or no examples found.
func runLocalHarness(cfg *config.Config, ui UI, targetFile, content, slug string) bool {
	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	ext := filepath.Ext(targetFile)
	langKey := template.GetLanguageByExtension(languages, ext)
	if langKey == "" {
		ui.WriteOutput(MsgError, "Unsupported file extension for local test.")
		return false
	}

	if slug == "" {
		slug = template.InferSlugFromPath(targetFile, content)
	}
	if slug == "" {
		ui.WriteOutput(MsgError, "Could not infer LeetCode problem slug.")
		return false
	}

	sig, ok := template.ExtractMethodSignature(langKey, content)
	if !ok {
		ui.WriteOutput(MsgError, "Could not parse the entry method (class Solution with a method) for %s.", langKey)
		ui.WriteOutput(MsgInfo, "Local testing works best with LeetCode-style templates containing class Solution.")
		return false
	}

	// Prefer examples from the local README.md (truly offline). Fall back to
	// fetching the problem description from LeetCode when none are available.
	examples := template.ExtractExamplesMarkdown(readLocalReadme(targetFile))
	exampleSource := "local README.md"
	if len(examples) == 0 {
		details, err := api.GetProblemDetails(slug)
		if err != nil {
			ui.WriteOutput(MsgError, "Could not fetch problem details from LeetCode: %v", err)
			return false
		}
		examples = template.ExtractExamples(details.Content)
		exampleSource = "LeetCode"
	}
	if len(examples) == 0 {
		ui.WriteOutput(MsgInfo, "No example input/output blocks found in the problem description (local or LeetCode).")
		return false
	}
	ui.WriteOutput(MsgInfo, "Found %d example testcase(s) from %s.", len(examples), exampleSource)

	testCases := template.BuildTestCases(sig, examples)
	if len(testCases) == 0 {
		ui.WriteOutput(MsgError, "Examples could not be translated into test cases.")
		return false
	}

	harness, ok := template.BuildLocalHarness(langKey, content, sig, testCases)
	if !ok {
		ui.WriteOutput(MsgError, "Local harness is not yet supported for %s.", langKey)
		return false
	}

	dir, err := os.MkdirTemp("", "leet_local_")
	if err != nil {
		ui.WriteOutput(MsgError, "Could not create temp dir: %v", err)
		return false
	}
	defer os.RemoveAll(dir)

	payload := make([]map[string]interface{}, 0, len(testCases))
	for _, tc := range testCases {
		payload = append(payload, map[string]interface{}{"args": tc.Args, "expected": tc.Expected})
	}
	payloadPath := filepath.Join(dir, "cases.json")
	if err := writeJSONFile(payloadPath, payload); err != nil {
		ui.WriteOutput(MsgError, "Could not write test payload: %v", err)
		return false
	}

	srcPath := filepath.Join(dir, harnessFileName(langKey))
	if err := os.WriteFile(srcPath, []byte(harness), 0644); err != nil {
		ui.WriteOutput(MsgError, "Could not write harness: %v", err)
		return false
	}

	// Java needs the solution in a separate Solution.java file.
	javaSolution := ""
	if langKey == "java" {
		javaSolution = filepath.Join(dir, "Solution.java")
		if err := os.WriteFile(javaSolution, []byte(content), 0644); err != nil {
			ui.WriteOutput(MsgError, "Could not write solution: %v", err)
			return false
		}
	}

	ui.WriteOutput(MsgInfo, "Running %d example testcase(s) locally (%s)...", len(testCases), langKey)

	output, err := compileAndRun(langKey, dir, srcPath, payloadPath)
	if err != nil {
		ui.WriteOutput(MsgError, "Local run failed: %v", err)
		if out := output; out != "" {
			ui.WriteOutput(MsgPlain, "%s", truncate(out, 4000))
		}
		return false
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
			status := "✘"
			if !line.ok {
				ui.WriteOutput(MsgError, "Case %d: FAIL — %s", i+1, line.err)
				ui.WriteOutput(MsgPlain, "  expected: %s", expected)
				continue
			}
			ui.WriteOutput(MsgError, "Case %d: %s", i+1, status)
			ui.WriteOutput(MsgPlain, "  expected: %s", expected)
			ui.WriteOutput(MsgPlain, "  got:      %s", line.value)
		}
	}

	ui.WriteOutput(MsgPlain, "")
	if passCount == len(testCases) {
		ui.WriteOutput(MsgSuccess, "All %d test cases passed locally.", len(testCases))
		return true
	}
	ui.WriteOutput(MsgError, "%d/%d test cases passed.", passCount, len(testCases))
	return false
}

type harnessResult struct {
	ok    bool
	value string
	err   string
}

// readLocalReadme returns the contents of the README.md next to a solution
// file (the problem description saved by `add`), or "" when absent.
func readLocalReadme(targetFile string) string {
	data, err := os.ReadFile(filepath.Join(filepath.Dir(targetFile), "README.md"))
	if err != nil {
		return ""
	}
	return string(data)
}

func parseHarnessOutput(output string, n int) []harnessResult {
	results := make([]harnessResult, 0, n)
	for _, line := range strings.Split(output, "\n") {
		if idx := strings.Index(line, "\t"); idx >= 0 {
			kind, value := line[:idx], line[idx+1:]
			if kind == "RESULT" {
				results = append(results, harnessResult{ok: true, value: value})
			} else if kind == "ERROR" {
				results = append(results, harnessResult{ok: false, err: value})
			}
		}
		if len(results) >= n {
			break
		}
	}
	for len(results) < n {
		results = append(results, harnessResult{ok: false, err: "no output (timeout/panic?)"})
	}
	return results
}

func jsonEquals(got, expected string) bool {
	var g, e interface{}
	if err := json.Unmarshal([]byte(got), &g); err != nil {
		return strings.TrimSpace(got) == strings.TrimSpace(expected)
	}
	if err := json.Unmarshal([]byte(expected), &e); err != nil {
		return false
	}
	gb, _ := json.Marshal(g)
	eb, _ := json.Marshal(e)
	return string(gb) == string(eb)
}

func harnessFileName(langKey string) string {
	switch langKey {
	case "python":
		return "harness.py"
	case "javascript":
		return "harness.js"
	case "typescript":
		return "harness.ts"
	case "go":
		return "main.go"
	case "java":
		return "Harness.java"
	case "cpp":
		return "harness.cpp"
	case "c":
		return "harness.c"
	}
	return "harness.txt"
}

func writeJSONFile(path string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "\n... (truncated)"
}

func compileAndRun(langKey, dir, srcPath, payloadPath string) (string, error) {
	switch langKey {
	case "python":
		return runCmdOutput(detectPythonCmd(), srcPath, payloadPath)

	case "javascript":
		return runCmdOutput("node", srcPath, payloadPath)

	case "typescript":
		out, err := runCmdOutput("npx", "tsx", srcPath, payloadPath)
		if err != nil {
			return runCmdOutput("ts-node", srcPath, payloadPath)
		}
		return out, nil

	case "go":
		exe := filepath.Join(dir, "leet_harness.exe")
		if runtime.GOOS != "windows" {
			exe = filepath.Join(dir, "leet_harness")
		}
		out, err := runCmdOutput("go", "build", "-o", exe, srcPath)
		if err != nil {
			return out, err
		}
		return runCmdOutput(exe, payloadPath)

	case "java":
		out, err := runCmdOutput("javac", "-d", dir, filepath.Join(dir, "Solution.java"), srcPath)
		if err != nil {
			return out, err
		}
		return runCmdOutput("java", "-cp", dir, "Harness")

	case "cpp":
		exe := filepath.Join(dir, "leet_harness.exe")
		if runtime.GOOS != "windows" {
			exe = filepath.Join(dir, "leet_harness")
		}
		out, err := runCmdOutput("g++", "-std=c++17", "-O2", "-o", exe, srcPath)
		if err != nil {
			return out, err
		}
		return runCmdOutput(exe)

	case "c":
		exe := filepath.Join(dir, "leet_harness.exe")
		if runtime.GOOS != "windows" {
			exe = filepath.Join(dir, "leet_harness")
		}
		out, err := runCmdOutput("gcc", "-std=c11", "-O2", "-o", exe, srcPath)
		if err != nil {
			return out, err
		}
		return runCmdOutput(exe)
	}
	return "", fmt.Errorf("local harness unsupported for %s", langKey)
}

// harnessTimeout bounds compile + run of the local test harness so a hung
// solution (infinite loop) or a stuck compiler cannot block the CLI forever.
const harnessTimeout = 120 * time.Second

func runCmdOutput(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), harnessTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("timed out after %s", harnessTimeout)
	}
	return string(out), err
}
