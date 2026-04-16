package tui

import "github.com/charmbracelet/lipgloss"

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39")).
			Background(lipgloss.Color("235")).
			Padding(0, 1)

	filterBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236"))

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("245")).
				Background(lipgloss.Color("234"))

	selectedRowStyle = lipgloss.NewStyle().
				Bold(true).
				Background(lipgloss.Color("237")).
				Foreground(lipgloss.Color("255"))

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("236")).
			Padding(0, 1)
)

func rowStyle(status string) lipgloss.Style {
	base := lipgloss.NewStyle()
	switch status {
	case "success":
		return base.Foreground(lipgloss.Color("2"))
	case "failure":
		return base.Foreground(lipgloss.Color("1"))
	case "in_progress":
		return base.Foreground(lipgloss.Color("3"))
	default:
		return base.Foreground(lipgloss.Color("245"))
	}
}
