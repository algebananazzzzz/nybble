package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Rounded box glyphs for the hand-drawn titled panels. We draw these by hand
// rather than with lipgloss.Border so the section title can sit in the top edge
// (╭─ TITLE ─────╮), which lipgloss borders don't support.
const (
	boxTL = "╭"
	boxTR = "╮"
	boxBL = "╰"
	boxBR = "╯"
	boxH  = "─"
	boxV  = "│"
)

// panel wraps body in a titled rounded box exactly w columns wide. body is laid
// out into the inner area (w-2 columns); lines too wide are truncated, short ones
// padded, so the right edge always lines up. A focused panel brightens its border
// and title so the one holding the cursor stands out from the idle ones.
func panel(title, body string, w int, focused bool) string {
	if w < 4 {
		w = 4
	}
	innerW := w - 2

	border := cFaint
	titleFg := cDim
	if focused {
		border = cAccent2
		titleFg = cAccent
	}
	bs := lipgloss.NewStyle().Foreground(border)
	ts := lipgloss.NewStyle().Foreground(titleFg).Bold(true)

	// Top edge: ╭─ title ──…──╮. The leading "─ " + title + " " eats into innerW;
	// the rest is filled with dashes.
	label := " " + title + " "
	used := 1 + lipgloss.Width(label) // leading dash + " title "
	if used > innerW {
		label = truncate(label, innerW-1)
		used = 1 + lipgloss.Width(label)
	}
	fill := innerW - used
	if fill < 0 {
		fill = 0
	}
	top := bs.Render(boxTL+boxH) + ts.Render(label) +
		bs.Render(strings.Repeat(boxH, fill)+boxTR)

	var b strings.Builder
	b.WriteString(top + "\n")
	for _, ln := range strings.Split(body, "\n") {
		b.WriteString(bs.Render(boxV) + padLine(ln, innerW) + bs.Render(boxV) + "\n")
	}
	b.WriteString(bs.Render(boxBL + strings.Repeat(boxH, innerW) + boxBR))
	return b.String()
}

// padLine pads a (possibly styled) line to exactly w display columns, truncating
// if it overflows, so it fills a panel row edge-to-edge.
func padLine(s string, w int) string {
	gap := w - lipgloss.Width(s)
	if gap < 0 {
		return truncateLine(s, w)
	}
	return s + strings.Repeat(" ", gap)
}

// padRight pads plain text on the right to w runes (no truncation) — used to align
// the label column inside a panel before the dim description.
func padRight(s string, w int) string {
	if n := len([]rune(s)); n < w {
		return s + strings.Repeat(" ", w-n)
	}
	return s
}
