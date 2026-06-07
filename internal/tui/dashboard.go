package tui

import (
	"strings"

	"github.com/algebananazzzzz/nybble/internal/auth"
	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	label string
	to    screen
	quit  bool
	clear bool
	hint  string
}

// dashboard is the home screen: a vertical menu into the config screens.
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
			{label: "Favorites & menu", to: scrFavorites, hint: "rank dishes — the booker takes the top in-stock pick"},
			{label: "Schedule", to: scrSchedule, hint: "when the booker runs + turn it on (installs the weekly job)"},
			{label: "Settings", to: scrSettings, hint: "where booking alerts go"},
			{label: "Re-authenticate", to: scrReauth, hint: "refresh the SSO login when the session expires"},
			{label: "Clear all data", clear: true, hint: "wipe everything local — cookies, config, favorites, catalog (keeps endpoints)"},
			{label: "Quit", quit: true, hint: "leave the configurator"},
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
	b.WriteString(subtitleStyle.Render("set up the autobooker — rank, schedule, and let it run") + "\n\n")
	for i, it := range d.items {
		b.WriteString(row(it.label, w, i == d.cursor) + "\n")
	}
	if d.confirmClear {
		b.WriteString("\n" + warnStyle.Render("Erase ALL local data? (keeps endpoints)  press y to confirm, any key to cancel"))
	} else {
		b.WriteString("\n" + metaStyle.Render(d.items[d.cursor].hint))
		if d.notice != "" {
			b.WriteString("\n" + d.notice)
		}
	}
	return b.String()
}

func (d *dashboard) Footer() string {
	if d.confirmClear {
		return footStyle.Render("y confirm clear   any key cancel")
	}
	return footStyle.Render("↑/↓ move   enter select   q quit")
}
