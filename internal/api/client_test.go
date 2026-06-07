package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCurrentBuilding(t *testing.T) {
	// The preferences endpoint double-encodes: data is a JSON *string* holding the
	// building object. CurrentBuilding must unwrap both layers.
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Write([]byte(`{"code":200,"data":"{\"mdmCode\":\"MDBD00000485\",\"regionName\":\"Guoco Tower\"}","message":"success"}`))
	}))
	defer srv.Close()

	code, name, err := New(srv.URL, srv.Client()).CurrentBuilding()
	if err != nil {
		t.Fatal(err)
	}
	if code != "MDBD00000485" || name != "Guoco Tower" {
		t.Fatalf("got code=%q name=%q, want MDBD00000485 / Guoco Tower", code, name)
	}
	if gotPath != "/mini-program/user/v1/preferences?key=CURRENT_BUILDING_KEY" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
}

func TestCurrentBuildingEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"code":200,"data":"","message":"success"}`))
	}))
	defer srv.Close()

	if _, _, err := New(srv.URL, srv.Client()).CurrentBuilding(); err == nil {
		t.Fatal("empty data should error (no building selected)")
	}
}

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
