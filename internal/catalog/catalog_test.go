package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func testMirror() *Mirror {
	return &Mirror{Services: []Service{
		{ServiceID: "apache-couchdb"}, {ServiceID: "valkey-io-valkey"}, {ServiceID: "eyevinn-open-live"},
		{ServiceID: "eyevinn-open-live-studio"}, {ServiceID: "birme-osc-postgresql"},
	}}
}

func TestSuggest(t *testing.T) {
	m := testMirror()
	cases := map[string]string{
		"eyevinn-couchdb":  "apache-couchdb",
		"valkey":           "valkey-io-valkey",
		"open-live":        "eyevinn-open-live",
		"eyevinn-postgres": "birme-osc-postgresql",
	}
	for in, want := range cases {
		got := m.Suggest(in, 3)
		if len(got) == 0 || got[0] != want {
			t.Errorf("Suggest(%q) = %v, want first %q", in, got, want)
		}
	}
	if got := m.Suggest("zzzzzzzzzzzzzzzzzzzz", 3); len(got) != 0 {
		t.Errorf("expected no suggestions, got %v", got)
	}
}

func TestFetchAndFind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(testMirror())
	}))
	defer srv.Close()
	m, err := Fetch(context.Background(), srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if m.Find("valkey-io-valkey") == nil || m.Find("nope") != nil {
		t.Fatal("Find wrong")
	}
	if _, err := Fetch(context.Background(), srv.URL+"/missing"); err == nil {
		t.Fatal("expected error for non-200")
	}
}
