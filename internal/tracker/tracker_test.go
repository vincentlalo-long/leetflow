package tracker

import (
	"path/filepath"
	"testing"
)

func TestUpsertAndStatus(t *testing.T) {
	baseDir := t.TempDir()

	p := Load(baseDir)
	if len(p.Problems) != 0 {
		t.Fatalf("new tracker should have no problems")
	}

	p.Upsert(baseDir, "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "solved", "8 ms", "10 MB")
	if err := p.Save(baseDir); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	reloaded := Load(baseDir)
	e := reloaded.Get("1")
	if e == nil {
		t.Fatal("problem 1 missing after reload")
	}
	if e.Status != "solved" {
		t.Errorf("Status = %q, want solved", e.Status)
	}
	if e.Submissions != 1 {
		t.Errorf("Submissions = %d, want 1", e.Submissions)
	}
	if e.AcceptedCount != 1 {
		t.Errorf("AcceptedCount = %d, want 1", e.AcceptedCount)
	}
	if e.BestRuntime != "8 ms" {
		t.Errorf("BestRuntime = %q, want 8 ms", e.BestRuntime)
	}
	if e.SolvedDate == "" {
		t.Errorf("SolvedDate should be set")
	}
}

func TestUpsertTracksBestRuntime(t *testing.T) {
	p := Load(t.TempDir())

	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "solved", "12 ms", "12 MB")
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "solved", "5 ms", "8 MB")
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "unsolved", "20 ms", "20 MB")

	e := p.Get("1")
	if e.BestRuntime != "5 ms" {
		t.Errorf("BestRuntime = %q, want 5 ms", e.BestRuntime)
	}
	if e.Submissions != 3 {
		t.Errorf("Submissions = %d, want 3", e.Submissions)
	}
	if e.AcceptedCount != 2 {
		t.Errorf("AcceptedCount = %d, want 2", e.AcceptedCount)
	}
}

func TestMarkReviewedSchedulesNext(t *testing.T) {
	p := Load(t.TempDir())
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "solved", "5 ms", "8 MB")

	if !p.MarkReviewed("1") {
		t.Fatal("MarkReviewed(1) should succeed")
	}
	if p.Get("1").ReviewCount != 1 {
		t.Errorf("ReviewCount = %d, want 1", p.Get("1").ReviewCount)
	}
	if p.Get("1").NextReview == "" {
		t.Errorf("NextReview should be scheduled")
	}
	// reviewing a nonexistent problem fails
	if p.MarkReviewed("999") {
		t.Errorf("MarkReviewed(999) should fail")
	}
}

func TestDueReviews(t *testing.T) {
	p := Load(t.TempDir())

	// solved with no next_review -> due immediately
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", []string{"Array", "Hash Table"}, "solved", "5 ms", "8 MB")
	// reviewed -> scheduled in future, not due
	p.Upsert("", "2", "Add Two", "add-two", "Medium", "string", []string{"Array", "Linked List"}, "solved", "5 ms", "8 MB")
	p.MarkReviewed("2")

	due := p.DueReviews()
	if len(due) != 1 {
		t.Fatalf("DueReviews len = %d, want 1", len(due))
	}
	if due[0].Number != "1" {
		t.Errorf("due[0] = %q, want 1", due[0].Number)
	}
}

func TestProgressPath(t *testing.T) {
	got := Path(filepath.Join("C:", "repo"))
	if got != filepath.Join("C:", "repo", ".leet", "progress.json") {
		t.Errorf("Path = %q", got)
	}
}
