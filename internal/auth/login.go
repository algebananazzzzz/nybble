package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/algebananazzzzz/nybble/internal/config"
	"github.com/algebananazzzzz/nybble/internal/session"
)

// pwSession is the playwright-cli session name shared by all auth browser steps.
const pwSession = "nybble"

// loginHelper is the external browser-automation CLI that drives the one-time SSO
// login. It is shelled out in every auth step, so its absence must be reported once,
// up front, with an actionable hint — see ensureLoginHelper.
const loginHelper = "playwright-cli"

func profilePath() (string, error) { return filepath.Abs(".auth/profile") }
func statePath() (string, error)   { return filepath.Abs(".auth/state.json") }

// ensureLoginHelper verifies playwright-cli is installed before any browser step runs,
// so a missing dependency surfaces as a clear install instruction instead of a cryptic
// "executable file not found in $PATH". Mirrors notify.ProbeLark's dependency gate.
func ensureLoginHelper() error {
	if _, err := exec.LookPath(loginHelper); err != nil {
		return fmt.Errorf("%s not found on PATH — required for `nybble auth`; install it with "+
			"`npm i -g @playwright/cli && %s install chromium` (see README → Requirements)",
			loginHelper, loginHelper)
	}
	return nil
}

// OpenBrowser launches the headed SSO login browser in a persistent playwright-cli
// session and returns once it is up; the session stays alive for Probe/Finalize.
func OpenBrowser() error {
	if err := ensureLoginHelper(); err != nil {
		return err
	}
	eps, err := config.LoadEndpoints()
	if err != nil {
		return err
	}
	profile, err := profilePath()
	if err != nil {
		return err
	}
	c := exec.Command("playwright-cli", "-s="+pwSession, "open", "--headed", "--persistent",
		"--profile="+profile, eps.LoginURL)
	// Keep stderr for real launch failures; playwright-cli's verbose markdown goes nowhere.
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	return nil
}

// CloseBrowser tears down the auth browser session. Safe to call more than once.
func CloseBrowser() { _ = exec.Command("playwright-cli", "-s="+pwSession, "close").Run() }

// Probe snapshots the live browser session and reports whether it authenticates yet
// and which building (if any) the user has browsed to. It is read-only and safe to
// call repeatedly while the user completes the SSO login — the TUI polls it.
func Probe(apiBase string) (loggedIn bool, buildingCode string) {
	state, err := statePath()
	if err != nil {
		return false, ""
	}
	if err := exec.Command("playwright-cli", "-s="+pwSession, "state-save", state).Run(); err != nil {
		return false, ""
	}
	store, err := session.FromPlaywrightState(state)
	if err != nil {
		return false, ""
	}
	if len(store) == 0 || !session.Valid(store, apiBase) {
		return false, ""
	}
	// Logged in: the building comes straight from the API preference (no menu needed).
	code, _, _ := session.ClientFor(store, apiBase, nil).CurrentBuilding()
	return true, code
}

// Finalize persists the authenticated cookies, auto-detects and saves the building,
// and closes the browser. Call once Probe reports a logged-in session. The returned
// detErr is a non-fatal building-detection note; err is a hard failure.
func Finalize(apiBase string) (detErr, err error) {
	defer CloseBrowser()
	state, perr := statePath()
	if perr != nil {
		return nil, perr
	}
	if serr := exec.Command("playwright-cli", "-s="+pwSession, "state-save", state).Run(); serr != nil {
		return nil, fmt.Errorf("state-save: %w", serr)
	}
	store, serr := session.FromPlaywrightState(state)
	if serr != nil {
		return nil, serr
	}
	dir, derr := config.ConfigDir()
	if derr != nil {
		return nil, derr
	}
	if serr := store.Save(filepath.Join(dir, "cookies.json")); serr != nil {
		return nil, serr
	}
	detErr = detectLocation(apiBase, store)
	_ = os.Remove(state)
	if !session.Valid(store, apiBase) {
		return detErr, fmt.Errorf("login did not produce a valid session")
	}
	return detErr, nil
}

// Clear resets the app to a clean slate. It removes everything under the config
// directory — cookies, config.json (building/pickup/schedule/notify), favorites,
// catalog, vendors, the exclude list, logs — and the playwright browser login
// profile. It keeps only .env (the endpoint URLs) so the app stays usable for a
// fresh login. It is idempotent: a clean slate is not an error.
func Clear() error {
	var firstErr error
	keep := func(err error) {
		if err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = err
		}
	}
	if dir, err := config.ConfigDir(); err == nil {
		entries, rerr := os.ReadDir(dir)
		keep(rerr)
		for _, e := range entries {
			if e.Name() == ".env" {
				continue // keep deployment endpoints
			}
			keep(os.RemoveAll(filepath.Join(dir, e.Name())))
		}
	} else {
		keep(err)
	}
	if authDir, err := filepath.Abs(".auth"); err == nil {
		keep(os.RemoveAll(authDir)) // playwright profile + state
	}
	return firstErr
}

// Login is the CLI flow (`nybble auth`): open the browser, wait for the user to
// finish at the terminal, then finalize. The TUI drives the same steps without
// blocking on stdin (see internal/tui/reauth.go).
func Login() error {
	if err := OpenBrowser(); err != nil {
		return err
	}
	fmt.Println("Opening browser — scan the SSO QR, wait for the canteen menu, then press Enter here.")
	fmt.Scanln()

	eps, err := config.LoadEndpoints()
	if err != nil {
		CloseBrowser()
		return err
	}
	detErr, err := Finalize(eps.APIBase)
	if err != nil {
		return err
	}
	dir, _ := config.ConfigDir()
	if detErr != nil {
		fmt.Fprintln(os.Stderr, "note: couldn't auto-detect your building ("+detErr.Error()+
			") — re-run `nybble auth` and open your canteen menu so it can detect it")
	} else {
		fmt.Println("✓ Detected your building →", filepath.Join(dir, "config.json"))
	}
	fmt.Println("✓ Logged in. Cookies saved to", filepath.Join(dir, "cookies.json"))
	return nil
}
