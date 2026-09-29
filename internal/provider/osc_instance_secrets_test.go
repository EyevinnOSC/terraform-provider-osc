package provider

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestInstanceSecretName(t *testing.T) {
	valid := regexp.MustCompile(`^[a-z0-9]+$`)
	name := instanceSecretName("eyevinn-encore-packager", "ivyproduction", "RedisUrl")
	if !valid.MatchString(name) || !strings.HasPrefix(name, "ivyproductionredisurl") {
		t.Errorf("name = %q, want lowercase alphanumeric starting with the instance and parameter", name)
	}
	if again := instanceSecretName("eyevinn-encore-packager", "ivyproduction", "RedisUrl"); again != name {
		t.Errorf("name is not deterministic: %q, then %q", name, again)
	}

	distinct := [][3]string{
		{"eyevinn-encore-packager", "ivyproduction", "RedisUrl"},
		{"eyevinn-encore-packager", "ivyproductionpreview", "RedisUrl"},
		{"eyevinn-encore-packager", "ivy", "PreviewKey"},
		{"eyevinn-encore-packager", "ivypreview", "Key"},
		{"eyevinn-encore-packager", "Prod", "Key"},
		{"eyevinn-encore-packager", "prod", "Key"},
		{"eyevinn-encore-packager", "my_app", "Key"},
		{"eyevinn-encore-packager", "myapp", "Key"},
		{"eyevinn-encore-transfer", "ivyproduction", "RedisUrl"},
	}
	seen := map[string][3]string{}
	for _, d := range distinct {
		n := instanceSecretName(d[0], d[1], d[2])
		if !valid.MatchString(n) {
			t.Errorf("%v: name %q is not lowercase alphanumeric", d, n)
		}
		if prev, ok := seen[n]; ok {
			t.Errorf("%v and %v share the secret name %q", prev, d, n)
		}
		seen[n] = d
	}

	long := instanceSecretName("svc", strings.Repeat("a", 60), "AwsSecretAccessKey")
	if len(long) != 48 {
		t.Errorf("long name %q has length %d, want 48", long, len(long))
	}
}

func TestDesiredSecretNames(t *testing.T) {
	sensitive := map[string]string{"RedisUrl": "redis://default:pw@host:6379", "PersonalAccessToken": "{{secrets.osctoken}}"}
	got := desiredSecretNames("eyevinn-encore-packager", "ivyproduction", sensitive, true)
	want := map[string]string{"RedisUrl": instanceSecretName("eyevinn-encore-packager", "ivyproduction", "RedisUrl")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("names = %v, want %v", got, want)
	}
	if got := desiredSecretNames("svc", "inst", sensitive, false); len(got) != 0 {
		t.Errorf("disabled: names = %v, want none", got)
	}

	sent := withSecretRefs(sensitive, want)
	wantSent := map[string]string{"RedisUrl": "{{secrets." + want["RedisUrl"] + "}}", "PersonalAccessToken": "{{secrets.osctoken}}"}
	if !reflect.DeepEqual(sent, wantSent) {
		t.Errorf("sent = %v, want %v", sent, wantSent)
	}
	if got := secretValues(sensitive, want); !reflect.DeepEqual(got, map[string]string{want["RedisUrl"]: sensitive["RedisUrl"]}) {
		t.Errorf("secret values = %v", got)
	}
}

func TestBuildInstanceBodyKeepsSecretRefs(t *testing.T) {
	body := buildInstanceBody(testService(), "inst", nil, map[string]string{"Debug": "{{secrets.flag}}"})
	if body["Debug"] != "{{secrets.flag}}" {
		t.Errorf("boolean parameter with a secret reference = %#v, want the reference", body["Debug"])
	}
}

func TestPlannedSecretNames(t *testing.T) {
	plan := InstanceResourceModel{
		ServiceID:           types.StringValue("svc"),
		Name:                types.StringValue("inst"),
		AsSecrets:           types.BoolValue(true),
		SensitiveParameters: stringsToMap(map[string]string{"Password": "pw"}),
	}
	want := stringsToMap(map[string]string{"Password": instanceSecretName("svc", "inst", "Password")})
	if got := plannedSecretNames(plan); !got.Equal(want) {
		t.Errorf("names = %v, want %v", got, want)
	}

	// State written by a provider version without secrets has no secret_names; the plan
	// then differs from state, which is what migrates the instance.
	plan.SensitiveParameters = types.MapNull(types.StringType)
	if got := plannedSecretNames(plan); !got.Equal(stringsToMap(nil)) {
		t.Errorf("no sensitive parameters: names = %v, want empty", got)
	}

	plan.SensitiveParameters = types.MapValueMust(types.StringType, map[string]attr.Value{"Password": types.StringUnknown()})
	if got := plannedSecretNames(plan); !got.IsUnknown() {
		t.Errorf("unknown value: names = %v, want unknown", got)
	}
	plan.SensitiveParameters = stringsToMap(map[string]string{"Password": "pw"})
	plan.Name = types.StringUnknown()
	if got := plannedSecretNames(plan); !got.IsUnknown() {
		t.Errorf("unknown name: names = %v, want unknown", got)
	}
}

func TestSecretDrift(t *testing.T) {
	names := map[string]string{"RedisUrl": "a1", "Password": "b2", "Token": "c3"}
	instance := map[string]interface{}{
		"RedisUrl": "redis://default:pw@host:6379", // stored in plain text
		"Password": "{{secrets.b2}}",
		// Token is not echoed back by the service.
	}
	if got := secretDrift(names, instance); !reflect.DeepEqual(got, []string{"RedisUrl"}) {
		t.Errorf("drift = %v, want [RedisUrl]", got)
	}
}

func TestMovesToSecrets(t *testing.T) {
	if movesToSecrets(map[string]string{"A": "a"}, map[string]string{"A": "a"}) {
		t.Error("unchanged names reported as a move")
	}
	if !movesToSecrets(nil, map[string]string{"A": "a"}) {
		t.Error("new secret not reported as a move")
	}
	if movesToSecrets(map[string]string{"A": "a"}, nil) {
		t.Error("removing secrets reported as a move")
	}
}

func TestSecretRequests(t *testing.T) {
	var mu sync.Mutex
	var batches []int
	var deleted []string
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Header.Get("x-pat-jwt") != "Bearer pat" {
			t.Errorf("%s %s: missing PAT", r.Method, r.URL.Path)
		}
		switch {
		case r.Method == http.MethodPut && r.URL.Path == "/mysecrets/svc/batch":
			var body struct {
				Secrets []struct {
					SecretName string `json:"secretName"`
					SecretData string `json:"secretData"`
				} `json:"secrets"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode batch: %v", err)
			}
			for _, s := range body.Secrets {
				if s.SecretData != "v"+s.SecretName {
					t.Errorf("secret %q has value %q", s.SecretName, s.SecretData)
				}
			}
			batches = append(batches, len(body.Secrets))
		case r.Method == http.MethodDelete && r.URL.Path == "/mysecrets/svc/gone":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		case r.Method == http.MethodDelete:
			deleted = append(deleted, strings.TrimPrefix(r.URL.Path, "/mysecrets/svc/"))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	secrets := map[string]string{}
	for i := 0; i < 23; i++ {
		n := "s" + strings.Repeat("x", i)
		secrets[n] = "v" + n
	}
	if err := putSecrets(ctx, "svc", secrets); err != nil {
		t.Fatalf("putSecrets: %v", err)
	}
	if !reflect.DeepEqual(batches, []int{10, 10, 3}) {
		t.Errorf("batches = %v, want [10 10 3]", batches)
	}
	if err := putSecrets(ctx, "svc", nil); err != nil || len(batches) != 3 {
		t.Errorf("no secrets: err %v, %d batches", err, len(batches))
	}

	if err := deleteSecret(ctx, "svc", "gone"); err != nil {
		t.Errorf("deleting a missing secret: %v", err)
	}
	r := &InstanceResource{osaasContext: ctx}
	diags := r.deleteSecrets("svc", map[string]string{"A": "old", "B": "kept"}, map[string]string{"B": "kept"})
	if diags.HasError() || !reflect.DeepEqual(deleted, []string{"old"}) {
		t.Errorf("deleted = %v (%v), want [old]", deleted, diags)
	}
}

func TestRestartInstance(t *testing.T) {
	var restarted string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get("x-jwt") == "Bearer sat" {
			restarted = r.URL.Path
		}
	}))
	defer srv.Close()
	client := httpClient
	httpClient = srv.Client()
	defer func() { httpClient = client }()

	service := &catalogService{ServiceId: "svc", ApiUrl: srv.URL + "/svcinstance"}
	instance := map[string]interface{}{"_links": map[string]interface{}{"restart": map[string]interface{}{"href": "/restart/inst"}}}
	ok, err := restartInstance(service, "sat", instance)
	if err != nil || !ok || restarted != "/restart/inst" {
		t.Errorf("restart: ok %v, err %v, path %q", ok, err, restarted)
	}
	if ok, err := restartInstance(service, "sat", map[string]interface{}{}); ok || err != nil {
		t.Errorf("no restart link: ok %v, err %v", ok, err)
	}
}
