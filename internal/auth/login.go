package auth

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/algebananazzzzz/bytecanteen/internal/session"
)

// Login drives playwright-cli for a one-time QR login, then exports cookies.
func Login() error {
	eps, err := config.LoadEndpoints()
	if err != nil {
		return err
	}
	profile, err := filepath.Abs(".auth/profile")
	if err != nil {
		return err
	}
	state, err := filepath.Abs(".auth/state.json")
	if err != nil {
		return err
	}

	fmt.Println("Opening browser — scan the SSO QR, wait for the canteen menu, then press Enter here.")
	open := exec.Command("playwright-cli", "-s=canteen", "open", "--headed", "--persistent",
		"--profile="+profile, eps.LoginURL)
	open.Stdout, open.Stderr = os.Stdout, os.Stderr
	if err := open.Run(); err != nil {
		return fmt.Errorf("launch browser: %w", err)
	}
	fmt.Scanln()

	if err := exec.Command("playwright-cli", "-s=canteen", "state-save", state).Run(); err != nil {
		return fmt.Errorf("state-save: %w", err)
	}
	_ = exec.Command("playwright-cli", "-s=canteen", "close").Run()

	store, err := session.FromPlaywrightState(state)
	if err != nil {
		return err
	}
	dir, err := config.ConfigDir()
	if err != nil {
		return err
	}
	cookiePath := filepath.Join(dir, "cookies.json")
	if err := store.Save(cookiePath); err != nil {
		return err
	}
	_ = os.Remove(state)

	if !session.Valid(store, eps.APIBase) {
		return fmt.Errorf("login did not produce a valid session")
	}
	fmt.Println("✓ Logged in. Cookies saved to", cookiePath)
	return nil
}
