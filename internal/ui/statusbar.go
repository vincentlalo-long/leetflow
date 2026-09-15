package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func RenderStatusBar(width int) string {
	home, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	wd := strings.Replace(cwd, home, "~", 1)
	wd = filepath.Base(wd)

	mode := "no sandbox"
	model := "IDEAL-core (100%)"

	left := StatusBarText.Render(wd)
	center := StatusBarText.Copy().Foreground(Red).Render(mode)
	right := StatusBarText.Copy().Foreground(Magenta).Render(model)

	sep := DimmedStyle.Render(strings.Repeat("─", width))

	bar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		strings.Repeat(" ", max(0, width-lipgloss.Width(left)-lipgloss.Width(center)-lipgloss.Width(right)-4)),
		center,
		"  ",
		right,
	)

	return fmt.Sprintf("%s\n%s\n", sep, bar)
}
