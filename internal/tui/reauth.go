package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/algebananazzzzz/nybble/internal/auth"
	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// reauth runs the SSO browser login entirely inside the TUI. Instead of suspending
// Bubble Tea to run `nybble auth` as a subprocess, it drives auth's composable steps
// (OpenBrowser → Probe… → Finalize) as async Cmds and shows live progress: the
// browser opens in its own window, the screen polls the session, and once you've
// logged in and opened your canteen menu it finalizes and detects the building.
type reauthPhase int

const (
	raIdle       reauthPhase = iota
	raOpening                // launching the browser
	raPolling                // browser up; polling until logged in + building seen
	raFinalizing             // saving cookies + detecting building
	raDone                   // finished (ok or error)
)

// pollInterval is how often Probe runs. pollTimeout caps the overall wait for the
// SSO login to complete; buildingGrace caps how long after login we keep waiting for
// the canteen menu (building) to appear before finishing logged-in without it.
const (
	pollInterval  = 2 * time.Second
	pollTimeout   = 1 * time.Minute
	buildingGrace = 25 * time.Second
)

// authStep is the reauth screen's next action after a probe.
type authStep int

const (
	stepWait authStep = iota
	stepFinalize
	stepFail
)

// nextAuthStep decides what to do from the latest probe. Once logged in, finalize as
// soon as the building is known, or after sinceLogin passes buildingGrace (the user
// may never open their menu). If login itself never completes within timeout, fail.
func nextAuthStep(loggedIn bool, buildingCode string, elapsed, sinceLogin, timeout, grace time.Duration) authStep {
	if loggedIn && (buildingCode != "" || sinceLogin >= grace) {
		return stepFinalize
	}
	if !loggedIn && elapsed >= timeout {
		return stepFail
	}
	return stepWait
}

type reauth struct {
	phase    reauthPhase
	apiBase  string
	epErr    error // endpoints missing → can't auth at all
	sp       spinner.Model
	start    time.Time // poll start
	loginAt  time.Time // first probe that reported logged-in (zero until then)
	elapsed  time.Duration
	building string
	note     error // non-fatal building-detection note
	err      error
}

// reauth-screen messages, routed through the Bubble Tea loop.
type (
	browserOpenedMsg struct{ err error }
	probeResultMsg   struct {
		loggedIn bool
		building string
	}
	pollTickMsg struct{}
)

func newReauth() (*reauth, tea.Cmd) {
	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(cAccent)
	r := &reauth{phase: raIdle, sp: sp}
	if eps, err := config.LoadEndpoints(); err != nil {
		r.epErr = err
	} else {
		r.apiBase = eps.APIBase
	}
	return r, nil // idle until the user presses Enter
}

func (r *reauth) begin() tea.Cmd {
	r.phase = raOpening
	r.err, r.note, r.building = nil, nil, ""
	r.loginAt = time.Time{}
	return tea.Batch(r.sp.Tick, openBrowserCmd())
}

func openBrowserCmd() tea.Cmd {
	return func() tea.Msg { return browserOpenedMsg{auth.OpenBrowser()} }
}

func (r *reauth) probeCmd() tea.Cmd {
	api := r.apiBase
	return func() tea.Msg {
		ok, b := auth.Probe(api)
		return probeResultMsg{loggedIn: ok, building: b}
	}
}

func finalizeCmd(apiBase string) tea.Cmd {
	return func() tea.Msg {
		det, err := auth.Finalize(apiBase)
		return reauthDoneMsg{err: err, note: det}
	}
}

func (r *reauth) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		if r.phase == raOpening || r.phase == raPolling || r.phase == raFinalizing {
			var cmd tea.Cmd
			r.sp, cmd = r.sp.Update(msg)
			return r, cmd
		}
		return r, nil

	case browserOpenedMsg:
		if msg.err != nil {
			r.phase, r.err = raDone, msg.err
			return r, nil
		}
		r.phase = raPolling
		r.start = time.Now()
		return r, tea.Batch(r.sp.Tick, r.probeCmd())

	case probeResultMsg:
		r.elapsed = time.Since(r.start)
		r.building = msg.building
		if msg.loggedIn && r.loginAt.IsZero() {
			r.loginAt = time.Now()
		}
		var sinceLogin time.Duration
		if !r.loginAt.IsZero() {
			sinceLogin = time.Since(r.loginAt)
		}
		switch nextAuthStep(msg.loggedIn, msg.building, r.elapsed, sinceLogin, pollTimeout, buildingGrace) {
		case stepFinalize:
			r.phase = raFinalizing
			return r, tea.Batch(r.sp.Tick, finalizeCmd(r.apiBase))
		case stepFail:
			r.phase = raDone
			r.err = fmt.Errorf("timed out waiting for login")
			auth.CloseBrowser()
			return r, nil
		default: // stepWait
			return r, tea.Tick(pollInterval, func(time.Time) tea.Msg { return pollTickMsg{} })
		}

	case pollTickMsg:
		if r.phase != raPolling {
			return r, nil
		}
		return r, r.probeCmd()

	case reauthDoneMsg:
		r.phase = raDone
		r.err = msg.err
		r.note = msg.note
		return r, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			if r.phase == raOpening || r.phase == raPolling || r.phase == raFinalizing {
				auth.CloseBrowser()
			}
			return r, nav(scrDashboard)
		case "enter", "r":
			if r.epErr != nil {
				return r, nil // nothing to launch without endpoints
			}
			if r.phase == raIdle || r.phase == raDone {
				return r, r.begin()
			}
		}
	}
	return r, nil
}

func (r *reauth) View(w, h int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Authenticate") + "\n\n")
	switch r.phase {
	case raOpening:
		b.WriteString(r.sp.View() + textStyle.Render(" Opening browser…"))

	case raPolling:
		b.WriteString(r.sp.View() + textStyle.Render(" Waiting for login…") + "\n\n")
		b.WriteString(metaStyle.Render("In the browser: scan the SSO QR, then open your canteen menu.") + "\n")
		status := fmt.Sprintf("Detecting automatically — %ds elapsed", int(r.elapsed.Seconds()))
		if r.building != "" {
			status = "Building found — finishing up…"
		}
		b.WriteString(metaStyle.Render(status))

	case raFinalizing:
		b.WriteString(r.sp.View() + textStyle.Render(" Logged in — saving session & detecting building…"))

	case raDone:
		switch {
		case r.err != nil:
			b.WriteString(errNote("login failed") + "\n\n" + textStyle.Render(truncate(r.err.Error(), w)))
		case r.note != nil:
			b.WriteString(okNote("logged in") + "\n\n" +
				textStyle.Render("building not detected — re-run auth and open your canteen menu"))
		default:
			b.WriteString(okNote("logged in") + textStyle.Render("  building detected · session refreshed"))
		}

	default: // raIdle
		if r.epErr != nil {
			b.WriteString(errNote("not configured") + "\n\n" +
				textStyle.Render("set NYBBLE_API_BASE / NYBBLE_LOGIN_URL first (see README)"))
			break
		}
		b.WriteString(textStyle.Render("Log in to use Favorites and Settings.") + "\n\n")
		b.WriteString(metaStyle.Render("Press Enter to open the browser login, or esc to go back."))
	}
	return b.String()
}

func (r *reauth) Footer() string {
	switch r.phase {
	case raOpening, raFinalizing:
		return footStyle.Render("working…   esc cancel")
	case raPolling:
		return footStyle.Render("complete the login in the browser   esc cancel")
	case raDone:
		if r.err != nil {
			return footStyle.Render("r retry   esc back")
		}
		return footStyle.Render("esc back")
	default:
		return footStyle.Render("enter log in   esc back")
	}
}
