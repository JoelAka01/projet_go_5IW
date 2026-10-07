package style

import "github.com/charmbracelet/lipgloss"

var (
	Primary   = lipgloss.Color("63")
	Accent    = lipgloss.Color("212")
	Success   = lipgloss.Color("42")
	Danger    = lipgloss.Color("203")
	Muted     = lipgloss.Color("244")
	Highlight = lipgloss.Color("230")

	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("230")).
		Background(Primary).
		Padding(0, 1)

	Subtitle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	StatusOK = lipgloss.NewStyle().
			Foreground(Success).
			Bold(true)

	StatusErr = lipgloss.NewStyle().
			Foreground(Danger).
			Bold(true)

	HelpText = lipgloss.NewStyle().
			Foreground(Muted)

	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Primary).
		Padding(1, 2)

	SelectedItem = lipgloss.NewStyle().
			Foreground(Highlight).
			Background(Primary).
			Bold(true).
			Padding(0, 1)

	NormalItem = lipgloss.NewStyle().Padding(0, 1)
)

func Frame(appTitle, content, help string) string {
	return lipgloss.JoinVertical(
		lipgloss.Left,
		Title.Render(appTitle),
		"",
		content,
		"",
		HelpText.Render(help),
	)
}
