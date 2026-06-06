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
		out = append(out, &http.Cookie{Name: c.Name, Value: c.Value, Path: "/"})
	}
	return out
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
