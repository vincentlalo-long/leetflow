package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"leetcli/internal/config"
)

func TestViewProblemLocal(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.BaseDir = tmpDir

	probDir := filepath.Join(tmpDir, "array", "1-two-sum")
	if err := os.MkdirAll(probDir, 0755); err != nil {
		t.Fatalf("failed to create probDir: %v", err)
	}

	solFile := filepath.Join(probDir, "1_Two_Sum.cpp")
	if err := os.WriteFile(solFile, []byte("class Solution {};"), 0644); err != nil {
		t.Fatalf("failed to write solFile: %v", err)
	}

	readmeFile := filepath.Join(probDir, "README.md")
	readmeContent := `# [1. Two Sum](https://leetcode.com/problems/two-sum/)

- **Difficulty:** Easy
- **Tags:** Array, Hash Table

## Description

Given an array of integers nums and an integer target, return indices of the two numbers such that they add up to target.

![Two Sum Diagram](https://assets.leetcode.com/uploads/example.png)

### Example 1:
Input: nums = [2,7,11,15], target = 9
Output: [0,1]
`
	if err := os.WriteFile(readmeFile, []byte(readmeContent), 0644); err != nil {
		t.Fatalf("failed to write readmeFile: %v", err)
	}

	ui := &fakeUI{}
	err := ViewProblem([]string{"1", "--raw"}, cfg, ui)
	if err != nil {
		t.Fatalf("ViewProblem failed: %v", err)
	}

	out := ui.Output()
	if !strings.Contains(out, "Two Sum") {
		t.Errorf("expected 'Two Sum' in output, got: %s", out)
	}
	if !strings.Contains(out, "https://assets.leetcode.com/uploads/example.png") {
		t.Errorf("expected image url in output, got: %s", out)
	}

	// Test Glamour rendering
	ui = &fakeUI{}
	err = ViewProblem([]string{"1", "--no-pager"}, cfg, ui)
	if err != nil {
		t.Fatalf("ViewProblem with glamour failed: %v", err)
	}

	glamourOut := ui.Output()
	if !strings.Contains(glamourOut, "Two Sum") {
		t.Errorf("expected 'Two Sum' in glamour output, got: %s", glamourOut)
	}
	if !strings.Contains(glamourOut, "Attached Diagram(s)") {
		t.Errorf("expected diagram info in glamour output, got: %s", glamourOut)
	}
}
