package config

import (
	"fmt"
	"os"
)

// Endpoints holds the deployment-specific URLs for the canteen ordering portal.
// They are read from the environment so no real host is baked into source.
type Endpoints struct {
	APIBase  string // NYBBLE_API_BASE  — base for the order API, e.g. https://host/app
	LoginURL string // NYBBLE_LOGIN_URL — page opened for interactive SSO login
}

// LoadEndpoints reads endpoint configuration from the environment. Both
// NYBBLE_API_BASE and NYBBLE_LOGIN_URL are required; an unset or empty value
// is a hard error so a misconfigured run fails loudly instead of falling back to
// a hardcoded host.
func LoadEndpoints() (Endpoints, error) {
	base := os.Getenv("NYBBLE_API_BASE")
	if base == "" {
		return Endpoints{}, fmt.Errorf(`NYBBLE_API_BASE not set — see README "Configuration"`)
	}
	login := os.Getenv("NYBBLE_LOGIN_URL")
	if login == "" {
		return Endpoints{}, fmt.Errorf(`NYBBLE_LOGIN_URL not set — see README "Configuration"`)
	}
	return Endpoints{APIBase: base, LoginURL: login}, nil
}
