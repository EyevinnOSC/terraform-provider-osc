package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"terraform-provider-osc/internal/catalog"
)

func TestMirrorServiceAndSuggestions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(catalog.Mirror{Services: []catalog.Service{
			{ServiceID: "valkey-io-valkey", Title: "Valkey", ServiceType: "instance", Options: []catalog.Option{
				{Name: "name", Type: "string", Mandatory: true}, {Name: "Password", Type: "string", Sensitive: true},
			}},
			{ServiceID: "apache-couchdb", ServiceType: "instance"},
		}})
	}))
	defer srv.Close()
	t.Setenv(catalog.MirrorURLEnv, srv.URL)
	resetMirrorForTest()
	t.Cleanup(resetMirrorForTest)

	svc, suggestions, err := mirrorService(context.Background(), "valkey-io-valkey")
	if err != nil || svc == nil || len(suggestions) != 0 {
		t.Fatalf("expected service, got %v %v %v", svc, suggestions, err)
	}
	if svc.Metadata.Title != "Valkey" || len(svc.ServiceInstanceOptions) != 2 || findOption(svc, "Password") == nil {
		t.Fatalf("conversion lost data: %+v", svc)
	}

	svc, suggestions, err = mirrorService(context.Background(), "eyevinn-couchdb")
	if err != nil || svc != nil || len(suggestions) == 0 || suggestions[0] != "apache-couchdb" {
		t.Fatalf("expected suggestion apache-couchdb, got %v %v %v", svc, suggestions, err)
	}
	detail := unknownServiceDetail("eyevinn-couchdb", suggestions, nil)
	if !strings.Contains(detail, "apache-couchdb") || !strings.Contains(detail, catalog.GuidesBaseURL) {
		t.Fatalf("detail missing suggestion or guide link: %s", detail)
	}
}
