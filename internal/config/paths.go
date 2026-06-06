package config

import (
	"os"
	"path/filepath"
)

// ConfigDir returns ~/.config/canteen (honoring XDG_CONFIG_HOME), creating it.
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "canteen")
	return dir, os.MkdirAll(dir, 0o700)
}
