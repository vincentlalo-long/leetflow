package tracker

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"
)

// ProgressEntry tracks the solving status and review state of one problem.
type ProgressEntry struct {
	Number       string   `json:"number"`
	Title        string   `json:"title"`
	Slug         string   `json:"slug"`
	Difficulty   string   `json:"difficulty"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags,omitempty"`
	Status       string   `json:"status"` // unsolved | solved
	SolvedDate   string   `json:"solved_date,omitempty"`
	LastAccepted string   `json:"last_accepted,omitempty"`
	LastRuntime  string   `json:"last_runtime,omitempty"`
	LastMemory   string   `json:"last_memory,omitempty"`
	BestRuntime  string   `json:"best_runtime,omitempty"`
	BestMemory   string   `json:"best_memory,omitempty"`
	Submissions  int      `json:"submissions"`
	AcceptedCount int     `json:"accepted_count"`
	ReviewCount  int      `json:"review_count"`
	LastReviewed string   `json:"last_reviewed,omitempty"`
	NextReview   string   `json:"next_review,omitempty"`
}

// Progress is the on-disk state, stored in <base_dir>/.leet/progress.json.
type Progress struct {
	Problems map[string]*ProgressEntry `json:"problems"`
}

const fileName = "progress.json"

func dir(baseDir string) string {
	return filepath.Join(baseDir, ".leet")
}

func Path(baseDir string) string {
	return filepath.Join(dir(baseDir), fileName)
}

func Load(baseDir string) *Progress {
	p := &Progress{Problems: map[string]*ProgressEntry{}}
	data, err := os.ReadFile(Path(baseDir))
	if err != nil {
		return p
	}
	json.Unmarshal(data, p)
	if p.Problems == nil {
		p.Problems = map[string]*ProgressEntry{}
	}
	return p
}

func (p *Progress) Save(baseDir string) error {
	if err := os.MkdirAll(dir(baseDir), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(Path(baseDir), data, 0644)
}

func (p *Progress) Get(number string) *ProgressEntry {
	return p.Problems[number]
}

// Upsert records or updates a problem entry from a submission result.
func (p *Progress) Upsert(baseDir, number, title, slug, difficulty, category string, tags []string, status, runtime, memory string) {
	e := p.Problems[number]
	if e == nil {
		e = &ProgressEntry{Number: number}
		p.Problems[number] = e
	}
	now := time.Now().Format("2006-01-02")
	e.Title = title
	e.Slug = slug
	e.Difficulty = difficulty
	e.Category = category
	if len(tags) > 0 {
		e.Tags = tags
	}
	e.Submissions++
	if status == "solved" {
		e.Status = "solved"
		e.AcceptedCount++
		e.SolvedDate = now
		e.LastAccepted = now
		e.LastRuntime = runtime
		e.LastMemory = memory
		if e.BestRuntime == "" || numericLess(runtime, e.BestRuntime) {
			e.BestRuntime = runtime
		}
		if e.BestMemory == "" || numericLess(memory, e.BestMemory) {
			e.BestMemory = memory
		}
		if e.NextReview == "" {
			e.NextReview = now
		}
	} else {
		e.LastRuntime = runtime
	}
}

var numRe = regexp.MustCompile(`(\d+(?:\.\d+)?)`)

// numericLess compares "12 ms" vs "5 ms" numerically, treating missing/zero specially.
func numericLess(a, b string) bool {
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	ma := numRe.FindString(a)
	mb := numRe.FindString(b)
	if ma == "" || mb == "" {
		return a < b
	}
	na, errA := strconv.ParseFloat(ma, 64)
	nb, errB := strconv.ParseFloat(mb, 64)
	if errA != nil || errB != nil {
		return a < b
	}
	return na < nb
}

// SetStatus manually marks a problem solved/unsolved (e.g. review --solve / --unsolve).
func (p *Progress) SetStatus(number, title, difficulty string, tags []string, solved bool) {
	e := p.Problems[number]
	if e == nil {
		e = &ProgressEntry{Number: number}
		p.Problems[number] = e
	}
	e.Title = title
	e.Difficulty = difficulty
	if len(tags) > 0 {
		e.Tags = tags
	}
	if solved {
		e.Status = "solved"
		if e.SolvedDate == "" {
			e.SolvedDate = time.Now().Format("2006-01-02")
		}
	} else {
		e.Status = "unsolved"
	}
}

// MarkReviewed advances the spaced-repetition review schedule.
func (p *Progress) MarkReviewed(number string) bool {
	e := p.Problems[number]
	if e == nil {
		return false
	}
	e.ReviewCount++
	now := time.Now()
	e.LastReviewed = now.Format("2006-01-02")
	// intervals: 1, 3, 7, 15, 30, 60 days
	intervals := []int{1, 3, 7, 15, 30, 60}
	idx := e.ReviewCount - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(intervals) {
		idx = len(intervals) - 1
	}
	e.NextReview = now.AddDate(0, 0, intervals[idx]).Format("2006-01-02")
	return true
}

// DueReviews returns solved problems whose next review date is today or earlier.
func (p *Progress) DueReviews() []*ProgressEntry {
	today := time.Now().Format("2006-01-02")
	var due []*ProgressEntry
	for _, e := range p.Problems {
		if e.Status != "solved" {
			continue
		}
		if e.NextReview == "" || e.NextReview <= today {
			due = append(due, e)
		}
	}
	sort.Slice(due, func(i, j int) bool {
		if due[i].Number != due[j].Number {
			return due[i].Number < due[j].Number
		}
		return due[i].Title < due[j].Title
	})
	return due
}

// All returns all entries sorted by number.
func (p *Progress) All() []*ProgressEntry {
	out := make([]*ProgressEntry, 0, len(p.Problems))
	for _, e := range p.Problems {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Number != out[j].Number {
			return out[i].Number < out[j].Number
		}
		return out[i].Title < out[j].Title
	})
	return out
}
