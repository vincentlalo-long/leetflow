package tracker

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs"
)

// ProgressEntry tracks the solving status and review state of one problem.
type ProgressEntry struct {
	Number        string   `json:"number"`
	Title         string   `json:"title"`
	Slug          string   `json:"slug"`
	Difficulty    string   `json:"difficulty"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags,omitempty"`
	Status        string   `json:"status"` // unsolved | solved
	SolvedDate    string   `json:"solved_date,omitempty"`
	LastAccepted  string   `json:"last_accepted,omitempty"`
	LastRuntime   string   `json:"last_runtime,omitempty"`
	LastMemory    string   `json:"last_memory,omitempty"`
	BestRuntime   string   `json:"best_runtime,omitempty"`
	BestMemory    string   `json:"best_memory,omitempty"`
	Submissions   int      `json:"submissions"`
	AcceptedCount int      `json:"accepted_count"`
	ReviewCount   int      `json:"review_count"`
	LastReviewed  string   `json:"last_reviewed,omitempty"`
	NextReview    string   `json:"next_review,omitempty"`

	// FSRS fields
	Stability      float64 `json:"stability,omitempty"`
	DifficultyFSRS float64 `json:"fsrs_difficulty,omitempty"`
	ElapsedDays    uint64  `json:"elapsed_days,omitempty"`
	ScheduledDays  uint64  `json:"scheduled_days,omitempty"`
	Reps           uint64  `json:"reps,omitempty"`
	Lapses         uint64  `json:"lapses,omitempty"`
	State          int8    `json:"state,omitempty"`
	LastRating     string  `json:"last_rating,omitempty"`
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

// Spaced-repetition scheduling is powered by the Free Spaced Repetition Scheduler (FSRS)
// algorithm (https://github.com/open-spaced-repetition/fsrs4anki) via the
// go-fsrs library (https://github.com/open-spaced-repetition/go-fsrs).

// ToCard converts ProgressEntry into an fsrs.Card for scheduling.
func (e *ProgressEntry) ToCard() fsrs.Card {
	c := fsrs.NewCard()
	if e.Stability > 0 {
		c.Stability = e.Stability
		c.Difficulty = e.DifficultyFSRS
		c.ElapsedDays = e.ElapsedDays
		c.ScheduledDays = e.ScheduledDays
		c.Reps = e.Reps
		c.Lapses = e.Lapses
		c.State = fsrs.State(e.State)
		if e.NextReview != "" {
			if t, err := time.Parse("2006-01-02", e.NextReview); err == nil {
				c.Due = t
			}
		}
		if e.LastReviewed != "" {
			if t, err := time.Parse("2006-01-02", e.LastReviewed); err == nil {
				c.LastReview = t
			}
		}
	}
	return c
}

// ApplyCard updates ProgressEntry with the updated fsrs.Card state.
func (e *ProgressEntry) ApplyCard(c fsrs.Card) {
	e.Stability = c.Stability
	e.DifficultyFSRS = c.Difficulty
	e.ElapsedDays = c.ElapsedDays
	e.ScheduledDays = c.ScheduledDays
	e.Reps = c.Reps
	e.Lapses = c.Lapses
	e.State = int8(c.State)
	e.NextReview = c.Due.Format("2006-01-02")
	e.LastReviewed = c.LastReview.Format("2006-01-02")
}

// Retrievability calculates the estimated retention rate (0.0 to 1.0) on a given date.
func (e *ProgressEntry) Retrievability(now time.Time) float64 {
	if e.Stability <= 0 || e.LastReviewed == "" {
		return 1.0
	}
	last, err := time.Parse("2006-01-02", e.LastReviewed)
	if err != nil {
		return 1.0
	}
	elapsedDays := now.Sub(last).Hours() / 24.0
	if elapsedDays <= 0 {
		return 1.0
	}
	decay := -0.5
	factor := math.Pow(0.9, 1.0/decay) - 1.0
	r := math.Pow(1.0+factor*elapsedDays/e.Stability, decay)
	if r < 0.0 {
		return 0.0
	}
	if r > 1.0 {
		return 1.0
	}
	return r
}

// MarkReviewedFSRS advances the review schedule using the FSRS algorithm.
// Returns the updated entry, the number of days until the next review, and success boolean.
func (p *Progress) MarkReviewedFSRS(number string, rating fsrs.Rating) (*ProgressEntry, int, bool) {
	e := p.Problems[number]
	if e == nil {
		return nil, 0, false
	}

	now := time.Now()
	e.ReviewCount++
	card := e.ToCard()
	param := fsrs.DefaultParam()
	schedules := param.Repeat(card, now)

	info, ok := schedules[rating]
	if !ok {
		info = schedules[fsrs.Good]
	}

	nextCard := info.Card
	var daysUntilReview int

	switch nextCard.State {
	case fsrs.Learning, fsrs.Relearning:
		switch rating {
		case fsrs.Again:
			daysUntilReview = 1
		case fsrs.Hard:
			daysUntilReview = 2
		case fsrs.Good:
			daysUntilReview = 4
		case fsrs.Easy:
			daysUntilReview = 7
		default:
			daysUntilReview = 3
		}
		nextCard.ScheduledDays = uint64(daysUntilReview)
		nextCard.Due = now.AddDate(0, 0, daysUntilReview)
	default: // fsrs.Review
		daysUntilReview = int(nextCard.ScheduledDays)
		if daysUntilReview < 1 {
			daysUntilReview = 1
		}
		nextCard.Due = now.AddDate(0, 0, daysUntilReview)
	}

	e.ApplyCard(nextCard)
	e.LastRating = rating.String()
	return e, daysUntilReview, true
}

// MarkReviewed advances the spaced-repetition review schedule with Good rating default.
func (p *Progress) MarkReviewed(number string) bool {
	_, _, ok := p.MarkReviewedFSRS(number, fsrs.Good)
	return ok
}

func compareProblemNumbers(a, b string) bool {
	na, errA := strconv.Atoi(a)
	nb, errB := strconv.Atoi(b)
	if errA == nil && errB == nil {
		return na < nb
	}
	return a < b
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
			return compareProblemNumbers(due[i].Number, due[j].Number)
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
			return compareProblemNumbers(out[i].Number, out[j].Number)
		}
		return out[i].Title < out[j].Title
	})
	return out
}
