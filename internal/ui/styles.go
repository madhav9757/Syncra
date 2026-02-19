package ui

import "github.com/charmbracelet/lipgloss"

var (
	Primary   = lipgloss.Color("#7D56F4") // Vibrant Purple
	Secondary = lipgloss.Color("#00E5FF") // Electric Cyan
	Accent    = lipgloss.Color("#FF4081") // Hot Pink
	Success   = lipgloss.Color("#00C853") // Emerald Green
	Warning   = lipgloss.Color("#FFD600") // Neon Gold
	ErrorCol  = lipgloss.Color("#FF1744") // Radical Red
	Bg        = lipgloss.Color("#1A1B26") // Deep Tokyo Night Background
	Text      = lipgloss.Color("#C0CAF5") // Soft White/Blue Text
	Muted     = lipgloss.Color("#565F89") // Muted Blue Gray

	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			MarginLeft(2).
			MarginTop(1)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Italic(true).
			MarginLeft(2).
			MarginBottom(1)

	CardStyle = lipgloss.NewStyle().
			Padding(0, 2).
			Margin(1, 2).
			Width(64)

	StatusLabelStyle = lipgloss.NewStyle().
				Bold(true).
				MarginRight(2)

	InfoKeyStyle = lipgloss.NewStyle().
			Foreground(Secondary).
			Width(15)

	InfoValueStyle = lipgloss.NewStyle().
			Foreground(Text).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	ErrorTextStyle = lipgloss.NewStyle().
			Foreground(ErrorCol)

	FooterStyle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginLeft(3).
			MarginTop(1).
			Italic(true)
)

func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Success).Bold(true)
}
