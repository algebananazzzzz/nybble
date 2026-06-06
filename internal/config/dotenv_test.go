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
CANTEEN_API_BASE=https://host.example.com/app
export CANTEEN_LOGIN_URL="https://host.example.com/login"
QUOTED='single'
  SPACED  =  value
NOVALUE
EMPTY=
`)
	got := parseDotenv(in)
	want := map[string]string{
		"CANTEEN_API_BASE":  "https://host.example.com/app",
		"CANTEEN_LOGIN_URL": "https://host.example.com/login",
		"QUOTED":            "single",
		"SPACED":            "value",
		"EMPTY":             "",
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
	os.WriteFile(envFile, []byte("CANTEEN_DOTENV_KEEP=fromfile\nCANTEEN_DOTENV_NEW=fromfile\n"), 0o600)
	t.Setenv("CANTEEN_ENV_FILE", envFile)

	// Already-set var must win over the file.
	t.Setenv("CANTEEN_DOTENV_KEEP", "fromenv")
	os.Unsetenv("CANTEEN_DOTENV_NEW")
	defer os.Unsetenv("CANTEEN_DOTENV_NEW")

	LoadDotenv()

	if got := os.Getenv("CANTEEN_DOTENV_KEEP"); got != "fromenv" {
		t.Errorf("real env overridden: got %q, want fromenv", got)
	}
	if got := os.Getenv("CANTEEN_DOTENV_NEW"); got != "fromfile" {
		t.Errorf("unset var not loaded from file: got %q", got)
	}
}
