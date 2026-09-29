package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	osaasclient "github.com/EyevinnOSC/client-go"
)

func TestSuspendedRequests(t *testing.T) {
	var resumed, discarded atomic.Int32
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-pat-jwt") != "Bearer pat" {
			t.Errorf("%s %s: missing PAT", r.Method, r.URL.Path)
		}
		switch r.Method + " " + r.URL.Path {
		case "GET /mysuspended":
			_, _ = w.Write([]byte(`[{"serviceId":"encore","instanceName":"other","suspendedAt":"2026-09-29T08:00:00Z","parameters":{}},
				{"serviceId":"encore","instanceName":"ivydev","suspendedAt":"2026-09-29T08:00:00Z","reason":"idle",
				 "parameters":{"name":"ivydev","s3Endpoint":"https://minio","s3SecretAccessKey":"{{secrets.k}}"}}]`))
		case "POST /mysuspended/encore/ivydev/resume":
			resumed.Add(1)
			_, _ = w.Write([]byte(`{"serviceId":"encore","instanceName":"ivydev","tenantId":"t","resumedAt":"now"}`))
		case "DELETE /mysuspended/encore/ivydev":
			discarded.Add(1)
			_, _ = w.Write([]byte(`{"message":"ok"}`))
		case "DELETE /mysuspended/encore/running":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"reason":"not suspended"}`))
		default:
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusBadRequest)
		}
	}))

	s, err := findSuspended(ctx, "encore", "ivydev")
	if err != nil || s == nil || s.Reason != "idle" || s.Parameters["s3Endpoint"] != "https://minio" {
		t.Fatalf("findSuspended = %+v, %v", s, err)
	}
	if s, err := findSuspended(ctx, "encore", "running"); err != nil || s != nil {
		t.Errorf("running instance: %+v, %v", s, err)
	}
	if err := resumeSuspended(ctx, "encore", "ivydev"); err != nil || resumed.Load() != 1 {
		t.Errorf("resume: %v, %d calls", err, resumed.Load())
	}
	if err := discardSuspended(ctx, "encore", "ivydev"); err != nil || discarded.Load() != 1 {
		t.Errorf("discard: %v, %d calls", err, discarded.Load())
	}
	if err := discardSuspended(ctx, "encore", "running"); err != nil {
		t.Errorf("discarding an instance that is not suspended: %v", err)
	}

	service := &catalogService{ServiceId: "encore", ServiceInstanceOptions: []serviceOption{
		{Name: "name"}, {Name: "s3Endpoint"}, {Name: "s3SecretAccessKey", Sensitive: true},
	}}
	model := modelFromSuspended(service, "ivydev", s)
	if !model.Suspended.ValueBool() || model.ID.ValueString() != "encore/ivydev" ||
		!model.Parameters.Equal(stringsToMap(map[string]string{"s3Endpoint": "https://minio"})) ||
		!model.SensitiveParameters.Equal(stringsToMap(map[string]string{"s3SecretAccessKey": "{{secrets.k}}"})) {
		t.Errorf("imported model = %+v", model)
	}
}

// TestWaitForResumed covers the minute after a resume in which the service API answers
// 401 for the instance.
func TestWaitForResumed(t *testing.T) {
	interval := resumePollInterval
	resumePollInterval = time.Millisecond
	t.Cleanup(func() { resumePollInterval = interval })

	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`<html>401 Authorization Required</html>`))
		case 2:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
		default:
			_, _ = w.Write([]byte(`{"name":"ivydev","url":"https://x"}`))
		}
	}))
	defer srv.Close()
	service := &catalogService{ServiceId: "encore", ApiUrl: srv.URL + "/encoreinstance"}

	instance, err := waitForResumed(service, "ivydev", "sat", time.Second)
	if err != nil || instance["url"] != "https://x" || calls.Load() != 3 {
		t.Errorf("instance %v, err %v after %d calls", instance, err, calls.Load())
	}
}

func TestIsStateful(t *testing.T) {
	cases := map[*catalogService]bool{
		{ServiceId: "minio-minio"}: true,
		{ServiceId: "some-db"}:     false,
		{ServiceId: "encore"}:      false,
		{ServiceId: "x-postgres", Metadata: osaasclient.ServiceMetadata{Category: "Database"}}: true,
	}
	for s, want := range cases {
		if got := isStateful(s); got != want {
			t.Errorf("%s: stateful = %v, want %v", s.ServiceId, got, want)
		}
	}
}

// TestPlanSuspended runs the plan logic for the suspended attribute on real plan and
// state values.
func TestPlanSuspended(t *testing.T) {
	ctx := context.Background()
	r := &InstanceResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)

	base := func() InstanceResourceModel {
		return InstanceResourceModel{
			ID:                  types.StringValue("encore/ivydev"),
			ServiceID:           types.StringValue("encore"),
			Name:                types.StringValue("ivydev"),
			Parameters:          stringsToMap(map[string]string{"s3Endpoint": "https://minio"}),
			SensitiveParameters: types.MapNull(types.StringType),
			AsSecrets:           types.BoolValue(true),
			SecretNames:         stringsToMap(nil),
			WaitForReady:        types.BoolValue(true),
			AllowSuspend:        types.BoolValue(true),
			Suspended:           types.BoolValue(true),
			UseLatest:           types.BoolValue(false),
			URL:                 types.StringValue("https://ludde-ivydev.encore.auto.prod-se.osaas.io"),
			ExternalIP:          types.StringValue(""),
			ExternalPort:        types.Int64Value(0),
			Instance:            stringsToMap(nil),
		}
	}
	run := func(state *InstanceResourceModel, plan InstanceResourceModel) (*resource.ModifyPlanResponse, InstanceResourceModel) {
		t.Helper()
		st := tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
		if state != nil {
			if d := st.Set(ctx, state); d.HasError() {
				t.Fatalf("state: %v", d)
			}
		}
		pl := tfsdk.Plan{Schema: sr.Schema}
		if d := pl.Set(ctx, &plan); d.HasError() {
			t.Fatalf("plan: %v", d)
		}
		resp := &resource.ModifyPlanResponse{Plan: pl}
		resp.Diagnostics.Append(r.planSuspended(ctx, resource.ModifyPlanRequest{State: st, Plan: pl}, resp, plan)...)
		if resp.Diagnostics.HasError() {
			t.Fatalf("planSuspended: %v", resp.Diagnostics)
		}
		var out InstanceResourceModel
		resp.Plan.Get(ctx, &out)
		return resp, out
	}

	t.Run("allowed and unchanged stays suspended", func(t *testing.T) {
		state := base()
		resp, out := run(&state, base())
		if !out.Suspended.ValueBool() || out.ExternalIP.IsUnknown() || resp.Diagnostics.WarningsCount() != 0 {
			t.Errorf("suspended %v, external_ip %v, %v", out.Suspended, out.ExternalIP, resp.Diagnostics)
		}
	})
	t.Run("not allowed resumes", func(t *testing.T) {
		state := base()
		plan := base()
		plan.AllowSuspend = types.BoolValue(false)
		_, out := run(&state, plan)
		if out.Suspended.ValueBool() || !out.ExternalPort.IsUnknown() {
			t.Errorf("suspended %v, external_port %v", out.Suspended, out.ExternalPort)
		}
	})
	t.Run("parameter change resumes with a warning", func(t *testing.T) {
		state := base()
		plan := base()
		plan.Parameters = stringsToMap(map[string]string{"s3Endpoint": "https://other"})
		resp, out := run(&state, plan)
		if out.Suspended.ValueBool() || resp.Diagnostics.WarningsCount() != 1 {
			t.Errorf("suspended %v, %v", out.Suspended, resp.Diagnostics)
		}
	})
	t.Run("running stays running", func(t *testing.T) {
		state := base()
		state.Suspended = types.BoolValue(false)
		_, out := run(&state, base())
		if out.Suspended.ValueBool() {
			t.Error("a running instance was planned suspended")
		}
	})
	t.Run("create is not suspended", func(t *testing.T) {
		_, out := run(nil, base())
		if out.Suspended.ValueBool() {
			t.Error("a new instance was planned suspended")
		}
	})
}

// TestPatchAfterResume covers OSC applying the parameters it kept for a suspended
// instance after the instance is back, overwriting the update.
func TestPatchAfterResume(t *testing.T) {
	interval, settle := resumePollInterval, resumeSettle
	resumePollInterval, resumeSettle = time.Millisecond, 50*time.Millisecond
	t.Cleanup(func() { resumePollInterval, resumeSettle = interval, settle })

	var mu sync.Mutex
	region, patches, gets := "us-east-1", 0, 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPatch {
			patches++
			var body map[string]string
			_ = json.NewDecoder(r.Body).Decode(&body)
			region = body["s3Region"]
		} else {
			gets++
			if gets == 2 {
				region = "us-east-1" // the resume finishes and restores the kept parameters
			}
		}
		_, _ = w.Write([]byte(`{"name":"ivydev","s3Region":"` + region + `"}`))
	}))
	defer srv.Close()
	service := &catalogService{ServiceId: "encore", ApiUrl: srv.URL + "/encoreinstance"}

	_, err := patchAfterResume(service, "ivydev", "sat", map[string]interface{}{"s3Region": "eu-north-1"}, time.Now())
	if err != nil || region != "eu-north-1" || patches != 2 {
		t.Errorf("err %v, region %q after %d patches", err, region, patches)
	}
	if !bodyApplied(map[string]interface{}{"Debug": true, "Unechoed": "x"}, map[string]interface{}{"Debug": true}) {
		t.Error("values the service does not echo back count as not applied")
	}
}
