package template

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var ignoredDirs = map[string]bool{
	".git":         true,
	".github":      true,
	"internal":     true,
	"dist":         true,
	"extension":    true,
	"node_modules": true,
	"scratch":      true,
	".vscode":      true,
	".idea":        true,
	"bin":          true,
}

func IsIgnoredDir(name string) bool {
	return ignoredDirs[strings.ToLower(name)]
}

func GetAllSolutionFiles(baseDir string, extensions []string) []string {
	extSet := make(map[string]bool)
	for _, ext := range extensions {
		e := strings.TrimLeft(ext, ".")
		e = strings.ToLower(e)
		extSet[e] = true
	}

	var files []string
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if path != baseDir && IsIgnoredDir(info.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.TrimLeft(filepath.Ext(path), ".")
		if extSet[strings.ToLower(ext)] {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

var winIllegalFileRe = regexp.MustCompile(`[\\/:*?"<>|\x00-\x1f]+`)

// SanitizeFileName makes a problem name safe to use as a file name: it strips
// Windows-illegal characters and path separators (blocking directory
// traversal), trims leading/trailing dots and spaces, and collapses whitespace
// to underscores. Returns "" when nothing safe remains.
func SanitizeFileName(name string) string {
	s := winIllegalFileRe.ReplaceAllString(name, "_")
	s = strings.Trim(s, " .")
	s = strings.ReplaceAll(s, " ", "_")
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "._")
}

func CreateProblemDirectory(baseDir, dsFolder, folderName string) (string, error) {
	dir := filepath.Join(baseDir, dsFolder, folderName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

var solutionRe = regexp.MustCompile(`(?i)\bSolution\s+\d+`)

func CountSolutions(content string) int {
	matches := solutionRe.FindAllString(content, -1)
	return len(matches)
}

func DetectStructure(filePath string, dataStructures map[string]string) string {
	normalized := strings.ToLower(filepath.ToSlash(filePath))

	for name, folder := range dataStructures {
		token := strings.ToLower("/" + folder + "/")
		if strings.Contains(normalized, token) {
			return name
		}
	}
	return "unmatched"
}

func MatchesProblemNumber(filename, problemNum string) bool {
	if strings.HasPrefix(filename, problemNum+"_") {
		return true
	}
	cleanProblemNum := strings.TrimLeft(problemNum, "0")
	if cleanProblemNum == "" {
		cleanProblemNum = "0"
	}
	cleanFilename := strings.TrimLeft(filename, "0")
	if cleanFilename == "" {
		cleanFilename = filename
	}
	if strings.HasPrefix(cleanFilename, cleanProblemNum+"_") {
		return true
	}
	return false
}

func InferSlugFromPath(filePath, content string) string {
	parent := filepath.Base(filepath.Dir(filePath))
	re := regexp.MustCompile(`^\d+-(.+)$`)
	matches := re.FindStringSubmatch(parent)
	if len(matches) == 2 {
		return matches[1]
	}

	linkRe := regexp.MustCompile(`Link:\s*https?://leetcode\.com/problems/([^/]+)/`)
	linkMatches := linkRe.FindStringSubmatch(content)
	if len(linkMatches) == 2 {
		return linkMatches[1]
	}

	return ""
}

func MakeReadmeContent(cleanNum, title, link, difficulty, tagsStr, mdContent string) string {
	return fmt.Sprintf("# [%s. %s](%s)\n\n- **Difficulty:** %s\n- **Tags:** %s\n\n## Description\n\n%s",
		cleanNum, title, link, difficulty, tagsStr, strings.TrimSpace(mdContent))
}

func StripHTMLTags(html string) string {
	re := regexp.MustCompile(`<[^>]+>`)
	return strings.TrimSpace(re.ReplaceAllString(html, ""))
}

// DecodeHTMLEntities replaces common HTML entities with their literal text so
// problem descriptions read correctly as plain markdown (e.g. `&lt;=` -> `<=`).
func DecodeHTMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}

var (
	// blockTagRe matches block-level tags that should start a new line.
	blockTagRe = regexp.MustCompile(`(?i)</?(?:li|p|pre|ul|ol|div|h[1-6]|br)\s*/?>`)
	// supRe / subRe keep superscript/subscript readable, e.g. 10<sup>4</sup> -> 10^4.
	supRe = regexp.MustCompile(`(?i)<sup>(.*?)</sup>`)
	subRe = regexp.MustCompile(`(?i)<sub>(.*?)</sub>`)
	// sectionStartRe breaks lines before known section markers when they appear
	// mid-line, so fetched descriptions render as clean, line-separated markdown.
	sectionStartRe = regexp.MustCompile(`(?i)([^\n])((?:Example\s+\d+|Input|Output|Explanation|Constraints|Follow-up|Follow up)\s*:)`)
	imgTagRe       = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	srcAttrRe      = regexp.MustCompile(`(?i)src=["']([^"']+)["']`)
	altAttrRe      = regexp.MustCompile(`(?i)alt=["']([^"']*)["']`)
	mdImageRe      = regexp.MustCompile(`!\[([^\]]*)\]\((https?://[^)]+)\)`)
)

func replaceImgTags(html string) string {
	return imgTagRe.ReplaceAllStringFunc(html, func(m string) string {
		srcMatch := srcAttrRe.FindStringSubmatch(m)
		if len(srcMatch) < 2 || srcMatch[1] == "" {
			return ""
		}
		src := srcMatch[1]
		if strings.HasPrefix(src, "//") {
			src = "https:" + src
		} else if strings.HasPrefix(src, "/") {
			src = "https://leetcode.com" + src
		}
		alt := "image"
		altMatch := altAttrRe.FindStringSubmatch(m)
		if len(altMatch) >= 2 && strings.TrimSpace(altMatch[1]) != "" {
			alt = strings.TrimSpace(altMatch[1])
		}
		return fmt.Sprintf("\n\n![%s](%s)\n\n", alt, src)
	})
}

// ExtractImageURLs extracts all HTTP(S) image URLs from a markdown text.
func ExtractImageURLs(md string) []string {
	matches := mdImageRe.FindAllStringSubmatch(md, -1)
	var urls []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) >= 3 && !seen[m[2]] {
			urls = append(urls, m[2])
			seen[m[2]] = true
		}
	}
	return urls
}

// breakSectionLines inserts a newline before known section markers (Input:,
// Output:, Explanation:, ...) when they appear mid-line.
func breakSectionLines(s string) string {
	return sectionStartRe.ReplaceAllString(s, "$1\n$2")
}

// blockTagToNewline maps a matched block-level tag to a newline. List items
// become consecutive lines (<li> opens, </li> closes with no blank separator).
func blockTagToNewline(m string) string {
	lm := strings.ToLower(m)
	if strings.HasPrefix(lm, "</li>") {
		return ""
	}
	return "\n"
}

// FormatDescriptionMarkdown converts a raw LeetCode description (HTML) into
// clean markdown for a problem README: block tags become new lines, HTML
// entities are decoded, superscripts stay readable (10^4), and
// Input/Output/Explanation/Constraints each land on their own line.
func FormatDescriptionMarkdown(html string) string {
	// Preserve <img> tags as markdown before other tags are stripped.
	s := replaceImgTags(html)
	// Block-level tags -> newlines, so <li>/<p>/<pre> items are not glued
	// together on one line.
	s = blockTagRe.ReplaceAllStringFunc(s, blockTagToNewline)
	// Superscript / subscript -> ^N / _N before tags are stripped.
	s = supRe.ReplaceAllString(s, "^$1")
	s = subRe.ReplaceAllString(s, "_$1")
	// Strip the remaining inline tags.
	s = stripTagsRe.ReplaceAllString(s, "")
	s = DecodeHTMLEntities(s)
	s = breakSectionLines(s)
	// Drop standalone &nbsp; spacer lines (now blank after decoding) and
	// collapse runs of blank lines so paragraphs stay readable.
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	blank := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blank++
			if blank > 1 {
				continue
			}
			out = append(out, "")
			continue
		}
		blank = 0
		out = append(out, strings.TrimSpace(l))
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func GetSolvedMap(baseDir string, extensions []string) map[string]bool {
	solved := make(map[string]bool)
	files := GetAllSolutionFiles(baseDir, extensions)

	reNum := regexp.MustCompile(`^(\d+)`)
	for _, f := range files {
		base := filepath.Base(f)
		m := reNum.FindStringSubmatch(base)
		if len(m) == 2 {
			num := m[1]
			solved[num] = true
			clean := strings.TrimLeft(num, "0")
			if clean != "" {
				solved[clean] = true
			}
		}

		contentBytes, err := os.ReadFile(f)
		if err == nil {
			slug := InferSlugFromPath(f, string(contentBytes))
			if slug != "" {
				solved[slug] = true
			}
		}
	}
	return solved
}
