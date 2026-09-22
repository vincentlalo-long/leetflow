package tracker

import (
	"path/filepath"
	"testing"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs"
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

func TestSortingNumeric(t *testing.T) {
	p := Load(t.TempDir())
	p.Upsert("", "10", "Regular Expression Matching", "regex", "Hard", "dp", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "2", "Add Two Numbers", "add-two-numbers", "Medium", "linked_list", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "20", "Valid Parentheses", "valid-parentheses", "Easy", "stack", nil, "solved", "1 ms", "1 MB")

	all := p.All()
	if len(all) != 4 {
		t.Fatalf("All len = %d, want 4", len(all))
	}
	expected := []string{"1", "2", "10", "20"}
	for i, want := range expected {
		if all[i].Number != want {
			t.Errorf("all[%d].Number = %q, want %q", i, all[i].Number, want)
		}
	}
}

func TestFSRSSchedulingRatings(t *testing.T) {
	p := Load(t.TempDir())
	p.Upsert("", "1", "Two Sum", "two-sum", "Easy", "array", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "2", "Add Two Numbers", "add-two-numbers", "Medium", "linked_list", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "3", "Longest Substring", "longest-substring", "Medium", "string", nil, "solved", "1 ms", "1 MB")
	p.Upsert("", "4", "Median of Two Sorted Arrays", "median", "Hard", "binary_search", nil, "solved", "1 ms", "1 MB")

	// Review 1 with Again -> 1 day
	e1, days1, ok1 := p.MarkReviewedFSRS("1", fsrs.Again)
	if !ok1 || days1 != 1 {
		t.Errorf("Again: days = %d, want 1", days1)
	}
	if e1.LastRating != "Again" {
		t.Errorf("e1.LastRating = %q, want Again", e1.LastRating)
	}

	// Review 2 with Hard -> 2 days
	e2, days2, ok2 := p.MarkReviewedFSRS("2", fsrs.Hard)
	if !ok2 || days2 != 2 {
		t.Errorf("Hard: days = %d, want 2", days2)
	}
	if e2.LastRating != "Hard" {
		t.Errorf("e2.LastRating = %q, want Hard", e2.LastRating)
	}

	// Review 3 with Good -> 4 days
	e3, days3, ok3 := p.MarkReviewedFSRS("3", fsrs.Good)
	if !ok3 || days3 != 4 {
		t.Errorf("Good: days = %d, want 4", days3)
	}
	if e3.LastRating != "Good" {
		t.Errorf("e3.LastRating = %q, want Good", e3.LastRating)
	}

	// Review 4 with Easy -> 7 days
	e4, days4, ok4 := p.MarkReviewedFSRS("4", fsrs.Easy)
	if !ok4 || days4 < 7 {
		t.Errorf("Easy: days = %d, want >= 7", days4)
	}
	if e4.LastRating != "Easy" {
		t.Errorf("e4.LastRating = %q, want Easy", e4.LastRating)
	}
	if e4.Stability <= 0 {
		t.Errorf("e4.Stability = %f, want > 0", e4.Stability)
	}
}

func TestFSRSRetrievability(t *testing.T) {
	now := time.Now()
	entry := &ProgressEntry{
		Number:       "1",
		Stability:    10.0,
		LastReviewed: now.Format("2006-01-02"),
	}

	// Retrievability right now should be ~100%
	rNow := entry.Retrievability(now)
	if rNow < 0.99 || rNow > 1.0 {
		t.Errorf("rNow = %f, want ~1.0", rNow)
	}

	// Retrievability 10 days later (1 stability interval) should be approximately 0.90 (RequestRetention)
	future := now.AddDate(0, 0, 10)
	rFuture := entry.Retrievability(future)
	if rFuture < 0.85 || rFuture > 0.95 {
		t.Errorf("rFuture after 1 stability period = %f, want ~0.90", rFuture)
	}

	// Retrievability 30 days later should be lower
	farFuture := now.AddDate(0, 0, 30)
	rFar := entry.Retrievability(farFuture)
	if rFar >= rFuture {
		t.Errorf("rFar = %f should be less than rFuture = %f", rFar, rFuture)
	}
}
