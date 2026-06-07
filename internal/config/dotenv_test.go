package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotenv(t *testing.T) {
	in := strings.NewReader(`
# a comment
NYBBLE_API_BASE=https://host.example.com/app
export NYBBLE_LOGIN_URL="https://host.example.com/login"
QUOTED='single'
  SPACED  =  value
NOVALUE
EMPTY=
`)
	got := parseDotenv(in)
	want := map[string]string{
		"NYBBLE_API_BASE":  "https://host.example.com/app",
		"NYBBLE_LOGIN_URL": "https://host.example.com/login",
		"QUOTED":           "single",
		"SPACED":           "value",
		"EMPTY":            "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["NOVALUE"]; ok {
		t.Error("line without '=' should be skipped")
	}
}

func TestLoadDotenvDoesNotOverrideRealEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("NYBBLE_DOTENV_KEEP=fromfile\nNYBBLE_DOTENV_NEW=fromfile\n"), 0o600)
	t.Setenv("NYBBLE_ENV_FILE", envFile)

	// Already-set var must win over the file.
	t.Setenv("NYBBLE_DOTENV_KEEP", "fromenv")
	os.Unsetenv("NYBBLE_DOTENV_NEW")
	defer os.Unsetenv("NYBBLE_DOTENV_NEW")

	LoadDotenv()

	if got := os.Getenv("NYBBLE_DOTENV_KEEP"); got != "fromenv" {
		t.Errorf("real env overridden: got %q, want fromenv", got)
	}
	if got := os.Getenv("NYBBLE_DOTENV_NEW"); got != "fromfile" {
		t.Errorf("unset var not loaded from file: got %q", got)
	}
}
