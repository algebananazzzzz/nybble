package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ensureLoginHelper is the dependency preflight for `nybble auth`: it must fail with
// an actionable, install-pointing error when playwright-cli isn't on PATH, rather than
// letting OpenBrowser surface a cryptic "executable file not found".
func TestEnsureLoginHelperMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty dir → no playwright-cli

	err := ensureLoginHelper()
	if err == nil {
		t.Fatal("ensureLoginHelper must error when playwright-cli is absent")
	}
	for _, want := range []string{loginHelper, "@playwright/cli"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should mention %q so the user knows how to fix it", err, want)
		}
	}
}

func TestEnsureLoginHelperPresent(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, loginHelper)
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)

	if err := ensureLoginHelper(); err != nil {
		t.Errorf("ensureLoginHelper should pass when %s is on PATH: %v", loginHelper, err)
	}
}
