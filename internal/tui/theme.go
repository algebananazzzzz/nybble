package tui

import (
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

// huhTheme dresses the Settings form in the app palette so it reads as part of the
// same UI as the hand-built screens (periwinkle accent, calm neutrals, the same
// bar glyph and check/dot prefixes), instead of huh's generic default.
func huhTheme() *huh.Theme {
	t := huh.ThemeBase()
	f, b := &t.Focused, &t.Blurred

	t.FieldSeparator = lipgloss.NewStyle().SetString("\n")

	// Focused field: left accent bar, bold accent title, accent selection.
	f.Base = f.Base.Border(lipgloss.NormalBorder(), false).BorderLeft(true).
		BorderForeground(cAccent).PaddingLeft(1).MarginBottom(1)
	f.Title = f.Title.Foreground(cAccent).Bold(true)
	f.Description = f.Description.Foreground(cDim)
	f.ErrorIndicator = f.ErrorIndicator.Foreground(cErr)
	f.ErrorMessage = f.ErrorMessage.Foreground(cErr)
	f.SelectSelector = lipgloss.NewStyle().Foreground(cAccent).SetString(barGlyph + " ")
	f.Option = f.Option.Foreground(cText)
	f.SelectedOption = lipgloss.NewStyle().Foreground(cAccent).Bold(true)
	f.SelectedPrefix = lipgloss.NewStyle().Foreground(cOK).SetString(checkGlyph + " ")
	f.UnselectedPrefix = lipgloss.NewStyle().Foreground(cFaint).SetString(dot + " ")
	f.UnselectedOption = f.UnselectedOption.Foreground(cText)
	f.MultiSelectSelector = lipgloss.NewStyle().Foreground(cAccent).SetString(barGlyph + " ")
	f.NextIndicator = f.NextIndicator.Foreground(cAccent)
	f.PrevIndicator = f.PrevIndicator.Foreground(cAccent)
	f.TextInput.Cursor = f.TextInput.Cursor.Foreground(cAccent)
	f.TextInput.Prompt = f.TextInput.Prompt.Foreground(cAccent)
	f.TextInput.Text = f.TextInput.Text.Foreground(cText)
	f.TextInput.Placeholder = f.TextInput.Placeholder.Foreground(cFaint)

	// Blurred field: dimmed, border hidden so inactive fields recede.
	b.Base = b.Base.Border(lipgloss.HiddenBorder(), false).BorderLeft(true).
		PaddingLeft(1).MarginBottom(1)
	b.Title = b.Title.Foreground(cDim)
	b.Description = b.Description.Foreground(cFaint)
	b.SelectedOption = lipgloss.NewStyle().Foreground(cDim)
	b.TextInput.Text = b.TextInput.Text.Foreground(cDim)

	return t
}

// Palette — one accent plus a calm neutral ramp. No emoji anywhere in the UI;
// hierarchy comes from weight, color and spacing.
var (
	cAccent  = lipgloss.Color("141") // periwinkle — selection, titles
	cAccent2 = lipgloss.Color("105") // deeper accent — frame
	cText    = lipgloss.Color("252")
	cDim     = lipgloss.Color("245")
	cFaint   = lipgloss.Color("240")
	cSubtle  = lipgloss.Color("237") // selected-row background
	cInvert  = lipgloss.Color("231")
	cOK      = lipgloss.Color("114")
	cWarn    = lipgloss.Color("215")
	cErr     = lipgloss.Color("203")
)

var (
	appTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	brandStyle    = lipgloss.NewStyle().Foreground(cFaint)
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(cAccent)
	subtitleStyle = lipgloss.NewStyle().Foreground(cFaint).Italic(true)
	metaStyle     = lipgloss.NewStyle().Foreground(cDim)
	ruleStyle     = lipgloss.NewStyle().Foreground(cFaint)
	textStyle     = lipgloss.NewStyle().Foreground(cText)
	footStyle     = lipgloss.NewStyle().Foreground(cFaint)
	hintKeyStyle  = lipgloss.NewStyle().Foreground(cDim).Bold(true)

	okStyle   = lipgloss.NewStyle().Foreground(cOK).Bold(true)
	warnStyle = lipgloss.NewStyle().Foreground(cWarn).Bold(true)
	errStyle  = lipgloss.NewStyle().Foreground(cErr).Bold(true)

	// barStyle is the left accent bar on a selected row; selRowStyle fills the row.
	barStyle     = lipgloss.NewStyle().Foreground(cAccent)
	selRowStyle  = lipgloss.NewStyle().Bold(true).Foreground(cInvert).Background(cSubtle)
	grabRowStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("232")).
			Background(cWarn)
	norRowStyle = lipgloss.NewStyle().Foreground(cText)

	// docStyle frames the whole screen. GetFrameSize() reports the chrome we subtract.
	docStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cAccent2).
			Padding(1, 2)
)

const (
	barGlyph   = "▐"
	dot        = "·"
	checkGlyph = "✓"
	crossGlyph = "✗"
)

// okNote / errNote are the shared success / failure status indicators used
// across screens so every operation confirms its outcome the same way.
func okNote(s string) string  { return okStyle.Render(checkGlyph + " " + s) }
func errNote(s string) string { return errStyle.Render(crossGlyph + " " + s) }

func rule(w int) string {
	if w < 1 {
		w = 1
	}
	return ruleStyle.Render(strings.Repeat("─", w))
}

// row renders one selectable line: an accent bar + label when selected, padding
// otherwise, filling exactly w columns. style is applied to the whole row.
func row(label string, w int, selected bool) string {
	if selected {
		bar := barStyle.Render(barGlyph)
		inner := truncate(" "+label, w-1)
		return bar + selRowStyle.Width(w-1).Render(inner)
	}
	return norRowStyle.Width(w).Render(truncate("  "+label, w))
}

// truncate shortens s to at most w display columns (latin-width approximation).
func truncate(s string, w int) string {
	r := []rune(s)
	if w < 1 {
		w = 1
	}
	if len(r) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	return string(r[:w-1]) + "…"
}
