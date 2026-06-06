package session

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthedClientSendsCookies(t *testing.T) {
	var gotCookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCookie = r.Header.Get("Cookie")
		w.Write([]byte(`{"code":200}`))
	}))
	defer srv.Close()

	store := CookieStore{{Name: "session_id", Value: "abc"}}
	c := ClientFor(store, srv.URL, srv.Client())
	_, _ = c.UserInfo()
	if !strings.Contains(gotCookie, "session_id=abc") {
		t.Fatalf("cookie not sent: %q", gotCookie)
	}
}
