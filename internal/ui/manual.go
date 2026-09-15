package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"leetcli/internal/commands"
)

func RenderCommandManual(cmdName string) string {
	doc, ok := commands.CommandDocs[cmdName]
	if !ok {
		return ErrorStyle.Render(fmt.Sprintf("No detailed manual found for command '%s'.", cmdName)) + "\n"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(Cyan)
	labelStyle := lipgloss.NewStyle().Bold(true).Foreground(Yellow)
	valStyle := lipgloss.NewStyle().Foreground(White)
	codeStyle := lipgloss.NewStyle().Foreground(Green)

	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString(titleStyle.Render(fmt.Sprintf("📖 MANUAL: %s", strings.ToUpper(doc.Name))))
	sb.WriteString("\n")
	sb.WriteString(DimmedStyle.Render(strings.Repeat("─", 50)))
	sb.WriteString("\n")
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Summary:    "), valStyle.Render(doc.Summary)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Usage:      "), codeStyle.Render(doc.Usage)))
	sb.WriteString(fmt.Sprintf("%s %s\n", labelStyle.Render("Description:"), valStyle.Render(doc.Description)))
	if len(doc.Examples) > 0 {
		sb.WriteString(labelStyle.Render("Examples:   \n"))
		for _, ex := range doc.Examples {
			sb.WriteString(fmt.Sprintf("  • %s\n", codeStyle.Render(ex)))
		}
	}
	sb.WriteString(DimmedStyle.Render(strings.Repeat("─", 50)))
	sb.WriteString("\n")
	return sb.String()
}
