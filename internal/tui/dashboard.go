package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type menuItem struct {
	label string
	to    screen
	quit  bool
	hint  string
}

// dashboard is the home screen: a vertical menu into the config screens.
type dashboard struct {
	items  []menuItem
	cursor int
}

func newDashboard() *dashboard {
	return &dashboard{
		items: []menuItem{
			{label: "Favorites & menu", to: scrFavorites, hint: "rank dishes — the booker takes the top in-stock pick"},
			{label: "Settings", to: scrSettings, hint: "run day, booking days, open hour, notifications"},
			{label: "Re-authenticate", to: scrReauth, hint: "refresh the SSO login when the session expires"},
			{label: "Quit", quit: true, hint: "leave the configurator"},
		},
	}
}

func (d *dashboard) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
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
			if it.quit {
				return d, tea.Quit
			}
			return d, nav(it.to)
		}
	}
	return d, nil
}

func (d *dashboard) View(w, h int) string {
	var b strings.Builder
	b.WriteString(subtitleStyle.Render("configure the autobooker — bookings run from the CLI") + "\n\n")
	for i, it := range d.items {
		b.WriteString(row(it.label, w, i == d.cursor) + "\n")
	}
	b.WriteString("\n" + metaStyle.Render(d.items[d.cursor].hint))
	b.WriteString("\n\n" + metaStyle.Render("book now: ") + textStyle.Render("canteen book") +
		metaStyle.Render("   preview: ") + textStyle.Render("canteen book --dry"))
	return b.String()
}

func (d *dashboard) Footer() string {
	return footStyle.Render("↑/↓ move   enter select   q quit")
}
