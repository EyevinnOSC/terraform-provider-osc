package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// fastReads makes readConfirmed retry without waiting, and gives requests a short timeout.
func fastReads(t *testing.T) {
	t.Helper()
	attempts, interval, client := readAttempts, readInterval, httpClient
	readAttempts, readInterval = 3, time.Millisecond
	httpClient = &http.Client{Timeout: 200 * time.Millisecond}
	t.Cleanup(func() { readAttempts, readInterval, httpClient = attempts, interval, client })
}

// failure is a response OSC has been seen to answer with while the resource exists.
type failure struct {
	name    string
	handler http.HandlerFunc
}

func failures() []failure {
	status := func(code int, contentType, body string) http.HandlerFunc {
		return func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", contentType)
			w.WriteHeader(code)
			_, _ = w.Write([]byte(body))
		}
	}
	return []failure{
		{"401", status(http.StatusUnauthorized, "application/json", `{"message":"Unauthorized"}`)},
		{"500", status(http.StatusInternalServerError, "application/json", `{"message":"Internal Server Error"}`)},
		{"502 html", status(http.StatusBadGateway, "text/html", `<html><body>Bad Gateway</body></html>`)},
		{"503", status(http.StatusServiceUnavailable, "text/plain", `Service Unavailable`)},
		{"200 html", status(http.StatusOK, "text/html", `<html><body>Please wait</body></html>`)},
		{"404 html", status(http.StatusNotFound, "text/html", `<html><body>404 Not Found</body></html>`)},
		{"404 route", status(http.StatusNotFound, "application/json",
			`{"message":"Route GET:/api/v1/config/K not found","error":"Not Found","statusCode":404}`)},
		{"timeout", func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(time.Second):
			case <-r.Context().Done():
			}
		}},
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"reason":"Not found"}`))
}

func TestIsGone(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{&apiError{StatusCode: 404, Body: `{"reason":"Instance not found"}`}, true},
		{&apiError{StatusCode: 404, Body: ``}, true},
		{&apiError{StatusCode: 404, Body: `{"message":"Route POST:/api/v1/config not found","statusCode":404}`}, false},
		{&apiError{StatusCode: 404, Body: "  <!DOCTYPE html><html></html>"}, false},
		{&apiError{StatusCode: 401, Body: `{"message":"Unauthorized"}`}, false},
		{&apiError{StatusCode: 500}, false},
	}
	for _, c := range cases {
		if got := isGone(c.err); got != c.want {
			t.Errorf("isGone(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

func TestReadConfirmed(t *testing.T) {
	fastReads(t)
	sequence := func(steps ...func() (int, bool, error)) (func() (int, bool, error), *int) {
		calls := 0
		return func() (int, bool, error) {
			step := steps[len(steps)-1]
			if calls < len(steps) {
				step = steps[calls]
			}
			calls++
			return step()
		}, &calls
	}
	present := func() (int, bool, error) { return 7, true, nil }
	absent := func() (int, bool, error) { return 0, false, nil }
	failing := func() (int, bool, error) { return 0, false, &apiError{StatusCode: 503} }

	// A blip, then the resource.
	read, _ := sequence(failing, absent, present)
	if v, ok, err := readConfirmed(read); err != nil || !ok || v != 7 {
		t.Errorf("recovers: got %v %v %v", v, ok, err)
	}
	// Gone only when every attempt says so.
	read, calls := sequence(absent)
	if _, ok, err := readConfirmed(read); err != nil || ok || *calls != readAttempts {
		t.Errorf("gone: got %v %v after %d calls", ok, err, *calls)
	}
	// Missing once and failing otherwise is not proof it is gone.
	read, _ = sequence(absent, failing, failing)
	if _, ok, err := readConfirmed(read); err == nil || ok {
		t.Errorf("mixed: got %v %v", ok, err)
	}
}

// TestGetInstanceFailures reads an instance through the service API while it answers
// with each failure: every one must be an error, never "does not exist".
func TestGetInstanceFailures(t *testing.T) {
	fastReads(t)
	for _, f := range failures() {
		t.Run(f.name, func(t *testing.T) {
			srv := httptest.NewServer(f.handler)
			defer srv.Close()
			service := &catalogService{ServiceId: "minio-minio", ApiUrl: srv.URL + "/instances"}
			_, ok, err := readConfirmed(func() (map[string]interface{}, bool, error) {
				instance, err := getInstance(service, "shop", "tok")
				return instance, err == nil && instance != nil, err
			})
			if err == nil || ok {
				t.Errorf("got found=%v err=%v, want an error", ok, err)
			}
		})
	}

	srv := httptest.NewServer(http.HandlerFunc(notFound))
	defer srv.Close()
	instance, err := getInstance(&catalogService{ApiUrl: srv.URL + "/instances"}, "shop", "tok")
	if err != nil || instance != nil {
		t.Errorf("a JSON 404 means gone: got %v %v", instance, err)
	}
}

func TestParameterGetFailures(t *testing.T) {
	fastReads(t)
	for _, f := range failures() {
		t.Run(f.name, func(t *testing.T) {
			srv := httptest.NewServer(f.handler)
			defer srv.Close()
			c := &parameterClient{baseURL: srv.URL + "/api/v1/config", token: "tok"}
			if obj, err := c.get("DATABASE_URL"); err == nil {
				t.Errorf("got %+v, want an error", obj)
			}
		})
	}
}

// TestParameterWriteWaitsForRoutes is a store that has just been created: its API answers
// 404 for its routes for a while before it starts.
func TestParameterWriteWaitsForRoutes(t *testing.T) {
	var started atomic.Bool
	var posted atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if !started.Load() {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Route ` + r.Method + `:` + r.URL.Path + ` not found","error":"Not Found","statusCode":404}`))
			return
		}
		switch r.Method {
		case http.MethodGet:
			notFound(w, r)
		case http.MethodPost:
			posted.Add(1)
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	defer srv.Close()
	time.AfterFunc(200*time.Millisecond, func() { started.Store(true) })

	c := &parameterClient{baseURL: srv.URL + "/api/v1/config", token: "tok", retryFor: 30 * time.Second}
	if err := c.ready(); err != nil {
		t.Fatalf("ready: %v", err)
	}
	if err := c.put("DATABASE_URL", "postgres://", false); err != nil {
		t.Fatalf("put: %v", err)
	}
	if posted.Load() != 1 {
		t.Errorf("posted %d times", posted.Load())
	}
}

// deployManager points the deploy API at handler for the duration of the test.
func deployManager(t *testing.T, handler http.Handler) *osaasclient.Context {
	t.Helper()
	srv := httptest.NewServer(handler)
	base := deployBase
	deployBase = func(string) string { return srv.URL }
	t.Cleanup(func() { deployBase = base; srv.Close() })
	return &osaasclient.Context{PersonalAccessToken: "pat", Environment: "test"}
}

// readState runs a resource's Read on a state holding only the given attributes, and
// returns the response.
func readState(t *testing.T, r resource.Resource, attrs map[string]string) *resource.ReadResponse {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	state := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
	for k, v := range attrs {
		if d := state.SetAttribute(ctx, path.Root(k), v); d.HasError() {
			t.Fatalf("set %s: %v", k, d)
		}
	}
	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	return resp
}

// TestResourceReadKeepsStateOnFailure is F27: a Read while OSC fails must fail the plan,
// not drop the resource from state so that the plan recreates it.
func TestResourceReadKeepsStateOnFailure(t *testing.T) {
	fastReads(t)
	resources := []struct {
		name  string
		new   func(*osaasclient.Context) resource.Resource
		attrs map[string]string
	}{
		{"my_app", func(c *osaasclient.Context) resource.Resource { return &MyAppResource{osaasContext: c} },
			map[string]string{"id": "shopdev", "name": "shopdev"}},
		{"my_page", func(c *osaasclient.Context) resource.Resource { return &MyPageResource{osaasContext: c} },
			map[string]string{"id": "docs", "name": "docs"}},
		{"mailbox", func(c *osaasclient.Context) resource.Resource { return &MailboxResource{osaasContext: c} },
			map[string]string{"id": "acme"}},
		{"domain", func(c *osaasclient.Context) resource.Resource { return &DomainResource{osaasContext: c} },
			map[string]string{"id": "x", "service_id": webRunnerServiceID, "instance_name": "shopdev", "domain": "shop.example.com"}},
	}
	for _, res := range resources {
		for _, f := range failures() {
			t.Run(res.name+"/"+f.name, func(t *testing.T) {
				resp := readState(t, res.new(deployManager(t, f.handler)), res.attrs)
				if resp.State.Raw.IsNull() {
					t.Fatal("the resource was removed from state")
				}
				if !resp.Diagnostics.HasError() {
					t.Fatal("no error reported")
				}
			})
		}
		t.Run(res.name+"/gone", func(t *testing.T) {
			var gone http.HandlerFunc = notFound
			if res.name == "domain" {
				gone = func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) }
			}
			resp := readState(t, res.new(deployManager(t, gone)), res.attrs)
			if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
				t.Fatalf("a resource that is gone should leave state quietly: %v", resp.Diagnostics)
			}
		})
	}
}

// TestMyAppReadThroughRebuild is a plan during a rebuild: the app is missing for a few
// requests and then back.
func TestMyAppReadThroughRebuild(t *testing.T) {
	fastReads(t)
	var calls atomic.Int32
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/mydomains") {
			_, _ = w.Write([]byte(`[]`))
			return
		}
		if calls.Add(1) < 3 {
			notFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"id":"shopdev","name":"shopdev","buildStatus":"running","appDns":"x.apps.osaas.io"}`))
	}))
	resp := readState(t, &MyAppResource{osaasContext: ctx}, map[string]string{"id": "shopdev", "name": "shopdev"})
	if resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
		t.Fatalf("app was dropped or errored: %v", resp.Diagnostics)
	}
}

func TestCreateInstanceRetrying(t *testing.T) {
	var posts atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodPost && posts.Add(1) < 3:
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"reason":"ORCHESTRATOR_UNAVAILABLE"}`))
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"name":"shop"}`))
		default:
			notFound(w, r)
		}
	}))
	defer srv.Close()
	service := &catalogService{ServiceId: "minio-minio", ApiUrl: srv.URL}
	body := map[string]interface{}{"name": "shop"}

	out, err := createInstanceRetrying(service, "tok", body, false, 10*time.Second, time.Millisecond)
	if err != nil || out["name"] != "shop" || posts.Load() != 3 {
		t.Fatalf("got %v %v after %d posts", out, err, posts.Load())
	}

	// A rejection is returned at once, and is the only error the parameter guide follows.
	rejecting := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"reason":"RootPassword is required"}`))
	}))
	defer rejecting.Close()
	_, err = createInstanceRetrying(&catalogService{ApiUrl: rejecting.URL}, "tok", body, false, 10*time.Second, time.Millisecond)
	if err == nil || !isRejection(err) {
		t.Fatalf("rejection: %v", err)
	}
	if isRejection(&apiError{StatusCode: 503, Body: `{"reason":"ORCHESTRATOR_UNAVAILABLE"}`}) {
		t.Error("an unavailable service is not a rejection")
	}
	if _, err := httpClient.Get("https://" + strings.TrimPrefix(srv.URL, "http://")); err == nil || !isUnavailable(err) {
		t.Errorf("a TLS failure is unavailable: %v", err)
	}
}
