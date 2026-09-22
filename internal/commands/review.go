package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	fsrs "github.com/open-spaced-repetition/go-fsrs"
	"leetcli/internal/config"
	"leetcli/internal/template"
	"leetcli/internal/tracker"
)

// ReviewQueue lists due problems for spaced-repetition review and lets the user
// mark them as reviewed. Supports:
//
//	review                -> interactive: list due, select to mark reviewed
//	review --list         -> show all tracked problems + due state
//	review --due          -> show due reviews only (non-interactive safe)
//	review <num>          -> mark a problem as reviewed
//	review --solve <num>  -> mark a problem solved
//	review --unsolve <num>-> mark a problem unsolved
func ReviewQueue(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Review Queue ---\n")

	baseDir := cfg.BaseDir
	if baseDir == "" {
		ui.WriteOutput(MsgError, "Missing 'base_dir' in config")
		return fmt.Errorf("missing 'base_dir' in config")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist: %s", baseDir)
		return fmt.Errorf("base directory does not exist: %s", baseDir)
	}

	pos, flags := parseFlags(args)
	prog := tracker.Load(baseDir)

	// Solve / unsolve a specific problem.
	if hasFlag(flags, "solve") || hasFlag(flags, "unsolve") {
		num := flags["solve"]
		if num == "" {
			num = flags["unsolve"]
		}
		if num == "" && len(pos) > 0 {
			num = pos[0]
		}
		solved := hasFlag(flags, "solve")
		if num == "" {
			ui.WriteOutput(MsgError, "Specify a problem number: review --solve 1")
			return fmt.Errorf("specify a problem number: review --solve 1")
		}
		title, difficulty := lookupProblemInfo(baseDir, num)
		prog.SetStatus(num, title, difficulty, nil, solved)
		if err := prog.Save(baseDir); err != nil {
			ui.WriteOutput(MsgError, "Failed to save progress: %v", err)
			return fmt.Errorf("failed to save progress: %w", err)
		}
		state := "unsolved"
		if solved {
			state = "solved"
		}
		ui.WriteOutput(MsgSuccess, "Problem %s marked as %s.", num, state)
		return nil
	}

	// Mark a specific problem as reviewed.
	if len(pos) > 0 {
		num := pos[0]
		var rating fsrs.Rating
		if g, ok := flags["grade"]; ok && g != "" {
			rating = parseGrade(g)
		} else if g, ok := flags["g"]; ok && g != "" {
			rating = parseGrade(g)
		} else {
			rating = promptRating(ui, num)
		}

		entry, days, ok := prog.MarkReviewedFSRS(num, rating)
		if !ok {
			ui.WriteOutput(MsgError, "Problem %s is not in the progress tracker. Mark it solved first (review --solve %s).", num, num)
			return fmt.Errorf("problem %s is not in the progress tracker", num)
		}
		if err := prog.Save(baseDir); err != nil {
			ui.WriteOutput(MsgError, "Failed to save progress: %v", err)
			return fmt.Errorf("failed to save progress: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Problem %s reviewed with grade [%s]!", num, rating.String())
		ui.WriteOutput(MsgInfo, "Next review in %d day(s) on %s (Stability: %.1fd).", days, entry.NextReview, entry.Stability)
		return nil
	}

	due := prog.DueReviews()

	if hasFlag(flags, "list") {
		entries := prog.All()
		if len(entries) == 0 {
			ui.WriteOutput(MsgInfo, "No tracked problems yet. Use 'submit' or 'review --solve' to add some.")
			return nil
		}
		now := time.Now()
		today := now.Format("2006-01-02")
		ui.WriteOutput(MsgPlain, "\nTracked problems (FSRS Spaced Repetition):")
		for _, e := range entries {
			dueMark := " "
			if e.Status == "solved" && (e.NextReview == "" || e.NextReview <= today) {
				dueMark = "●"
			}
			state := e.Status
			retention := int(e.Retrievability(now) * 100)
			ui.WriteOutput(MsgPlain, "  %s %s. %s (%s, %s, %d reviews)", dueMark, e.Number, e.Title, e.Difficulty, state, e.ReviewCount)
			if e.LastReviewed != "" || e.NextReview != "" {
				ui.WriteOutput(MsgPlain, "       Recall: %d%% | Next: %s | Stability: %.1fd | Last rating: %s", retention, orDash(e.NextReview), e.Stability, orDash(e.LastRating))
			}
		}
		return nil
	}

	if hasFlag(flags, "due") {
		if len(due) == 0 {
			ui.WriteOutput(MsgSuccess, "No reviews due. Great job! 🎉")
			return nil
		}
		ui.WriteOutput(MsgPlain, "\nDue for review (%d):", len(due))
		for _, e := range due {
			ui.WriteOutput(MsgPlain, "  %s. %s (%s)", e.Number, e.Title, e.Difficulty)
		}
		ui.WriteOutput(MsgInfo, "To mark reviewed: review <number> [--grade again|hard|good|easy]")
		return nil
	}

	// Interactive flow.
	if len(due) == 0 {
		ui.WriteOutput(MsgSuccess, "No reviews due right now. Come back later!")
		ui.WriteOutput(MsgInfo, "Use 'review --list' to see all tracked problems.")
		return nil
	}

	ui.WriteOutput(MsgPlain, "\nProblems due for review (%d):", len(due))
	for i, e := range due {
		ui.WriteOutput(MsgPlain, "  %d. %s. %s (%s)", i+1, e.Number, e.Title, e.Difficulty)
	}
	ui.WriteOutput(MsgPlain, "")

	sel := ui.PromptSelect("Select problem to mark reviewed", labelsFromEntries(due))
	if sel == "" {
		return nil
	}
	num := ""
	for _, e := range due {
		if fmt.Sprintf("%s. %s", e.Number, e.Title) == sel {
			num = e.Number
			break
		}
	}
	if num == "" {
		return nil
	}

	var rating fsrs.Rating
	if g, ok := flags["grade"]; ok && g != "" {
		rating = parseGrade(g)
	} else if g, ok := flags["g"]; ok && g != "" {
		rating = parseGrade(g)
	} else {
		rating = promptRating(ui, num)
	}

	entry, days, ok := prog.MarkReviewedFSRS(num, rating)
	if ok {
		if err := prog.Save(baseDir); err != nil {
			ui.WriteOutput(MsgError, "Failed to save progress: %v", err)
			return fmt.Errorf("failed to save progress: %w", err)
		}
		ui.WriteOutput(MsgSuccess, "Problem %s reviewed with grade [%s]!", num, rating.String())
		ui.WriteOutput(MsgInfo, "Next review in %d day(s) on %s (Stability: %.1fd).", days, entry.NextReview, entry.Stability)
	}
	return nil
}

func parseGrade(s string) fsrs.Rating {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "again", "a":
		return fsrs.Again
	case "2", "hard", "h":
		return fsrs.Hard
	case "3", "good", "g", "":
		return fsrs.Good
	case "4", "easy", "e":
		return fsrs.Easy
	default:
		return fsrs.Good
	}
}

func promptRating(ui UI, num string) fsrs.Rating {
	options := []string{
		"1. Again - Forgot logic / needed solution (Review in ~1d)",
		"2. Hard  - Solved with difficulty (Review in ~2d)",
		"3. Good  - Solved smoothly (Review in ~4d)",
		"4. Easy  - Trivial / Mastered (Review in ~7+d)",
	}
	choice := ui.PromptSelect(fmt.Sprintf("Rate your recall for #%s", num), options)
	if strings.HasPrefix(choice, "1") {
		return fsrs.Again
	} else if strings.HasPrefix(choice, "2") {
		return fsrs.Hard
	} else if strings.HasPrefix(choice, "4") {
		return fsrs.Easy
	}
	return fsrs.Good
}

func labelsFromEntries(entries []*tracker.ProgressEntry) []string {
	labels := make([]string, len(entries))
	for i, e := range entries {
		labels[i] = fmt.Sprintf("%s. %s", e.Number, e.Title)
	}
	return labels
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// lookupProblemInfo finds the title & difficulty of a problem from local files.
func lookupProblemInfo(baseDir, num string) (string, string) {
	languages := template.NormalizeLanguages(nil)
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}
	files := template.GetAllSolutionFiles(baseDir, exts)
	for _, f := range files {
		if template.MatchesProblemNumber(filepath.Base(f), num) {
			dir := filepath.Dir(f)
			diff := readDifficultyFromReadme(dir)
			title := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), num+"_"), filepath.Ext(f))
			return title, diff
		}
	}
	return "", ""
}
