package session

import (
	"bytes"
	"log"
	"path/filepath"
	"testing"
)

// Regression for the `net/http: invalid byte '"' in Cookie.Value; dropping invalid
// bytes` spam: http.Cookie.String() is the exact path net/http takes to serialize a
// cookie onto a request, and it log.Printf's that warning for any forbidden byte.
// After HTTP() sanitizes, serializing every cookie must emit nothing.
func TestHTTPCookiesEmitNoNetHTTPWarning(t *testing.T) {
	var buf bytes.Buffer
	prev := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(prev)

	s := CookieStore{
		{Name: "bcookie", Value: `"v=2&0a1b9dd"`},
		{Name: "lidc", Value: `"b=OGST00:s=abc"`},
		{Name: "session_id", Value: "clean123"},
	}
	for _, c := range s.HTTP() {
		_ = c.String()
	}
	if buf.Len() != 0 {
		t.Fatalf("net/http still warned on cookie serialization: %q", buf.String())
	}
}

// Some providers store their cookie as an RFC 6265 quoted-string (literal
// surrounding quotes, e.g. LinkedIn's bcookie/lidc). net/http forbids '"', ';'
// and '\\' in a cookie value and strips them on every request while logging
// `net/http: invalid byte '"' in Cookie.Value; dropping invalid bytes`, which in
// the alt-screen TUI paints over the frame. HTTP() must hand net/http only valid
// bytes so that warning never fires.
func TestHTTPSanitizesCookieValues(t *testing.T) {
	s := CookieStore{
		{Name: "session_id", Value: "abc123"},     // clean — unchanged
		{Name: "bcookie", Value: `"v=2&0a1b9dd"`}, // quoted-string — unwrap
		{Name: "weird", Value: "a\"b;c\\d"},       // embedded forbidden bytes — dropped
	}
	by := map[string]string{}
	for _, c := range s.HTTP() {
		by[c.Name] = c.Value
		for i := 0; i < len(c.Value); i++ {
			if b := c.Value[i]; b == '"' || b == ';' || b == '\\' || b < 0x20 || b >= 0x7f {
				t.Errorf("cookie %s value retains net/http-invalid byte %q: %q", c.Name, b, c.Value)
			}
		}
	}
	if by["session_id"] != "abc123" {
		t.Errorf("clean value altered: %q", by["session_id"])
	}
	if by["bcookie"] != "v=2&0a1b9dd" {
		t.Errorf("quoted-string not unwrapped: %q", by["bcookie"])
	}
}

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
