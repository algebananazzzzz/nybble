package tui

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/schedule"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const repoURL = "github.com/algebananazzzzz/nybble"

// about is a read-only info screen: build version, configured endpoints, on-disk
// paths and a live status summary — a single place to confirm "is this thing set
// up right?" It reuses the header State snapshot for login/favorites/building so
// it never blocks on the network.
type about struct {
	st       State
	dir      string
	apiBase  string
	loginURL string
	sched    string // human schedule summary
}

func newAbout(st State) *about {
	a := &about{st: st}
	a.dir, _ = config.ConfigDir()

	if eps, err := config.LoadEndpoints(); err == nil {
		a.apiBase, a.loginURL = eps.APIBase, eps.LoginURL
	}

	cfg, err := config.Load(filepath.Join(a.dir, "config.json"))
	if err != nil {
		cfg = config.Default()
	}
	if schedule.Installed() {
		a.sched = fmt.Sprintf("on · %s %02d:00 · notify %d min before",
			dayLabel(normalizeRunDay(cfg.Schedule.Weekday)), cfg.Schedule.Hour, cfg.Schedule.Lead())
	} else {
		a.sched = "off"
	}
	return a
}

func (a *about) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok {
		switch k.String() {
		case "esc", "q", "enter":
			return a, nav(scrDashboard)
		}
	}
	return a, nil
}

func (a *about) View(w, h int) string {
	head := appTitleStyle.Render("nybble") + "  " + metaStyle.Render(Version) + "\n" +
		subtitleStyle.Render("canteen lunch autobooker · "+repoURL)

	endpoints := a.kv(w, [][2]string{
		{"API", orDash(a.apiBase)},
		{"Login", orDash(a.loginURL)},
	})
	paths := a.kv(w, [][2]string{
		{"Config", orDash(a.dir)},
		{"Log", filepath.Join(a.dir, "nybble.log")},
	})
	status := a.kv(w, [][2]string{
		{"Login", loginLabel(a.st)},
		{"Building", orDash(a.st.Building)},
		{"Favorites", strconv.Itoa(a.st.FavCount) + " dishes"},
		{"Schedule", a.sched},
		{"Notify", orDash(a.st.NotifyCh)},
	})

	return lipgloss.JoinVertical(lipgloss.Left,
		head,
		"",
		panel("Endpoints", endpoints, w, false),
		panel("Status", status, w, false),
		panel("Paths", paths, w, false),
	)
}

// kv renders aligned key/value rows sized to a panel's inner width (w-2).
func (a *about) kv(w int, rows [][2]string) string {
	inner := w - 2
	out := make([]string, len(rows))
	for i, r := range rows {
		line := " " + metaStyle.Render(padRight(r[0], 11)) + textStyle.Render(r[1])
		out[i] = padLine(line, inner)
	}
	return lipgloss.JoinVertical(lipgloss.Left, out...)
}

func (a *about) Footer() string { return footStyle.Render("esc back") }

func loginLabel(st State) string {
	if st.LoggedIn {
		return okStyle.Render(checkGlyph + " active")
	}
	return errStyle.Render(crossGlyph + " logged out")
}
