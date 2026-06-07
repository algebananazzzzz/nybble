package session

import (
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/algebananazzzzz/nybble/internal/api"
)

var ErrAuthExpired = errors.New("auth expired: run `nybble auth`")

// ClientFor builds an api.Client whose http.Client carries the cookie store.
func ClientFor(store CookieStore, base string, hc *http.Client) *api.Client {
	if hc == nil {
		hc = &http.Client{}
	}
	if hc.Timeout == 0 {
		hc.Timeout = api.RequestTimeout // never let a stalled request hang the run
	}
	jar, _ := cookiejar.New(nil)
	if u, err := url.Parse(base); err == nil {
		jar.SetCookies(u, store.HTTP())
		// Also scope cookies to the origin root so any path on the host authenticates.
		if u.Scheme != "" && u.Host != "" {
			root := &url.URL{Scheme: u.Scheme, Host: u.Host, Path: "/"}
			jar.SetCookies(root, store.HTTP())
		}
	}
	hc.Jar = jar
	return api.New(base, hc)
}

// Valid reports whether the stored session still authenticates (user_info code 200)
// against the given API base.
func Valid(store CookieStore, base string) bool {
	c := ClientFor(store, base, &http.Client{})
	out, err := c.UserInfo()
	if err != nil {
		return false
	}
	return out["code"] == float64(200) || out["code"] == "200"
}

// Refresh validates the stored cookies against the given API base. The long-lived
// SSO cookies authenticate the API directly, so a valid store is returned as-is; an
// invalid one yields ErrAuthExpired so the caller can trigger a browser re-auth.
func Refresh(store CookieStore, base string) (CookieStore, error) {
	if Valid(store, base) {
		return store, nil
	}
	return nil, ErrAuthExpired
}
