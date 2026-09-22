package commands

import (
	"strings"
	"testing"

	"leetcli/internal/config"
	"leetcli/internal/tracker"
)

func TestReviewQueueFSRS(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := config.Default()
	cfg.BaseDir = tmpDir

	prog := tracker.Load(tmpDir)
	prog.Upsert(tmpDir, "1", "Two Sum", "two-sum", "Easy", "array", nil, "solved", "1 ms", "1 MB")
	if err := prog.Save(tmpDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	ui := &fakeUI{}

	// Test review --due
	err := ReviewQueue([]string{"--due"}, cfg, ui)
	if err != nil {
		t.Fatalf("ReviewQueue --due failed: %v", err)
	}
	if !strings.Contains(ui.Output(), "Due for review") {
		t.Errorf("expected due review output, got: %s", ui.Output())
	}

	// Test review 1 with --grade easy
	ui = &fakeUI{}
	err = ReviewQueue([]string{"1", "--grade", "easy"}, cfg, ui)
	if err != nil {
		t.Fatalf("ReviewQueue 1 --grade easy failed: %v", err)
	}
	out := ui.Output()
	if !strings.Contains(out, "Easy") {
		t.Errorf("expected Easy in output, got: %s", out)
	}

	// Verify tracker state
	reloaded := tracker.Load(tmpDir)
	e := reloaded.Get("1")
	if e == nil {
		t.Fatal("problem 1 not found")
	}
	if e.LastRating != "Easy" {
		t.Errorf("LastRating = %q, want Easy", e.LastRating)
	}
	if e.Stability <= 0 {
		t.Errorf("Stability = %f, want > 0", e.Stability)
	}

	// Test review --list displays FSRS stats
	ui = &fakeUI{}
	err = ReviewQueue([]string{"--list"}, cfg, ui)
	if err != nil {
		t.Fatalf("ReviewQueue --list failed: %v", err)
	}
	listOut := ui.Output()
	if !strings.Contains(listOut, "FSRS Spaced Repetition") {
		t.Errorf("expected FSRS header, got: %s", listOut)
	}
	if !strings.Contains(listOut, "Recall:") {
		t.Errorf("expected Recall percentage in output, got: %s", listOut)
	}
}
