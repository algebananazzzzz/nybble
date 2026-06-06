package session

import (
	"path/filepath"
	"testing"
)

func TestCookieStoreRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "cookies.json")
	in := CookieStore{{Name: "session_id", Value: "abc", Domain: ".host.example.com", Path: "/"}}
	if err := in.Save(p); err != nil {
		t.Fatal(err)
	}
	out, err := LoadCookies(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Value != "abc" {
		t.Fatalf("round trip failed: %+v", out)
	}
	if len(out.HTTP()) != 1 || out.HTTP()[0].Name != "session_id" {
		t.Fatal("HTTP() conversion failed")
	}
}
