package tui

import "github.com/charmbracelet/lipgloss"

// Obsidian Spark palette in 24-bit colour. Mirrors web/app/globals.css.
var (
	primary   = lipgloss.Color("#a78bfa") // electric violet (oklch 0.66 0.22 285)
	primaryFg = lipgloss.Color("#fafafa")
	fg        = lipgloss.Color("#f5f5fa")
	muted     = lipgloss.Color("#9491b3")
	border    = lipgloss.Color("#2b2940")
	success   = lipgloss.Color("#84cc8d")
	danger    = lipgloss.Color("#ef7d70")
	warning   = lipgloss.Color("#fbbf61")
)

var (
	Title = lipgloss.NewStyle().
		Foreground(fg).
		Bold(true).
		MarginBottom(1)

	Subtle = lipgloss.NewStyle().
		Foreground(muted)

	Brand = lipgloss.NewStyle().
		Foreground(primary).
		Bold(true)

	Pill = lipgloss.NewStyle().
		Foreground(primaryFg).
		Background(primary).
		Padding(0, 1).
		Bold(true)

	Box = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(1, 2)

	OK = lipgloss.NewStyle().Foreground(success).Bold(true)
	X  = lipgloss.NewStyle().Foreground(danger).Bold(true)
	Wn = lipgloss.NewStyle().Foreground(warning).Bold(true)

	Mono = lipgloss.NewStyle().Foreground(fg)

	Dim = lipgloss.NewStyle().Foreground(muted).Italic(true)
)

const (
	IconOK    = "✓"
	IconErr   = "✗"
	IconWarn  = "!"
	IconArrow = "→"
	IconDot   = "•"
)
