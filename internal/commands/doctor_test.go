package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanWorkspaceGuards(t *testing.T) {
	ui := &fakeUI{}
	ds := map[string]string{"array": "array"}
	exts := []string{"cpp", "py"}

	// 1. Scanning "/" or drive root must be blocked
	code := scanWorkspace("/", ds, exts, ui)
	if code == 0 {
		t.Errorf("scanWorkspace('/') expected non-zero error code, got 0")
	}
	if !strings.Contains(ui.Output(), "unsafe root directory") {
		t.Errorf("expected unsafe root directory warning, got: %s", ui.Output())
	}

	// 2. Scanning home dir must be blocked
	home, _ := os.UserHomeDir()
	if home != "" {
		ui = &fakeUI{}
		code = scanWorkspace(home, ds, exts, ui)
		if code == 0 {
			t.Errorf("scanWorkspace(home) expected non-zero error code, got 0")
		}
		if !strings.Contains(ui.Output(), "unsafe root directory") {
			t.Errorf("expected unsafe root directory warning, got: %s", ui.Output())
		}
	}

	// 3. Normal workspace directory with category filtering
	tmpDir := t.TempDir()
	// Category folder
	arrDir := filepath.Join(tmpDir, "array", "1_two_sum")
	os.MkdirAll(arrDir, 0755)
	os.WriteFile(filepath.Join(arrDir, "solution.cpp"), []byte("// solution"), 0644)
	os.WriteFile(filepath.Join(arrDir, "README.md"), []byte("# 1. Two Sum"), 0644)

	// Non-category folder (e.g. downloads, photos, node_modules) should be skipped
	otherDir := filepath.Join(tmpDir, "random_folder", "nested")
	os.MkdirAll(otherDir, 0755)
	os.WriteFile(filepath.Join(otherDir, "random.cpp"), []byte("// not leetcode"), 0644)

	ui = &fakeUI{}
	code = scanWorkspace(tmpDir, ds, exts, ui)
	if code != 0 {
		t.Errorf("scanWorkspace(tmpDir) expected 0 issues, got %d", code)
	}
	out := ui.Output()
	if strings.Contains(out, "random.cpp") {
		t.Errorf("scanWorkspace should ignore non-category folders, but found random.cpp: %s", out)
	}
}
