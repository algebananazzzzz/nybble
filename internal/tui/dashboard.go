package tui

import (
	"strings"

	"github.com/algebananazzzzz/nybble/internal/auth"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	label string
	desc  string // one-line purpose, shown inline beside the label
	group string // concern panel this item lives in
	to    screen
	quit  bool
	clear bool
}

// dashboard is the home screen: the config screens grouped into concern panels.
// items stays a flat, ordered list so the cursor (and the keyboard) flows across
// panels as one menu; the panels are derived from the items' group field.
type dashboard struct {
	items        []menuItem
	cursor       int
	notice       string // transient, already-styled status (e.g. "cleared")
	confirmClear bool   // clear wipes all data, so it asks before acting
}

// clearDoneMsg carries the result of an async auth.Clear back to the dashboard.
type clearDoneMsg struct{ err error }

func newDashboard() *dashboard {
	return &dashboard{
		items: []menuItem{
			{group: "Booking", label: "Favorites & menu", to: scrFavorites, desc: "rank dishes & vendors"},
			{group: "Booking", label: "Schedule", to: scrSchedule, desc: "when the booker runs"},
			{group: "Settings", label: "Notifications", to: scrSettings, desc: "where booking alerts go"},
			{group: "Settings", label: "Re-authenticate", to: scrReauth, desc: "refresh the SSO login"},
			{group: "System", label: "About", to: scrAbout, desc: "version, endpoints & status"},
			{group: "System", label: "Clear all data", clear: true, desc: "wipe everything local"},
			{group: "System", label: "Quit", quit: true, desc: "leave the configurator"},
		},
	}
}

func clearCmd() tea.Cmd {
	return func() tea.Msg { return clearDoneMsg{auth.Clear()} }
}

func (d *dashboard) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case clearDoneMsg:
		if msg.err != nil {
			d.notice = errNote("clear failed")
		} else {
			d.notice = okNote("cleared")
		}
		return d, loadStateCmd() // refresh the header badges (now logged out)

	case tea.KeyMsg:
		// While confirming a destructive clear, only 'y' proceeds; anything cancels.
		if d.confirmClear {
			d.confirmClear = false
			if s := msg.String(); s == "y" || s == "Y" {
				d.notice = ""
				return d, clearCmd()
			}
			d.notice = metaStyle.Render("clear cancelled")
			return d, nil
		}
		switch msg.String() {
		case "q":
			return d, tea.Quit
		case "up", "k":
			if d.cursor > 0 {
				d.cursor--
			}
		case "down", "j":
			if d.cursor < len(d.items)-1 {
				d.cursor++
			}
		case "enter", " ":
			it := d.items[d.cursor]
			switch {
			case it.quit:
				return d, tea.Quit
			case it.clear:
				d.confirmClear = true
				d.notice = ""
				return d, nil
			default:
				return d, nav(it.to)
			}
		}
	}
	return d, nil
}

func (d *dashboard) View(w, h int) string {
	var b strings.Builder
	avail := h
	if h >= 9 { // only spend lines on the tagline when the menu still has room
		b.WriteString(subtitleStyle.Render("set up the autobooker — rank, schedule, and let it run") + "\n\n")
		avail -= 2
	}

	b.WriteString(d.menu(w, avail))

	if d.confirmClear {
		b.WriteString("\n\n" + warnStyle.Render("Erase ALL local data? (keeps endpoints)  press y to confirm, any key to cancel"))
	} else if d.notice != "" {
		b.WriteString("\n\n" + d.notice)
	}
	return b.String()
}

// menu renders the items as titled concern panels, or — when the area is too
// short to fit the panels — falls back to a compact borderless list so the menu
// is always usable. The panel holding the cursor is drawn focused.
func (d *dashboard) menu(w, h int) string {
	groups := d.groups()
	labelCol := d.labelCol()
	showDesc := w >= 44

	// Panel chrome is 2 lines/group (top+bottom) plus one line per item; bail to the
	// compact list when that won't fit so small terminals never lose menu rows.
	needed := 2 * len(groups)
	for _, g := range groups {
		needed += len(g.idx)
	}
	if needed > h {
		return d.compact(w, showDesc, labelCol)
	}

	// Breathe: when there's surplus height, drop a blank line between panels so the
	// menu spreads down the screen instead of clumping at the top.
	sep := "\n"
	if len(groups) > 1 && h-needed >= len(groups)-1 {
		sep = "\n\n"
	}

	blocks := make([]string, 0, len(groups))
	for _, g := range groups {
		var body strings.Builder
		for n, i := range g.idx {
			if n > 0 {
				body.WriteString("\n")
			}
			body.WriteString(menuRow(d.items[i], w-2, labelCol, i == d.cursor, showDesc))
		}
		focused := d.cursor >= g.idx[0] && d.cursor <= g.idx[len(g.idx)-1]
		blocks = append(blocks, panel(g.title, body.String(), w, focused))
	}
	return strings.Join(blocks, sep)
}

// compact renders the flat menu without panel borders for cramped terminals.
func (d *dashboard) compact(w int, showDesc bool, labelCol int) string {
	var b strings.Builder
	for i, it := range d.items {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(menuRow(it, w, labelCol, i == d.cursor, showDesc))
	}
	return b.String()
}

// menuRow renders one item line w columns wide: an accent bar + highlight when
// selected, the label, and a dim inline description when there's room.
func menuRow(it menuItem, w, labelCol int, selected, showDesc bool) string {
	text := it.label
	if showDesc && it.desc != "" {
		text = padRight(it.label, labelCol) + it.desc
	}
	if selected {
		bar := barStyle.Render(barGlyph)
		return " " + bar + selRowStyle.Width(w-2).Render(truncate(" "+text, w-2))
	}
	if showDesc && it.desc != "" {
		line := "  " + textStyle.Render(padRight(it.label, labelCol)) + footStyle.Render(it.desc)
		return padLine(line, w)
	}
	return norRowStyle.Width(w).Render(truncate("  "+text, w))
}

// menuGroup is one concern panel: its title and the indices (into items) it owns.
type menuGroup struct {
	title string
	idx   []int
}

// groups derives the ordered concern panels from items' group field, preserving
// first-seen order so adding an item just slots into its group.
func (d *dashboard) groups() []menuGroup {
	var out []menuGroup
	pos := map[string]int{}
	for i, it := range d.items {
		p, ok := pos[it.group]
		if !ok {
			pos[it.group] = len(out)
			out = append(out, menuGroup{title: it.group})
			p = len(out) - 1
		}
		out[p].idx = append(out[p].idx, i)
	}
	return out
}

// labelCol is the width the label column is padded to before the inline
// description, so every description starts at the same column.
func (d *dashboard) labelCol() int {
	max := 0
	for _, it := range d.items {
		if n := len([]rune(it.label)); n > max {
			max = n
		}
	}
	return max + 3
}

func (d *dashboard) Footer() string {
	if d.confirmClear {
		return footStyle.Render("y confirm clear   any key cancel")
	}
	return footStyle.Render("↑/↓ move   enter select   q quit")
}
