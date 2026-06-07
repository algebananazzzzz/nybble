package session

import (
	"encoding/json"
	"net/http"
	"os"
)

type Cookie struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Domain string `json:"domain"`
	Path   string `json:"path"`
}
type CookieStore []Cookie

func LoadCookies(path string) (CookieStore, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s CookieStore
	return s, json.Unmarshal(raw, &s)
}

func (s CookieStore) Save(path string) error {
	raw, _ := json.MarshalIndent(s, "", "  ")
	return os.WriteFile(path, raw, 0o600)
}

func (s CookieStore) HTTP() []*http.Cookie {
	var out []*http.Cookie
	for _, c := range s {
		out = append(out, &http.Cookie{Name: c.Name, Value: sanitizeCookieValue(c.Value), Path: "/"})
	}
	return out
}

// sanitizeCookieValue makes a stored value safe for net/http, which (stricter than
// RFC 6265) rejects '"', ';' and '\\' in a cookie value: it drops them on every
// request while logging `net/http: invalid byte ... in Cookie.Value; dropping
// invalid bytes`, and that log corrupts the alt-screen TUI. Some providers store
// the value as an RFC 6265 quoted-string (literal surrounding quotes, e.g.
// LinkedIn's bcookie/lidc); the quotes are delimiters, not data, so we unwrap them.
// Any remaining forbidden byte is dropped here — the same result net/http would
// reach, minus the per-request log line.
func sanitizeCookieValue(v string) string {
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		v = v[1 : len(v)-1]
	}
	ok := true
	for i := 0; i < len(v); i++ {
		if !validCookieValueByte(v[i]) {
			ok = false
			break
		}
	}
	if ok {
		return v
	}
	b := make([]byte, 0, len(v))
	for i := 0; i < len(v); i++ {
		if validCookieValueByte(v[i]) {
			b = append(b, v[i])
		}
	}
	return string(b)
}

// validCookieValueByte mirrors net/http's own predicate so HTTP() drops exactly
// the bytes net/http would have dropped (just without the warning).
func validCookieValueByte(b byte) bool {
	return 0x20 <= b && b < 0x7f && b != '"' && b != ';' && b != '\\'
}

// FromPlaywrightState parses a playwright-cli `state-save` JSON into a CookieStore.
func FromPlaywrightState(path string) (CookieStore, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var st struct {
		Cookies []Cookie `json:"cookies"`
	}
	if err := json.Unmarshal(raw, &st); err != nil {
		return nil, err
	}
	return CookieStore(st.Cookies), nil
}
