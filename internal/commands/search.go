package commands

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

func SearchProblems(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- Search Problems ---\n")

	baseDir := cfg.BaseDir
	dataStructures := cfg.GetDataStructures()

	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory not found: %s", baseDir)
		return fmt.Errorf("base directory not found: %s", baseDir)
	}

	query := ""
	if len(args) > 0 {
		query = strings.Join(args, " ")
	}
	if strings.TrimSpace(query) == "" {
		query = ui.PromptText("Enter problem name or number to search")
	}
	if strings.TrimSpace(query) == "" {
		ui.WriteOutput(MsgError, "Search query cannot be empty.")
		return fmt.Errorf("search query cannot be empty")
	}

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	allProblems := collectProblems(baseDir, dataStructures, exts)
	if len(allProblems) == 0 {
		ui.WriteOutput(MsgError, "No problems found in the base directory.")
		return fmt.Errorf("no problems found in the base directory")
	}

	results := searchProblems(allProblems, query)

	if len(results) == 0 {
		ui.WriteOutput(MsgError, "No problems found matching your search.")
		return fmt.Errorf("no problems found matching your search")
	}

	ui.WriteOutput(MsgInfo, "Found %d problem(s):\n", len(results))
	for _, r := range results {
		status := "unsolved"
		if r.Solutions > 0 {
			status = "solved"
		}
		line := fmt.Sprintf("  %-40s [%s] (%s)", r.FileName, status, r.Structure)
		ui.WriteOutput(MsgPlain, line)
	}
	return nil
}

func searchProblems(records []ProblemRecord, query string) []ProblemRecord {
	q := strings.ToLower(strings.TrimSpace(query))
	var results []ProblemRecord

	for _, r := range records {
		name := strings.ToLower(r.FileName)

		re := regexp.MustCompile(`(\d+)`)
		match := re.FindStringSubmatch(name)
		problemNumber := ""
		if len(match) > 1 {
			problemNumber = match[1]
		}

		if problemNumber != "" && strings.Contains(problemNumber, q) {
			results = append(results, r)
		} else if strings.Contains(name, q) {
			results = append(results, r)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		re := regexp.MustCompile(`(\d+)`)
		mi := re.FindStringSubmatch(results[i].FileName)
		mj := re.FindStringSubmatch(results[j].FileName)
		ni := 999999
		nj := 999999
		if len(mi) > 1 {
			fmt.Sscanf(mi[1], "%d", &ni)
		}
		if len(mj) > 1 {
			fmt.Sscanf(mj[1], "%d", &nj)
		}
		return ni < nj
	})

	return results
}
