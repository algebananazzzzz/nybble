package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClearWipesAllButEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cdir := filepath.Join(dir, "nybble")
	if err := os.MkdirAll(cdir, 0o700); err != nil {
		t.Fatal(err)
	}
	wiped := []string{"cookies.json", "config.json", "favorites.json", "catalog.json", "vendors.json", "nybble.log"}
	for _, n := range wiped {
		if err := os.WriteFile(filepath.Join(cdir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	env := filepath.Join(cdir, ".env")
	if err := os.WriteFile(env, []byte("NYBBLE_API_BASE=x"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Clear(); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	for _, n := range wiped {
		if _, err := os.Stat(filepath.Join(cdir, n)); !os.IsNotExist(err) {
			t.Errorf("%s should be removed by clean-slate clear", n)
		}
	}
	if _, err := os.Stat(env); err != nil {
		t.Errorf(".env (endpoints) must be kept: %v", err)
	}
	// Idempotent: a second clear on an already-clean slate is a no-op.
	if err := Clear(); err != nil {
		t.Fatalf("second Clear should be a no-op, got %v", err)
	}
}
