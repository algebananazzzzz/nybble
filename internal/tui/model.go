package tui

import (
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

// State is the cached status snapshot shown in the header. It is loaded
// asynchronously (snapshot calls the network) so the event loop never blocks.
type State struct {
	Loading  bool
	LoggedIn bool
	FavCount int
	Building string
	NotifyCh string
}

type screen int

const (
	scrDashboard screen = iota
	scrFavorites
	scrSettings
	scrSchedule
	scrReauth
)

// screenModel is one full-screen view. View renders only the body box; the root
// supplies header/footer chrome. width/height are the body area in cells.
type screenModel interface {
	Update(tea.Msg) (screenModel, tea.Cmd)
	View(width, height int) string
	Footer() string
}

// Messages routed through the Bubble Tea loop instead of mutating siblings.
type (
	stateMsg      State // async snapshot result
	navMsg        struct{ to screen }
	bodySizeMsg   struct{ w, h int } // body area handed to the active child
	reauthDoneMsg struct{ err, note error }
)

type Model struct {
	width, height int
	body          struct{ w, h int }
	screen        screen
	state         State
	child         screenModel
}

func New() Model {
	m := Model{
		width: 80, height: 24,
		screen: scrDashboard,
		state:  State{Loading: true},
		child:  newDashboard(),
	}
	m.body.w, m.body.h = bodySize(m.width, m.height)
	return m
}

func (m Model) Init() tea.Cmd { return loadStateCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.body.w, m.body.h = bodySize(m.width, m.height)
		var cmd tea.Cmd
		m.child, cmd = m.child.Update(bodySizeMsg{m.body.w, m.body.h})
		return m, cmd

	case stateMsg:
		m.state = State(msg)
		return m, nil

	case navMsg:
		return m.navigate(msg.to)

	case reauthDoneMsg:
		// let the reauth screen show its result, and refresh the header badges.
		var cmd tea.Cmd
		m.child, cmd = m.child.Update(msg)
		return m, tea.Batch(cmd, loadStateCmd())

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.child, cmd = m.child.Update(msg)
	return m, cmd
}

// navigate swaps the active screen, sizes the new child, and refreshes state
// when returning to the dashboard (a sub-screen may have changed config/auth).
func (m Model) navigate(to screen) (tea.Model, tea.Cmd) {
	// Favorites and Settings need a live session (catalog rescan, profile, etc.),
	// so gate them behind auth: a logged-out user is sent to the re-auth screen.
	if (to == scrFavorites || to == scrSettings || to == scrSchedule) && !m.state.LoggedIn {
		to = scrReauth
	}
	m.screen = to
	var cmds []tea.Cmd

	switch to {
	case scrDashboard:
		m.child = newDashboard()
		cmds = append(cmds, loadStateCmd())
	case scrFavorites:
		m.child = newFavModel()
	case scrSettings:
		c, cmd := newSettings()
		m.child = c
		cmds = append(cmds, cmd)
	case scrSchedule:
		c, cmd := newSchedule()
		m.child = c
		cmds = append(cmds, cmd)
	case scrReauth:
		c, cmd := newReauth()
		m.child = c
		cmds = append(cmds, cmd)
	}

	var szcmd tea.Cmd
	m.child, szcmd = m.child.Update(bodySizeMsg{m.body.w, m.body.h})
	cmds = append(cmds, szcmd)
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return frame(m.width, m.height, m.state,
		m.child.View(m.body.w, m.body.h), m.child.Footer())
}

// loadStateCmd snapshots config/cookies (and validates the session over the
// network) off the event loop.
func loadStateCmd() tea.Cmd {
	return func() tea.Msg { return stateMsg(snapshot()) }
}

func snapshot() State {
	dir, _ := config.ConfigDir()
	cfg, err := config.Load(filepath.Join(dir, "config.json"))
	if err != nil {
		cfg = config.Default()
	}
	favs, _ := config.LoadFavorites(filepath.Join(dir, "favorites.json"))
	cookies, cerr := session.LoadCookies(filepath.Join(dir, "cookies.json"))
	notifyCh := "off"
	if cfg.Notify.LarkOn() {
		notifyCh = "lark"
	}
	st := State{
		FavCount: len(favs),
		Building: cfg.Building.Name, NotifyCh: notifyCh,
	}
	if eps, eperr := config.LoadEndpoints(); eperr == nil && cerr == nil && len(cookies) > 0 && session.Valid(cookies, eps.APIBase) {
		st.LoggedIn = true
	}
	return st
}

// nav returns a Cmd that requests a screen change through the message loop.
func nav(to screen) tea.Cmd {
	return func() tea.Msg { return navMsg{to} }
}

func Run() error {
	// Libraries such as net/http log to the standard logger (os.Stderr by default).
	// In alt-screen mode those lines paint over the rendered frame, so route the
	// std logger to a file (falling back to discard) for the program's lifetime.
	restore := quietStdLog()
	defer restore()

	_, err := tea.NewProgram(New(), tea.WithAltScreen()).Run()
	return err
}

func quietStdLog() func() {
	prev := log.Writer()
	if dir, err := config.ConfigDir(); err == nil {
		if f, ferr := os.OpenFile(filepath.Join(dir, "nybble.log"),
			os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600); ferr == nil {
			log.SetOutput(f)
			return func() { log.SetOutput(prev); f.Close() }
		}
	}
	log.SetOutput(io.Discard)
	return func() { log.SetOutput(prev) }
}
