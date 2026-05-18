package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Banner returns the welcome screen rendered as a string.
func Banner(version string) string {
	logo := []string{
		"  ╭───────────────╮",
		"  │  ┏━┓┏━╸┏━┓┏━┓ │",
		"  │  ┣┳┛┣╸ ┣━┛┗━┓ │",
		"  │  ╹╰╴┗━╸╹   ━┛ │",
		"  ╰───────────────╯",
	}
	logoStr := Brand.Render(strings.Join(logo, "\n"))

	tag := lipgloss.JoinHorizontal(
		lipgloss.Top,
		Pill.Render("reps"),
		" ",
		Subtle.Render(fmt.Sprintf("v%s · personalized interview rehearsal", version)),
	)

	body := Subtle.Render(
		"Four agents · Planner, Interviewer, Judge, Coach. Theory-only.\n" +
			"Reads your real shipped work. ELO-tracked. Local-first.",
	)

	hint := Dim.Render(
		"You can press " + lipgloss.NewStyle().Foreground(primary).Render("Ctrl-C") +
			" at any time to quit. Settings save to ~/.reps/.",
	)

	return Box.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			logoStr,
			"",
			tag,
			"",
			body,
			"",
			hint,
		),
	)
}

// Heading returns a labelled step heading like:
//   ◆ Step 2/5 · API key
func Heading(step, total int, label string) string {
	left := Brand.Render(fmt.Sprintf("◆ Step %d/%d", step, total))
	right := Subtle.Render(IconDot + " " + label)
	return left + " " + right + "\n"
}

// Done renders the final success card.
func Done(summary []string) string {
	header := lipgloss.JoinHorizontal(
		lipgloss.Top,
		OK.Render(IconOK+" ready"),
		"   ",
		Subtle.Render("reps is set up."),
	)
	body := strings.Join(summary, "\n")
	next := Dim.Render(
		"Next: " + Mono.Render("make dev") +
			"  or  " + Mono.Render("reps drill --qs 3"),
	)
	return Box.BorderForeground(primary).Render(
		lipgloss.JoinVertical(lipgloss.Left, header, "", body, "", next),
	)
}
