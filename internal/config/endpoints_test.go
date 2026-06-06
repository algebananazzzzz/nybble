package config

import "testing"

func TestLoadEndpointsReturnsEnvValues(t *testing.T) {
	t.Setenv("CANTEEN_API_BASE", "https://host.example.com/app")
	t.Setenv("CANTEEN_LOGIN_URL", "https://host.example.com/login")

	got, err := LoadEndpoints()
	if err != nil {
		t.Fatalf("LoadEndpoints: %v", err)
	}
	if got.APIBase != "https://host.example.com/app" {
		t.Errorf("APIBase = %q", got.APIBase)
	}
	if got.LoginURL != "https://host.example.com/login" {
		t.Errorf("LoginURL = %q", got.LoginURL)
	}
}

func TestLoadEndpointsErrorsWhenUnset(t *testing.T) {
	cases := []struct {
		name, base, login string
	}{
		{"api base unset", "", "https://host.example.com/login"},
		{"login unset", "https://host.example.com/app", ""},
		{"both unset", "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv("CANTEEN_API_BASE", c.base)
			t.Setenv("CANTEEN_LOGIN_URL", c.login)
			if _, err := LoadEndpoints(); err == nil {
				t.Fatal("want error for unset endpoint, got nil")
			}
		})
	}
}
