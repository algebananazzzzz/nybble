package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetMenuSendsStaticHeaders(t *testing.T) {
	var gotHdr http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHdr = r.Header
		w.Write([]byte(`{"code":200,"data":{"menuSites":[]}}`))
	}))
	defer srv.Close()

	c := New(srv.URL, srv.Client())
	_, err := c.Menu("BLDG00000001", "2026-06-10", "lunch")
	if err != nil {
		t.Fatal(err)
	}
	if gotHdr.Get("x-client-type") != "h5" {
		t.Fatalf("missing x-client-type header: %v", gotHdr)
	}
}
