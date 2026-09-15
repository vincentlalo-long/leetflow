package ui

import "github.com/charmbracelet/lipgloss"

var (
	Subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	Highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7C56DC"}
	Special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	Cyan      = lipgloss.Color("#00D9FF")
	Magenta   = lipgloss.Color("#FF79C6")
	White     = lipgloss.Color("#FFFFFF")
	Gray      = lipgloss.Color("#808080")
	DimColor  = lipgloss.Color("#585858")
	Red       = lipgloss.Color("#FF5555")
	Green     = lipgloss.Color("#50FA7B")
	Yellow    = lipgloss.Color("#F1FA8C")

	BannerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Cyan)

	PromptStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Cyan).
			MarginRight(1)

	CommandStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Highlight)

	ErrorStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Red)

	SuccessStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Green)

	InfoStyle = lipgloss.NewStyle().
			Foreground(Cyan)

	DimmedStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(DimColor).
			Background(lipgloss.Color("#1a1a2e"))

	StatusBarText = lipgloss.NewStyle().
			Foreground(Gray).
			Background(lipgloss.Color("#1a1a2e"))

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	AppStyle = lipgloss.NewStyle().
			Padding(0, 1, 0, 0)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true)
)
