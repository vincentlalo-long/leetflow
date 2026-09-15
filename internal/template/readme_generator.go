package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"leetcli/internal/config"
)

type ProblemEntry struct {
	Num          int
	NumStr       string
	Title        string
	Link         string
	Difficulty   string
	Category     string
	Tags         []string
	DirRelPath   string
	ReadmeRel    string
	SolutionRels []string
}

// GenerateRootReadme scans baseDir for all problem directories, extracts metadata,
// generates a dynamic index markdown, and writes it to baseDir/README.md.
func GenerateRootReadme(baseDir string, cfg *config.Config) (string, int, error) {
	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		absBase = baseDir
	}

	languages := NormalizeLanguages(nil)
	if cfg != nil && cfg.Languages != nil {
		for k, v := range cfg.Languages {
			languages[k] = LanguageInfo{Label: v.Label, Ext: v.Ext}
		}
	}
	extSet := make(map[string]bool)
	for _, info := range languages {
		extSet["."+strings.ToLower(info.Ext)] = true
	}

	dsMap := make(map[string]string)
	if cfg != nil {
		dsMap = cfg.GetDataStructures()
	}

	entriesMap := make(map[string]*ProblemEntry)

	err = filepath.Walk(absBase, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path != absBase && IsIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		dirPath := filepath.Dir(path)
		if dirPath == absBase {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		fileName := filepath.Base(path)

		isReadme := fileName == "README.md"
		isSolution := extSet[ext]

		if !isReadme && !isSolution {
			return nil
		}

		entry, ok := entriesMap[dirPath]
		if !ok {
			entry = parseProblemFolder(absBase, dirPath, dsMap)
			entriesMap[dirPath] = entry
		}

		relPath, _ := filepath.Rel(absBase, path)
		relPathSlash := filepath.ToSlash(relPath)

		if isReadme {
			entry.ReadmeRel = relPathSlash
		} else if isSolution {
			dup := false
			for _, s := range entry.SolutionRels {
				if s == relPathSlash {
					dup = true
					break
				}
			}
			if !dup {
				entry.SolutionRels = append(entry.SolutionRels, relPathSlash)
			}
		}

		return nil
	})

	if err != nil {
		return "", 0, err
	}

	var entries []*ProblemEntry
	for _, entry := range entriesMap {
		if entry.NumStr == "" && entry.Title == "" && len(entry.SolutionRels) == 0 && entry.ReadmeRel == "" {
			continue
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Num != entries[j].Num {
			return entries[i].Num < entries[j].Num
		}
		return strings.ToLower(entries[i].Title) < strings.ToLower(entries[j].Title)
	})

	content := buildReadmeMarkdown(entries)

	readmePath := filepath.Join(absBase, "README.md")

	// Skip the write when nothing changed. This avoids pointless git churn
	// (and mtime updates) when `leet sync`/`readme` is re-run without any real
	// workspace changes.
	if old, err := os.ReadFile(readmePath); err == nil && string(old) == content {
		return readmePath, len(entries), nil
	}

	err = os.WriteFile(readmePath, []byte(content), 0644)
	if err != nil {
		return "", 0, err
	}

	return readmePath, len(entries), nil
}

var titleRe = regexp.MustCompile(`(?m)^#\s*\[(\d+)\.\s*([^\]]+)\](?:\(([^)]+)\))?`)
var diffRe = regexp.MustCompile(`(?i)-\s*\*\*Difficulty:\*\*\s*(.+)`)
var tagsRe = regexp.MustCompile(`(?i)-\s*\*\*Tags:\*\*\s*(.+)`)

func parseProblemFolder(baseDir, dirPath string, dsMap map[string]string) *ProblemEntry {
	entry := &ProblemEntry{
		DirRelPath: filepath.ToSlash(dirPath),
	}

	relDir, _ := filepath.Rel(baseDir, dirPath)
	parts := strings.Split(filepath.ToSlash(relDir), "/")
	if len(parts) > 1 {
		folderCategory := parts[0]
		entry.Category = formatCategoryName(folderCategory, dsMap)
	} else {
		entry.Category = "Uncategorized"
	}

	readmePath := filepath.Join(dirPath, "README.md")
	if data, err := os.ReadFile(readmePath); err == nil {
		text := string(data)
		mTitle := titleRe.FindStringSubmatch(text)
		if len(mTitle) >= 3 {
			entry.NumStr = mTitle[1]
			entry.Num, _ = strconv.Atoi(mTitle[1])
			entry.Title = strings.TrimSpace(mTitle[2])
			if len(mTitle) >= 4 {
				entry.Link = strings.TrimSpace(mTitle[3])
			}
		}

		mDiff := diffRe.FindStringSubmatch(text)
		if len(mDiff) >= 2 {
			entry.Difficulty = strings.TrimSpace(mDiff[1])
		}

		mTags := tagsRe.FindStringSubmatch(text)
		if len(mTags) >= 2 {
			raw := strings.TrimSpace(mTags[1])
			if raw != "" && raw != "None" {
				for _, t := range strings.Split(raw, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						entry.Tags = append(entry.Tags, t)
					}
				}
			}
		}
	}

	if entry.NumStr == "" || entry.Title == "" {
		folderName := filepath.Base(dirPath)
		numRe := regexp.MustCompile(`^(\d+)-(.*)$`)
		if m := numRe.FindStringSubmatch(folderName); len(m) == 3 {
			if entry.NumStr == "" {
				entry.NumStr = m[1]
				entry.Num, _ = strconv.Atoi(m[1])
			}
			if entry.Title == "" {
				entry.Title = titleCase(strings.ReplaceAll(m[2], "-", " "))
			}
		}
	}

	if entry.Title == "" {
		entry.Title = filepath.Base(dirPath)
	}

	if entry.Link == "" && entry.Title != "" {
		slug := strings.ToLower(strings.ReplaceAll(entry.Title, " ", "-"))
		entry.Link = fmt.Sprintf("https://leetcode.com/problems/%s/", slug)
	}

	return entry
}

func formatCategoryName(folder string, dsMap map[string]string) string {
	folderLower := strings.ToLower(folder)
	for name, f := range dsMap {
		if strings.ToLower(f) == folderLower {
			return titleCase(name)
		}
	}
	return titleCase(strings.ReplaceAll(folder, "_", " "))
}

func titleCase(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func formatDifficultyBadge(diff string) string {
	switch strings.ToLower(strings.TrimSpace(diff)) {
	case "easy":
		return "🟢 Easy"
	case "medium":
		return "🟡 Medium"
	case "hard":
		return "🔴 Hard"
	default:
		if diff != "" {
			return diff
		}
		return "⚪ Unknown"
	}
}

func encodeMarkdownPath(relPath string) string {
	return strings.ReplaceAll(relPath, " ", "%20")
}

func buildReadmeMarkdown(entries []*ProblemEntry) string {
	easyCount, medCount, hardCount := 0, 0, 0
	for _, e := range entries {
		switch strings.ToLower(strings.TrimSpace(e.Difficulty)) {
		case "easy":
			easyCount++
		case "medium":
			medCount++
		case "hard":
			hardCount++
		}
	}

	now := time.Now().Format("2006-01-02")

	var sb strings.Builder
	sb.WriteString("# 🧩 LeetCode Solutions Index\n\n")
	sb.WriteString("> Automated index of solved LeetCode problems generated by **[Go CLI (`leet`)](go-cli/)**.\n\n")

	sb.WriteString("## 📊 Workspace Summary\n\n")
	sb.WriteString(fmt.Sprintf("- **Total Problems:** %d\n", len(entries)))
	sb.WriteString(fmt.Sprintf("- **Difficulty Breakdown:** 🟢 **Easy:** %d | 🟡 **Medium:** %d | 🔴 **Hard:** %d\n",
		easyCount, medCount, hardCount))
	sb.WriteString(fmt.Sprintf("- **Last Synced:** %s\n\n", now))

	sb.WriteString("---\n\n")
	sb.WriteString("## 📚 Problem List\n\n")
	sb.WriteString("| # | Problem Title | Category | Difficulty | Tags | Solution File | Details & Notes |\n")
	sb.WriteString("|---|---------------|----------|------------|------|---------------|-----------------|\n")

	for _, e := range entries {
		numDisplay := e.NumStr
		if numDisplay == "" {
			numDisplay = "-"
		}

		titleCell := e.Title
		if e.Link != "" {
			titleCell = fmt.Sprintf("[%s](%s)", e.Title, e.Link)
		}

		diffCell := formatDifficultyBadge(e.Difficulty)
		catCell := e.Category
		if catCell == "" {
			catCell = "Uncategorized"
		}

		solCell := "-"
		if len(e.SolutionRels) > 0 {
			sort.Strings(e.SolutionRels)
			var solLinks []string
			for _, sol := range e.SolutionRels {
				filename := filepath.Base(sol)
				target := "./" + encodeMarkdownPath(sol)
				solLinks = append(solLinks, fmt.Sprintf("[`%s`](%s)", filename, target))
			}
			solCell = strings.Join(solLinks, ", ")
		}

		readmeCell := "-"
		if e.ReadmeRel != "" {
			target := "./" + encodeMarkdownPath(e.ReadmeRel)
			readmeCell = fmt.Sprintf("[`README.md`](%s)", target)
		}

		tagsCell := "-"
		if len(e.Tags) > 0 {
			tagsCell = strings.Join(e.Tags, ", ")
		}

		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s | %s |\n",
			numDisplay, titleCell, catCell, diffCell, tagsCell, solCell, readmeCell))
	}

	sb.WriteString("\n---\n*Automated index created with ❤️ using `leet sync` or `leet readme`.*\n")

	return sb.String()
}
