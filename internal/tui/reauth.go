package tui

import (
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// reauth refreshes the SSO login. auth.Login() reads stdin and prints to the
// terminal, so it cannot run as a background Cmd while Bubble Tea owns the tty.
// Instead we use tea.ExecProcess to suspend the TUI and run `canteen auth` as a
// subprocess with full terminal access, then resume and refresh state.
type reauth struct {
	running bool
	done    bool
	err     error
}

func newReauth() (*reauth, tea.Cmd) {
	r := &reauth{running: true}
	return r, r.launch()
}

func (r *reauth) launch() tea.Cmd {
	exe, err := os.Executable()
	if err != nil {
		return func() tea.Msg { return reauthDoneMsg{err} }
	}
	c := exec.Command(exe, "auth")
	return tea.ExecProcess(c, func(err error) tea.Msg { return reauthDoneMsg{err} })
}

func (r *reauth) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case reauthDoneMsg:
		r.running = false
		r.done = true
		r.err = msg.err
		return r, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			return r, nav(scrDashboard)
		case "enter", "r":
			if r.done {
				r.running = true
				r.done = false
				r.err = nil
				return r, r.launch()
			}
		}
	}
	return r, nil
}

func (r *reauth) View(w, h int) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Re-authenticate") + "\n\n")
	switch {
	case r.running:
		b.WriteString(textStyle.Render("Launching browser login…") + "\n\n")
		b.WriteString(metaStyle.Render("Scan the SSO QR, wait for the canteen menu, then press Enter in that window."))
	case r.err == nil:
		b.WriteString(okNote("logged in") + textStyle.Render("  session refreshed"))
	default:
		b.WriteString(errNote("login failed") + "\n\n" +
			textStyle.Render(truncate(r.err.Error(), w)))
	}
	return b.String()
}

func (r *reauth) Footer() string {
	if r.done {
		return footStyle.Render("r retry   esc back")
	}
	return footStyle.Render("follow the browser   esc back")
}
