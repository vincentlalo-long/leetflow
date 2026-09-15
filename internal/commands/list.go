package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"leetcli/internal/config"
	"leetcli/internal/template"
)

type ProblemRecord struct {
	FilePath   string
	FileName   string
	Structure  string
	Solutions  int
	Difficulty string
	Tags       []string
}

func ListProblems(args []string, cfg *config.Config, ui UI) error {
	ui.WriteOutput(MsgPlain, "--- List Problems ---\n")

	pos, flags := parseFlags(args)

	baseDir := cfg.BaseDir
	dataStructures := cfg.GetDataStructures()

	if baseDir == "" {
		ui.WriteOutput(MsgError, "Missing 'base_dir' in config")
		return fmt.Errorf("missing 'base_dir' in config")
	}
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		ui.WriteOutput(MsgError, "Base directory does not exist: %s", baseDir)
		return fmt.Errorf("base directory does not exist: %s", baseDir)
	}

	languages := template.NormalizeLanguages(nil)
	for k, v := range cfg.Languages {
		languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
	}
	var exts []string
	for _, info := range languages {
		exts = append(exts, info.Ext)
	}

	records := collectProblems(baseDir, dataStructures, exts)
	if len(records) == 0 {
		ui.WriteOutput(MsgInfo, "No problem files found")
		return nil
	}

	// Non-interactive flags: leet list --ds array --difficulty Easy --unsolved --tag "Hash Table"
	filterFlagged := false
	selectedStructure := ""
	difficulty := ""
	unsolvedOnly := false
	tagFilter := ""
	if hasFlag(flags, "ds") {
		filterFlagged = true
		selectedStructure = flags["ds"]
	}
	if strings.EqualFold(selectedStructure, "all") {
		selectedStructure = ""
	}
	if hasFlag(flags, "difficulty") || hasFlag(flags, "diff") {
		filterFlagged = true
		d := flags["difficulty"]
		if d == "" {
			d = flags["diff"]
		}
		difficulty = strings.Title(strings.ToLower(d))
	}
	if hasFlag(flags, "unsolved") || hasFlag(flags, "u") {
		filterFlagged = true
		unsolvedOnly = true
	}
	if hasFlag(flags, "tag") {
		filterFlagged = true
		tagFilter = flags["tag"]
	}

	if !filterFlagged && len(pos) == 0 {
		structSet := make(map[string]bool)
		for _, r := range records {
			structSet[r.Structure] = true
		}
		availableStructs := make([]string, 0, len(structSet))
		for s := range structSet {
			availableStructs = append(availableStructs, s)
		}
		sort.Strings(availableStructs)

		filterChoices := append([]string{"All"}, availableStructs...)
		selectedStructure = ui.PromptSelect("Filter by data structure", filterChoices)
		unsolvedOnly = ui.PromptConfirm("Show only unsolved problems?")
	}

	if !filterFlagged && len(pos) > 0 && pos[0] != "all" && pos[0] != "" {
		if isDifficulty(pos[0]) {
			difficulty = strings.Title(strings.ToLower(pos[0]))
		} else {
			selectedStructure = pos[0]
		}
	}

	if selectedStructure == "All" {
		selectedStructure = ""
	}

	filtered := filterProblems(records, selectedStructure, difficulty, unsolvedOnly, tagFilter)

	ui.WriteOutput(MsgPlain, "Overview")
	ui.WriteOutput(MsgInfo, "Total problems: %d", len(records))
	ui.WriteOutput(MsgInfo, "After filters: %d", len(filtered))
	if selectedStructure != "" {
		ui.WriteOutput(MsgInfo, "Structure: %s", selectedStructure)
	}
	if difficulty != "" {
		ui.WriteOutput(MsgInfo, "Difficulty: %s", difficulty)
	}
	if tagFilter != "" {
		ui.WriteOutput(MsgInfo, "Tag: %s", tagFilter)
	}
	if unsolvedOnly {
		ui.WriteOutput(MsgInfo, "Mode: Unsolved only")
	} else {
		ui.WriteOutput(MsgInfo, "Mode: All")
	}

	if len(filtered) == 0 {
		ui.WriteOutput(MsgInfo, "No problems match selected filters")
		return nil
	}

	ui.WriteOutput(MsgPlain, "\nProblems:")
	for i, r := range filtered {
		status := "unsolved"
		if r.Solutions > 0 {
			status = fmt.Sprintf("%d solution(s)", r.Solutions)
		}
		diff := r.Difficulty
		if diff == "" {
			diff = "?"
		}
		tagsStr := ""
		if len(r.Tags) > 0 {
			tagsStr = fmt.Sprintf(" [%s]", strings.Join(r.Tags, ", "))
		}
		ui.WriteOutput(MsgPlain, "  %d. %s (%s, %s, %s)%s", i+1, r.FileName, r.Structure, diff, status, tagsStr)
		ui.WriteOutput(MsgPlain, "     %s", r.FilePath)
	}
	return nil
}

func isDifficulty(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "easy", "medium", "hard":
		return true
	}
	return false
}

func collectProblems(baseDir string, dataStructures map[string]string, exts []string) []ProblemRecord {
	files := template.GetAllSolutionFiles(baseDir, exts)
	records := make([]ProblemRecord, 0, len(files))

	for _, f := range files {
		solutions := 0
		content, err := os.ReadFile(f)
		if err == nil {
			solutions = template.CountSolutions(string(content))
		}

		records = append(records, ProblemRecord{
			FilePath:   f,
			FileName:   filepath.Base(f),
			Structure:  template.DetectStructure(f, dataStructures),
			Solutions:  solutions,
			Difficulty: readDifficultyFromReadme(filepath.Dir(f)),
			Tags:       readTagsFromReadme(filepath.Dir(f)),
		})
	}

	sort.Slice(records, func(i, j int) bool {
		re := regexp.MustCompile(`(\d+)`)
		mi := re.FindStringSubmatch(records[i].FileName)
		mj := re.FindStringSubmatch(records[j].FileName)
		ni := 999999
		nj := 999999
		if len(mi) > 1 {
			fmt.Sscanf(mi[1], "%d", &ni)
		}
		if len(mj) > 1 {
			fmt.Sscanf(mj[1], "%d", &nj)
		}
		if ni != nj {
			return ni < nj
		}
		return strings.ToLower(records[i].FileName) < strings.ToLower(records[j].FileName)
	})

	return records
}

func readDifficultyFromReadme(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`(?i)-\s*\*\*Difficulty:\*\*\s*(.+)`)
	m := re.FindStringSubmatch(string(data))
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func readTagsFromReadme(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		return nil
	}
	re := regexp.MustCompile(`(?i)-\s*\*\*Tags:\*\*\s*(.+)`)
	m := re.FindStringSubmatch(string(data))
	if len(m) < 2 {
		return nil
	}
	raw := strings.TrimSpace(m[1])
	if raw == "" || raw == "None" {
		return nil
	}
	parts := strings.Split(raw, ",")
	var tags []string
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

func filterProblems(records []ProblemRecord, selectedStructure, difficulty string, unsolvedOnly bool, tagFilter string) []ProblemRecord {
	var filtered []ProblemRecord
	for _, r := range records {
		if selectedStructure != "" && r.Structure != selectedStructure {
			continue
		}
		if difficulty != "" && !strings.EqualFold(r.Difficulty, difficulty) {
			continue
		}
		if unsolvedOnly && r.Solutions > 0 {
			continue
		}
		if tagFilter != "" && !hasTag(r.Tags, tagFilter) {
			continue
		}
		filtered = append(filtered, r)
	}
	return filtered
}

func hasTag(tags []string, filter string) bool {
	filterLower := strings.ToLower(filter)
	for _, t := range tags {
		if strings.ToLower(t) == filterLower {
			return true
		}
	}
	return false
}
