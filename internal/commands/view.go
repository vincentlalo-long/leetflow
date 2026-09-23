package commands

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"golang.org/x/term"

	"leetcli/internal/api"
	"leetcli/internal/config"
	"leetcli/internal/template"
)

// ViewProblem renders the problem description directly in the terminal using Glamour.
func ViewProblem(args []string, cfg *config.Config, ui UI) error {
	pos, flags := parseFlags(args)

	problemNum := ""
	if len(pos) > 0 {
		problemNum = pos[0]
	}
	if problemNum == "" && hasFlag(flags, "num") {
		problemNum = flags["num"]
	}
	if problemNum == "" {
		problemNum = ui.PromptText("Enter problem number (or title slug) to view")
	}
	if problemNum == "" {
		return fmt.Errorf("problem number is required")
	}

	ui.WriteOutput(MsgInfo, "Searching for problem #%s...", problemNum)

	baseDir := cfg.BaseDir
	if baseDir != "" {
		baseDir = config.ExpandHome(baseDir)
	}

	var readmeContent string
	var problemTitle string
	var problemLink string
	foundLocal := false

	// 1. Try reading from local workspace
	if baseDir != "" {
		languages := template.NormalizeLanguages(nil)
		for k, v := range cfg.Languages {
			languages[k] = template.LanguageInfo{Label: v.Label, Ext: v.Ext}
		}
		var exts []string
		for _, info := range languages {
			exts = append(exts, info.Ext)
		}

		allFiles := template.GetAllSolutionFiles(baseDir, exts)
		for _, f := range allFiles {
			base := filepath.Base(f)
			if template.MatchesProblemNumber(base, problemNum) {
				localReadme := filepath.Join(filepath.Dir(f), "README.md")
				if data, err := os.ReadFile(localReadme); err == nil && len(data) > 0 {
					readmeContent = string(data)
					foundLocal = true
					ui.WriteOutput(MsgSuccess, "Loaded from local file: %s", localReadme)
					break
				}
			}
		}
	}

	// 2. Fall back to fetching from LeetCode API directly
	if !foundLocal {
		ui.WriteOutput(MsgInfo, "Not found locally. Fetching problem from LeetCode API...")
		slug := ""
		problemData, err := api.GetProblemByID(problemNum)
		if err == nil && problemData != nil {
			slug = problemData.Slug
			problemTitle = problemData.Title
		} else {
			slug = api.Slugify(problemNum)
		}

		details, err := api.GetProblemDetails(slug)
		if err != nil {
			ui.WriteOutput(MsgError, "Could not fetch problem details from LeetCode: %v", err)
			return fmt.Errorf("problem not found: %w", err)
		}

		cleanNum := details.QuestionFrontendID
		if cleanNum == "" {
			cleanNum = problemNum
		}
		if problemTitle == "" {
			problemTitle = details.Title
		}
		problemLink = fmt.Sprintf("https://leetcode.com/problems/%s/", slug)

		var tagNames []string
		for _, t := range details.TopicTags {
			tagNames = append(tagNames, t.Name)
		}
		tagsStr := strings.Join(tagNames, ", ")
		if tagsStr == "" {
			tagsStr = "None"
		}

		mdBody := template.FormatDescriptionMarkdown(details.Content)
		readmeContent = template.MakeReadmeContent(cleanNum, details.Title, problemLink, details.Difficulty, tagsStr, mdBody)
		ui.WriteOutput(MsgSuccess, "Fetched: %s (%s)", details.Title, details.Difficulty)
	}

	// Raw output option
	if hasFlag(flags, "raw") {
		ui.WriteString("\n" + readmeContent + "\n")
		return nil
	}

	// 3. Render markdown using Glamour
	termWidth := 100
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 30 {
		termWidth = w - 4
		if termWidth > 120 {
			termWidth = 120
		}
	}

	themeStyle := "dark"
	if cfg.Theme == "light" {
		themeStyle = "light"
	}

	renderer, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(themeStyle),
		glamour.WithWordWrap(termWidth),
	)
	if err != nil {
		renderer, _ = glamour.NewTermRenderer(glamour.WithAutoStyle(), glamour.WithWordWrap(termWidth))
	}

	rendered, err := renderer.Render(readmeContent)
	if err != nil {
		ui.WriteOutput(MsgError, "Markdown rendering failed: %v", err)
		ui.WriteString("\n" + readmeContent + "\n")
		return nil
	}

	// 4. Check for images and handle them
	imgURLs := template.ExtractImageURLs(readmeContent)
	if len(imgURLs) > 0 {
		var imgInfo strings.Builder
		imgInfo.WriteString("\n🖼️  Attached Diagram(s) / Image(s):\n")
		for i, url := range imgURLs {
			// OSC 8 terminal hyperlink format: \x1b]8;;URL\x1b\TEXT\x1b]8;;\x1b\
			hyperlink := fmt.Sprintf("\x1b]8;;%s\x1b\\%s\x1b]8;;\x1b\\", url, url)
			imgInfo.WriteString(fmt.Sprintf("   [%d] %s\n", i+1, hyperlink))
		}
		rendered += imgInfo.String()
	}

	// 5. Display rendered output
	// If in headless/CLI mode and stdout is a terminal, support pager for long descriptions
	if isHeadlessUI(ui) && term.IsTerminal(int(os.Stdout.Fd())) && !hasFlag(flags, "no-pager") {
		lines := strings.Count(rendered, "\n")
		_, termHeight, _ := term.GetSize(int(os.Stdout.Fd()))
		if termHeight > 0 && lines > termHeight {
			if tryPager(rendered) {
				// Display kitty images if in Kitty terminal
				displayKittyImages(imgURLs)
				return nil
			}
		}
	}

	ui.WriteString("\n" + rendered + "\n")
	displayKittyImages(imgURLs)
	return nil
}

func isHeadlessUI(ui UI) bool {
	_, ok := ui.(HeadlessUI)
	return ok
}

func tryPager(content string) bool {
	pager := os.Getenv("PAGER")
	if pager == "" {
		pager = "less -R"
	}
	parts := strings.Fields(pager)
	if len(parts) == 0 {
		return false
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Stdin = strings.NewReader(content)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

func isKittyTerminal() bool {
	term := strings.ToLower(os.Getenv("TERM"))
	return strings.Contains(term, "kitty") || os.Getenv("KITTY_PID") != "" || os.Getenv("KITTY_WINDOW_ID") != ""
}

func displayKittyImages(urls []string) {
	if len(urls) == 0 || !isKittyTerminal() {
		return
	}
	if _, err := exec.LookPath("kitty"); err != nil {
		return
	}
	for _, u := range urls {
		fmt.Printf("\n[Displaying Image: %s]\n", u)
		cmd := exec.Command("kitty", "+kitten", "icat", "--align", "left", u)
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.Discard
		_ = cmd.Run()
	}
}
