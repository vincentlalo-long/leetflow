package ui

import (
	"fmt"
	"regexp"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"leetcli/internal/commands"
	"leetcli/internal/config"
)

var (
	suggestedRe = regexp.MustCompile(`(?i)\(suggested:\s*([^)]+)\)`)
	quoteRe     = regexp.MustCompile(`(?i)for '([^']+)'`)
)

func extractSuggestion(label string) string {
	m := suggestedRe.FindStringSubmatch(label)
	if len(m) == 2 {
		return strings.TrimSpace(m[1])
	}
	m2 := quoteRe.FindStringSubmatch(label)
	if len(m2) == 2 {
		return strings.TrimSpace(m2[1])
	}
	return ""
}

type PromptKind int

const (
	PromptNone PromptKind = iota
	PromptText
	PromptSelect
	PromptConfirm
)

type SelectItem struct {
	Index int
	Label string
}

type sharedState struct {
	mu           sync.Mutex
	promptKind   PromptKind
	promptLabel  string
	promptResult chan string
	allItems     []SelectItem
	selectItems  []SelectItem
	selectIdx    int
	filterQuery  string
	output       strings.Builder
	redrawCh     chan struct{}
}

func (s *sharedState) notifyRedraw() {
	if s.redrawCh != nil {
		select {
		case s.redrawCh <- struct{}{}:
		default:
		}
	}
}

func (s *sharedState) applyFilter(query string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.filterQuery = query
	if query == "" {
		s.selectItems = make([]SelectItem, len(s.allItems))
		copy(s.selectItems, s.allItems)
		s.selectIdx = 0
		return
	}

	lowerQuery := strings.ToLower(query)
	var filtered []SelectItem
	for _, item := range s.allItems {
		if strings.Contains(strings.ToLower(item.Label), lowerQuery) {
			filtered = append(filtered, item)
		}
	}
	s.selectItems = filtered
	s.selectIdx = 0
}

type RedrawMsg struct{}

func waitForRedraw(ch chan struct{}) tea.Cmd {
	return func() tea.Msg {
		if ch == nil {
			return nil
		}
		<-ch
		return RedrawMsg{}
	}
}

type Model struct {
	ready        bool
	input        textinput.Model
	history      []string
	historyPos   int
	scrollOffset int // 0 = at bottom, >0 = scrolled up by N lines
	termWidth    int
	termHeight   int

	state      *sharedState
	cfg        *config.Config
	showBanner bool
}

func New(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Placeholder = "Type a command... (Use PgUp/PgDn to scroll output)"
	ti.Prompt = "> "
	ti.PromptStyle = PromptStyle
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(Cyan)
	ti.CharLimit = 512
	ti.Width = 80

	ti.Focus()

	state := &sharedState{
		redrawCh: make(chan struct{}, 100),
	}

	return Model{
		input:        ti,
		history:      []string{},
		cfg:          cfg,
		showBanner:   true,
		scrollOffset: 0,
		state:        state,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, waitForRedraw(m.state.redrawCh))
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RedrawMsg:
		return m, waitForRedraw(m.state.redrawCh)

	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.ready = true
		m.input.Width = m.termWidth - 4

	case tea.KeyMsg:
		// Check global scroll keys (PgUp, PgDn) first
		switch msg.String() {
		case "pgup":
			m.scrollOffset += 5
			return m, nil
		case "pgdown":
			if m.scrollOffset > 5 {
				m.scrollOffset -= 5
			} else {
				m.scrollOffset = 0
			}
			return m, nil
		}

		m.state.mu.Lock()
		pk := m.state.promptKind
		pr := m.state.promptResult
		m.state.mu.Unlock()

		switch pk {
		case PromptText:
			return m.updateTextPrompt(msg, pr)
		case PromptSelect:
			return m.updateSelectPrompt(msg, pr)
		case PromptConfirm:
			return m.updateConfirmPrompt(msg, pr)
		default:
			return m.updateCommandInput(msg)
		}
	}

	return m, nil
}

func (m Model) updateTextPrompt(msg tea.KeyMsg, resultCh chan string) (tea.Model, tea.Cmd) {
	m.state.mu.Lock()
	label := m.state.promptLabel
	m.state.mu.Unlock()

	sugg := extractSuggestion(label)

	switch msg.String() {
	case "tab", "right":
		if m.input.Value() == "" && sugg != "" {
			m.input.SetValue(sugg)
			m.input.SetCursor(len(sugg))
			return m, nil
		}
	case "enter":
		val := m.input.Value()
		if val == "" && sugg != "" {
			val = sugg
		}
		m.input.SetValue("")
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.mu.Unlock()
		resultCh <- val
	case "ctrl+c", "esc":
		// Cancel active prompt cleanly without quitting TUI
		m.input.SetValue("")
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.output.WriteString(DimmedStyle.Render("(Cancelled prompt)") + "\n")
		m.state.mu.Unlock()
		select {
		case resultCh <- "":
		default:
		}
		return m, nil
	case "ctrl+d":
		return m, tea.Quit
	default:
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) updateSelectPrompt(msg tea.KeyMsg, resultCh chan string) (tea.Model, tea.Cmd) {
	m.state.mu.Lock()
	selItems := m.state.selectItems
	selIdx := m.state.selectIdx
	filterQuery := m.state.filterQuery
	m.state.mu.Unlock()

	keyStr := msg.String()

	switch keyStr {
	case "up":
		if selIdx > 0 {
			selIdx--
			m.state.mu.Lock()
			m.state.selectIdx = selIdx
			m.state.mu.Unlock()
		}
	case "down":
		if selIdx < len(selItems)-1 {
			selIdx++
			m.state.mu.Lock()
			m.state.selectIdx = selIdx
			m.state.mu.Unlock()
		}
	case "enter":
		val := ""
		if len(selItems) > 0 && selIdx < len(selItems) {
			val = selItems[selIdx].Label
		}
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.filterQuery = ""
		m.state.mu.Unlock()
		resultCh <- val

	case "backspace":
		if len(filterQuery) > 0 {
			filterQuery = filterQuery[:len(filterQuery)-1]
			m.state.applyFilter(filterQuery)
		}

	case "ctrl+c":
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.filterQuery = ""
		m.state.output.WriteString(DimmedStyle.Render("(Cancelled prompt)") + "\n")
		m.state.mu.Unlock()
		select {
		case resultCh <- "":
		default:
		}
		return m, nil

	case "esc":
		if filterQuery != "" {
			m.state.applyFilter("")
		} else {
			m.scrollOffset = 0
			m.state.mu.Lock()
			m.state.promptKind = PromptNone
			m.state.filterQuery = ""
			m.state.output.WriteString(DimmedStyle.Render("(Cancelled prompt)") + "\n")
			m.state.mu.Unlock()
			select {
			case resultCh <- "":
			default:
			}
			return m, nil
		}

	case "ctrl+d":
		return m, tea.Quit

	default:
		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] <= 126 {
			filterQuery += keyStr
			m.state.applyFilter(filterQuery)
		}
	}
	return m, nil
}

func (m Model) updateConfirmPrompt(msg tea.KeyMsg, resultCh chan string) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.mu.Unlock()
		resultCh <- "yes"
	case "n", "N":
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.mu.Unlock()
		resultCh <- "no"
	case "enter":
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.mu.Unlock()
		resultCh <- "yes"
	case "ctrl+c", "esc":
		m.scrollOffset = 0
		m.state.mu.Lock()
		m.state.promptKind = PromptNone
		m.state.output.WriteString(DimmedStyle.Render("(Cancelled prompt)") + "\n")
		m.state.mu.Unlock()
		select {
		case resultCh <- "no":
		default:
		}
		return m, nil
	case "ctrl+d":
		return m, tea.Quit
	}
	return m, nil
}

func getCommandSuggestions(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}
	parts := strings.Fields(input)
	if len(parts) > 1 {
		return nil
	}
	prefix := strings.ToLower(parts[0])
	var matches []string
	for _, cmd := range allCommands {
		if strings.HasPrefix(cmd, prefix) && cmd != prefix {
			matches = append(matches, cmd)
		}
	}
	return matches
}

func (m Model) updateCommandInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "ctrl+d":
		return m, tea.Quit

	case "enter":
		cmd := strings.TrimSpace(m.input.Value())
		m.input.SetValue("")
		m.scrollOffset = 0
		if cmd != "" {
			m.history = append(m.history, cmd)
			m.historyPos = len(m.history)
			m.state.mu.Lock()
			m.state.output.WriteString(fmt.Sprintf("\n> %s\n", cmd))
			m.state.mu.Unlock()
			return m, m.dispatchCommand(cmd)
		}
		return m, nil

	case "up":
		if len(m.history) > 0 && m.historyPos > 0 {
			m.historyPos--
			m.input.SetValue(m.history[m.historyPos])
		}
		return m, nil

	case "down":
		if m.historyPos < len(m.history)-1 {
			m.historyPos++
			m.input.SetValue(m.history[m.historyPos])
		} else {
			m.historyPos = len(m.history)
			m.input.SetValue("")
		}
		return m, nil

	case "tab", "right":
		suggs := getCommandSuggestions(m.input.Value())
		if len(suggs) > 0 {
			m.input.SetValue(suggs[0] + " ")
			m.input.SetCursor(len(m.input.Value()))
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) dispatchCommand(cmd string) tea.Cmd {
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return nil
	}

	switch parts[0] {
	case "exit", "quit", "q":
		m.state.mu.Lock()
		m.state.output.WriteString("Goodbye!\n")
		m.state.mu.Unlock()
		m.cfg.Save()
		return tea.Quit

	case "help", "/help", "man":
		if len(parts) > 1 {
			m.state.mu.Lock()
			m.state.output.WriteString(RenderCommandManual(parts[1]))
			m.state.mu.Unlock()
			return nil
		}
		go m.interactiveHelp()
		return nil

	case "clear", "cls":
		m.state.mu.Lock()
		m.state.output.Reset()
		m.state.mu.Unlock()
		m.scrollOffset = 0
		return nil

	default:
		if handler, ok := commands.Registry[parts[0]]; ok {
			go handler(parts[1:], m.cfg, m.state)
			return nil
		}
		m.state.mu.Lock()
		m.state.output.WriteString(fmt.Sprintf("Unknown command: '%s'. Type 'help' to see available commands.\n\n", parts[0]))
		m.state.mu.Unlock()
		return nil
	}
}

func (m *Model) interactiveHelp() {
	ui := m.state

	orderedCmds := []string{
		"add", "add-sol", "list", "search", "manage-structures",
		"stats", "theme", "daily", "random", "hint", "similar",
		"open", "web", "run", "test", "submit", "timer", "note", "review",
		"sync", "clean", "profile", "contest", "config", "clear", "exit",
	}

	var choices []string
	var cmdKeys []string

	for _, c := range orderedCmds {
		if doc, ok := commands.CommandDocs[c]; ok {
			label := fmt.Sprintf("%-18s - %s", c, doc.Summary)
			choices = append(choices, label)
			cmdKeys = append(cmdKeys, c)
		}
	}

	selected := ui.PromptSelect("Select a command to view manual (or press Esc/Ctrl+C to cancel):", choices)
	if selected == "" {
		return
	}

	selectedCmd := ""
	for i, label := range choices {
		if label == selected {
			selectedCmd = cmdKeys[i]
			break
		}
	}

	if selectedCmd != "" {
		ui.WriteString(RenderCommandManual(selectedCmd))
		ui.PromptText("Press Enter to return to main prompt")
	}
}

var allCommands = []string{
	"add", "add-sol", "list", "search", "manage-structures",
	"stats", "theme", "daily", "random", "hint", "similar",
	"open", "web", "browser", "run", "test", "submit", "timer",
	"note", "review", "clean", "sync", "profile", "contest", "config", "cfg",
	"help", "clear", "exit", "quit",
}

func (m Model) View() string {
	if !m.ready {
		return "\nInitializing..."
	}

	m.state.mu.Lock()
	outputStr := m.state.output.String()
	pk := m.state.promptKind
	selItems := make([]SelectItem, len(m.state.selectItems))
	copy(selItems, m.state.selectItems)
	selIdx := m.state.selectIdx
	label := m.state.promptLabel
	filterQuery := m.state.filterQuery
	m.state.mu.Unlock()

	var buf strings.Builder

	if m.showBanner {
		buf.WriteString(RenderBanner())
	}

	buf.WriteString(SeparatorStyle.Render(strings.Repeat("─", m.termWidth)))
	buf.WriteString("\n")

	lines := strings.Split(outputStr, "\n")
	availableHeight := m.termHeight - 10
	if m.showBanner {
		availableHeight -= 8
	}
	if availableHeight < 3 {
		availableHeight = 3
	}

	maxScroll := len(lines) - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}

	currentScroll := m.scrollOffset
	if currentScroll > maxScroll {
		currentScroll = maxScroll
	}

	end := len(lines) - currentScroll
	if end < availableHeight {
		end = availableHeight
	}
	if end > len(lines) {
		end = len(lines)
	}

	start := end - availableHeight
	if start < 0 {
		start = 0
	}

	for _, line := range lines[start:end] {
		if len(line) > m.termWidth-2 {
			buf.WriteString(line[:m.termWidth-2])
		} else {
			buf.WriteString(line)
		}
		buf.WriteString("\n")
	}

	buf.WriteString("\n")
	buf.WriteString(RenderStatusBar(m.termWidth))
	buf.WriteString(DimmedStyle.Render(strings.Repeat("─", m.termWidth)))
	buf.WriteString("\n")

	switch pk {
	case PromptNone:
		buf.WriteString(m.input.View())
		suggs := getCommandSuggestions(m.input.Value())
		if len(suggs) > 0 {
			buf.WriteString("\n  " + DimmedStyle.Render("💡 Suggestions: "))
			for i, s := range suggs {
				if i > 0 {
					buf.WriteString("  ")
				}
				if i == 0 {
					buf.WriteString(CommandStyle.Render(s))
				} else {
					buf.WriteString(DimmedStyle.Render(s))
				}
			}
			buf.WriteString(" " + HelpStyle.Render("(Press Tab or → to complete)"))
		}
	case PromptText:
		sugg := extractSuggestion(label)
		hint := ""
		if m.input.Value() == "" && sugg != "" {
			hint = " " + HelpStyle.Render("(Tab to accept: "+sugg+")")
		}
		buf.WriteString(fmt.Sprintf("%s%s ", PromptStyle.Render(label), hint))
		buf.WriteString(m.input.View())
	case PromptSelect:
		if filterQuery != "" {
			buf.WriteString(fmt.Sprintf("%s  %s\n", PromptStyle.Render(label), InfoStyle.Render(fmt.Sprintf("🔍 Filter: %q (%d matches)", filterQuery, len(selItems)))))
		} else {
			buf.WriteString(fmt.Sprintf("%s\n", PromptStyle.Render(label)))
		}

		if len(selItems) == 0 {
			buf.WriteString("  " + ErrorStyle.Render("No items match filter '"+filterQuery+"'") + "\n")
		} else {
			maxVisible := 8
			if m.termHeight > 15 {
				maxVisible = m.termHeight - 12
				if maxVisible > 10 {
					maxVisible = 10
				}
			}
			if maxVisible < 3 {
				maxVisible = 3
			}

			winStart := 0
			if selIdx >= maxVisible {
				winStart = selIdx - maxVisible/2
			}
			winEnd := winStart + maxVisible
			if winEnd > len(selItems) {
				winEnd = len(selItems)
				winStart = winEnd - maxVisible
				if winStart < 0 {
					winStart = 0
				}
			}

			if winStart > 0 {
				buf.WriteString(DimmedStyle.Render(fmt.Sprintf("  ▲ ... %d items above", winStart)) + "\n")
			}

			for i := winStart; i < winEnd; i++ {
				item := selItems[i]
				prefix := "  "
				style := DimmedStyle
				if i == selIdx {
					prefix = fmt.Sprintf("> %d. ", i+1)
					style = CommandStyle
				}
				buf.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(item.Label)))
			}

			if winEnd < len(selItems) {
				buf.WriteString(DimmedStyle.Render(fmt.Sprintf("  ▼ ... %d items below", len(selItems)-winEnd)) + "\n")
			}
		}
		buf.WriteString(HelpStyle.Render("Type to filter | ↑↓ to navigate | Enter to select | Esc to clear/cancel"))
	case PromptConfirm:
		buf.WriteString(fmt.Sprintf("%s %s", PromptStyle.Render(label), HelpStyle.Render("(y/n - Esc/Ctrl+C to cancel)")))
	}

	return AppStyle.Render(buf.String())
}

// sharedState implements commands.UI interface.
// These methods are called from command handler goroutines.

func (s *sharedState) PromptText(label string) string {
	ch := make(chan string, 1)
	s.mu.Lock()
	s.promptKind = PromptText
	s.promptLabel = label
	s.promptResult = ch
	s.mu.Unlock()
	s.notifyRedraw()

	return <-ch
}

func (s *sharedState) PromptSelect(label string, choices []string) string {
	ch := make(chan string, 1)
	s.mu.Lock()
	s.promptKind = PromptSelect
	s.promptLabel = label
	s.promptResult = ch
	s.allItems = make([]SelectItem, len(choices))
	s.selectItems = make([]SelectItem, len(choices))
	for i, c := range choices {
		item := SelectItem{Index: i, Label: c}
		s.allItems[i] = item
		s.selectItems[i] = item
	}
	s.selectIdx = 0
	s.filterQuery = ""
	s.mu.Unlock()
	s.notifyRedraw()

	return <-ch
}

func (s *sharedState) PromptConfirm(label string) bool {
	ch := make(chan string, 1)
	s.mu.Lock()
	s.promptKind = PromptConfirm
	s.promptLabel = label
	s.promptResult = ch
	s.mu.Unlock()
	s.notifyRedraw()

	return <-ch == "yes"
}

func (s *sharedState) WriteOutput(kind commands.MsgKind, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	s.mu.Lock()
	switch kind {
	case commands.MsgError:
		s.output.WriteString(ErrorStyle.Render("✘ ") + ErrorStyle.Render(msg) + "\n")
	case commands.MsgSuccess:
		s.output.WriteString(SuccessStyle.Render("✔ ") + SuccessStyle.Render(msg) + "\n")
	case commands.MsgInfo:
		s.output.WriteString(InfoStyle.Render("ℹ ") + msg + "\n")
	default:
		s.output.WriteString(msg + "\n")
	}
	s.mu.Unlock()
	s.notifyRedraw()
}

func (s *sharedState) WriteString(str string) {
	s.mu.Lock()
	s.output.WriteString(str)
	s.mu.Unlock()
	s.notifyRedraw()
}

func (s *sharedState) Writef(format string, args ...interface{}) {
	s.mu.Lock()
	s.output.WriteString(fmt.Sprintf(format, args...))
	s.mu.Unlock()
	s.notifyRedraw()
}
