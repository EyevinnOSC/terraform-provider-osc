package provider

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestMyAppSourceURL(t *testing.T) {
	cases := []struct {
		app  myApp
		want string
	}{
		{myApp{GitURL: "https://github.com/acme/api#main"}, "https://github.com/acme/api"},
		{myApp{GitHubURL: "https://github.com/acme/api"}, "https://github.com/acme/api"},
		{myApp{GitURL: "https://gitea.example/acme/api.git#v1.2.0", GitHubURL: "ignored"}, "https://gitea.example/acme/api.git"},
	}
	for _, c := range cases {
		if got := c.app.sourceURL(); got != c.want {
			t.Errorf("sourceURL(%+v) = %q, want %q", c.app, got, c.want)
		}
	}
}

func TestMyAppBoundConfigService(t *testing.T) {
	if got := (&myApp{ConfigService: "undefined"}).boundConfigService(); got != "" {
		t.Errorf("undefined should read as unbound, got %q", got)
	}
	if got := (&myApp{ConfigService: "store"}).boundConfigService(); got != "store" {
		t.Errorf("got %q", got)
	}
}

func TestNormalizeGitURL(t *testing.T) {
	a := normalizeGitURL("https://github.com/Acme/API.git/")
	b := normalizeGitURL("https://github.com/acme/api")
	if a != b {
		t.Errorf("%q != %q", a, b)
	}
	if normalizeGitURL("https://github.com/acme/api") == normalizeGitURL("https://github.com/acme/web") {
		t.Error("different repositories normalized to the same URL")
	}
}

func TestMyPageIDAndDomain(t *testing.T) {
	if got := (&myPage{Name: "docs"}).id(); got != "docs" {
		t.Errorf("id falls back to name, got %q", got)
	}
	if got := (&myPage{ID: "p1", Name: "docs"}).id(); got != "p1" {
		t.Errorf("got %q", got)
	}
	if got := (&myPage{DomainAlt: "www.example.com"}).customDomain(); got != "www.example.com" {
		t.Errorf("got %q", got)
	}
}

func TestGitCredentialRef(t *testing.T) {
	if got := gitCredentialRef("github-acme"); got != "user.gitcred.github-acme" {
		t.Errorf("got %q", got)
	}
	if got := gitCredentialRef(""); got != "" {
		t.Errorf("empty name should give no ref, got %q", got)
	}
}

func TestApplyParameter(t *testing.T) {
	// Plain values are recorded as they are.
	m := ParameterResourceModel{Value: types.StringValue("old"), SecretValue: types.StringNull()}
	applyParameter(&parameterObject{Key: "K", Value: "new"}, false, &m)
	if m.Value.ValueString() != "new" || !m.SecretValue.IsNull() {
		t.Errorf("plain: %+v", m)
	}

	// A secret read without the API key comes back masked; keep the value in state.
	m = ParameterResourceModel{Value: types.StringNull(), SecretValue: types.StringValue("s3cret")}
	applyParameter(&parameterObject{Key: "K", Value: "********", Secret: true}, false, &m)
	if m.SecretValue.ValueString() != "s3cret" || !m.Value.IsNull() {
		t.Errorf("masked secret: %+v", m)
	}

	// With the key the plain text is authoritative, so drift is detected.
	applyParameter(&parameterObject{Key: "K", Value: "rotated", Secret: true}, true, &m)
	if m.SecretValue.ValueString() != "rotated" {
		t.Errorf("readable secret: %+v", m)
	}

	// A value that became a secret outside Terraform moves to secret_value.
	m = ParameterResourceModel{Value: types.StringValue("v"), SecretValue: types.StringNull()}
	applyParameter(&parameterObject{Key: "K", Value: "********", Secret: true}, false, &m)
	if !m.Value.IsNull() || m.SecretValue.IsNull() {
		t.Errorf("converted to secret: %+v", m)
	}
}

func TestParameterClientRequests(t *testing.T) {
	type call struct {
		method, path, apiKey, jwt, contentType string
		body                                   map[string]interface{}
	}
	var calls []call
	stored := map[string]bool{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c := call{method: r.Method, path: r.URL.Path, apiKey: r.Header.Get("x-config-api-key"),
			jwt: r.Header.Get("x-jwt"), contentType: r.Header.Get("Content-Type")}
		if b, _ := io.ReadAll(r.Body); len(b) > 0 {
			_ = json.Unmarshal(b, &c.body)
		}
		calls = append(calls, c)
		key := r.URL.Path[len("/api/v1/config"):]
		switch {
		case r.Method == http.MethodGet && !stored[key]:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"reason":"not found"}`))
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"key":"K","value":"v"}`))
		case r.Method == http.MethodPost:
			stored["/"+c.body["key"].(string)] = true
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()

	c := &parameterClient{baseURL: srv.URL + "/api/v1/config", token: "tok", apiKey: "key"}
	if err := c.put("K", "v", true); err != nil {
		t.Fatal(err)
	}
	if err := c.put("K", "v2", true); err != nil {
		t.Fatal(err)
	}
	if err := c.delete("K"); err != nil {
		t.Fatal(err)
	}

	want := []struct{ method, path string }{
		{"GET", "/api/v1/config/K"}, {"POST", "/api/v1/config"},
		{"GET", "/api/v1/config/K"}, {"PUT", "/api/v1/config/K"},
		{"DELETE", "/api/v1/config/K"},
	}
	if len(calls) != len(want) {
		t.Fatalf("got %d calls, want %d: %+v", len(calls), len(want), calls)
	}
	for i, w := range want {
		got := calls[i]
		if got.method != w.method || got.path != w.path {
			t.Errorf("call %d: got %s %s, want %s %s", i, got.method, got.path, w.method, w.path)
		}
		if got.jwt != "Bearer tok" || got.apiKey != "key" {
			t.Errorf("call %d: missing credentials: %+v", i, got)
		}
		// Fastify rejects an empty body declared as JSON, so only requests with a body
		// may carry the header.
		if (got.body != nil) != (got.contentType == "application/json") {
			t.Errorf("call %d: content type %q with body %v", i, got.contentType, got.body)
		}
	}
	if calls[1].body["secret"] != true || calls[1].body["key"] != "K" {
		t.Errorf("create body: %v", calls[1].body)
	}
	if calls[3].body["value"] != "v2" {
		t.Errorf("update body: %v", calls[3].body)
	}
}

func TestPollMyAppBuild(t *testing.T) {
	app := func(status string) *myApp { return &myApp{ID: "a1", BuildStatus: status} }
	sequence := func(steps ...*myApp) func() (*myApp, error) {
		i := 0
		return func() (*myApp, error) {
			if i >= len(steps) {
				return steps[len(steps)-1], nil
			}
			i++
			return steps[i-1], nil
		}
	}

	// A rebuild: the app is missing for a few polls, then builds and runs.
	got, err := pollMyAppBuild(sequence(app("building"), nil, nil, nil, app("building"), app("running")),
		"a1", time.Second, time.Millisecond, 100*time.Millisecond)
	if err != nil || got == nil || got.BuildStatus != "running" {
		t.Fatalf("rebuild with a gap: got %+v, %v", got, err)
	}

	// Missing for longer than allowed is an error.
	if _, err := pollMyAppBuild(sequence(app("building"), nil), "a1", time.Second, time.Millisecond, 20*time.Millisecond); err == nil ||
		!strings.Contains(err.Error(), "disappeared") {
		t.Fatalf("app gone for good: got %v", err)
	}

	// A failed build is reported at once.
	if _, err := pollMyAppBuild(sequence(app("building"), app("failed")), "a1", time.Second, time.Millisecond, time.Second); err == nil ||
		!strings.Contains(err.Error(), "failed") {
		t.Fatalf("failed build: got %v", err)
	}

	// The timeout still applies while the app is missing.
	if _, err := pollMyAppBuild(sequence(nil), "a1", 20*time.Millisecond, time.Millisecond, time.Second); err == nil ||
		!strings.Contains(err.Error(), "within") {
		t.Fatalf("timeout while missing: got %v", err)
	}
}
