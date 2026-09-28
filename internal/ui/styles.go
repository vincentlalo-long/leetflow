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
			Foreground(DimColor)

	StatusBarText = lipgloss.NewStyle().
			Foreground(Gray)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(DimColor)

	AppStyle = lipgloss.NewStyle().
			Padding(0, 1, 0, 0)

	HelpStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Italic(true)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#101018")).
			Background(Cyan).
			Padding(0, 1)

	BadgeStyle = lipgloss.NewStyle().
			Foreground(Magenta).
			Bold(true)

	SectionTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(Cyan)

	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DimColor).
			Padding(1, 1)

	SidebarItemStyle = lipgloss.NewStyle().
				Foreground(Gray).
				Padding(0, 1)

	SidebarActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(White).
				Background(Highlight).
				Padding(0, 1)

	StatCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(DimColor).
			Foreground(Gray).
			Padding(0, 1)

	StatValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Special)

	ActionStyle = lipgloss.NewStyle().
			Foreground(White).
			Padding(0, 1)

	ActionActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(White).
				Background(lipgloss.Color("#30304d")).
				Padding(0, 1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(Gray).
			Background(lipgloss.Color("#1d1d2a")).
			Padding(0, 1)
)
