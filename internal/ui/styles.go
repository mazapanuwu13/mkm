package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Nord-inspired palette
	colorBg        = lipgloss.Color("#2E3440")
	colorBgPanel   = lipgloss.Color("#3B4252")
	colorBorder    = lipgloss.Color("#4C566A")
	colorAccent    = lipgloss.Color("#88C0D0")
	colorSelected  = lipgloss.Color("#81A1C1")
	colorText      = lipgloss.Color("#ECEFF4")
	colorMuted     = lipgloss.Color("#D8DEE9")
	colorSuccess   = lipgloss.Color("#A3BE8C")
	colorError     = lipgloss.Color("#BF616A")
	colorHighlight = lipgloss.Color("#5E81AC")

	PanelStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	ActivePanelStyle = PanelStyle.
				BorderForeground(colorAccent)

	TitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 1)

	ItemStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Background(colorHighlight).
				Bold(true).
				Padding(0, 1)

	DescStyle = lipgloss.NewStyle().
			Foreground(colorBorder).
			Italic(true)

	OutputStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
	StatusSuccessStyle = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	StatusErrorStyle   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	StatusRunningStyle = lipgloss.NewStyle().Foreground(colorSelected).Bold(true)

	AppTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true).
			Background(colorBgPanel).
			Padding(0, 2)

	HelpStyle = lipgloss.NewStyle().
			Foreground(colorBorder).
			Padding(0, 1)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	BreadcrumbStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Background(colorBgPanel).
				Bold(true).
				Padding(0, 1)
)
