package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Ultra-Vibrant Neon Palette
	Primary   = lipgloss.Color("#FF00FF") // Neon Magenta
	Secondary = lipgloss.Color("#00FFFF") // Electric Cyan
	Accent    = lipgloss.Color("#FFFF00") // Laser Yellow
	Success   = lipgloss.Color("#39FF14") // Neon Green
	Warning   = lipgloss.Color("#FFAD00") // Pure Orange
	ErrorCol  = lipgloss.Color("#FF3131") // Neon Red
	Bg        = lipgloss.Color("#050505") // Near Black for maximum pop
	Text      = lipgloss.Color("#FFFFFF") // Pure White
	Muted     = lipgloss.Color("#888888") // Medium Gray for visibility

	// Minimalist Styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true).
			Padding(1, 1).
			MarginLeft(1)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(Muted).
			PaddingLeft(2).
			MarginBottom(1)

	CardStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(Muted).
			MarginLeft(2).
			Width(64)

	StatusLabelStyle = lipgloss.NewStyle().
				Padding(0, 1).
				Bold(true)

	InfoKeyStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Width(16)

	InfoValueStyle = lipgloss.NewStyle().
			Foreground(Text)

	InputStyle = lipgloss.NewStyle().
			Foreground(Accent).
			Bold(true)

	MutedStyle = lipgloss.NewStyle().
			Foreground(Muted)

	ErrorTextStyle = lipgloss.NewStyle().
			Foreground(ErrorCol).
			PaddingLeft(2)

	FooterStyle = lipgloss.NewStyle().
			Foreground(Muted).
			MarginTop(1).
			PaddingLeft(4).
			Faint(true)

	// Layout Styles
	MainContainerStyle = lipgloss.NewStyle().
				Padding(1, 2).
				Width(80)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(Secondary).
				Bold(true).
				MarginBottom(1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(Primary).
			Bold(true)
)

func SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(Success)
}
