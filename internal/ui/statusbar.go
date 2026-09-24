package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"leetcli/internal/config"
)

func RenderStatusBar(width int, cfg *config.Config) string {
	home, _ := os.UserHomeDir()
	baseDir := ""
	lang := "cpp"
	editor := "nvim"

	if cfg != nil {
		baseDir = cfg.BaseDir
		if home != "" && strings.HasPrefix(baseDir, home) {
			baseDir = "~" + strings.TrimPrefix(baseDir, home)
		}
		if cfg.DefaultLanguage != "" {
			lang = cfg.DefaultLanguage
		}
		if cfg.Editor != "" {
			editor = cfg.Editor
		}
	}
	if baseDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			baseDir = filepath.Base(cwd)
		}
	}

	left := StatusBarText.Render(fmt.Sprintf(" 📁 %s", baseDir))
	center := StatusBarText.Copy().Foreground(Cyan).Render(fmt.Sprintf("⚡ %s │ 📝 %s", lang, editor))
	right := StatusBarText.Copy().Foreground(DimColor).Render("[PgUp/PgDn: scroll] ")

	totalContent := lipgloss.Width(left) + lipgloss.Width(center) + lipgloss.Width(right)
	space := width - totalContent
	if space < 2 {
		space = 2
	}
	gap := strings.Repeat(" ", space/2)

	bar := lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		gap,
		center,
		gap,
		right,
	)

	return bar
}
