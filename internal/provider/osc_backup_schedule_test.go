package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestInvalidCron(t *testing.T) {
	valid := []string{"0 2 * * *", "*/15 * * * *", "0 3 * * 1-5", "0 3 * * 0", "30 1 1,15 * *", "0 0-23/6 * * *", "59 23 31 12 6"}
	for _, c := range valid {
		if msg := invalidCron(c); msg != "" {
			t.Errorf("%q: %s", c, msg)
		}
	}
	invalid := []string{"", "garbage", "@daily", "0 2 * * * *", "0 2 * *", "0 25 * * *", "60 * * * *", "0 2 0 * *",
		"0 2 * 13 *", "0 2 * * 7", "0 2 * * MON", "0 2 * JAN *", "5-1 * * * *", "*/0 * * * *", "0 2 * * * ", " 0 2 * * *",
		"CRON_TZ=Europe/Stockholm 0 2 * * *", "-1 * * * *", "+1 * * * *", "1- * * * *", ", * * * *"}
	for _, c := range invalid {
		if invalidCron(c) == "" {
			t.Errorf("%q was accepted", c)
		}
	}
}

func TestIsBackupService(t *testing.T) {
	if !isBackupService("birme-osc-postgresql") || !isBackupService("go-gitea-gitea") {
		t.Error("supported service rejected")
	}
	if isBackupService("minio-minio") || isBackupService("encore") {
		t.Error("unsupported service accepted")
	}
}

func TestFindBackupPolicy(t *testing.T) {
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mybackups/policies" {
			t.Errorf("unexpected %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{"serviceId":"valkey-io-valkey","instanceName":"db","enabled":false,"schedule":"0 2 * * *","retentionDays":30},
			{"serviceId":"birme-osc-postgresql","instanceName":"db","enabled":true,"schedule":"0 3 * * 0","retentionDays":14,
			 "lastBackupAt":1790000000,"lastError":"","credentialStatus":"ok"}
		]`))
	}))
	p, err := findBackupPolicy(ctx, "birme-osc-postgresql", "db")
	if err != nil || p == nil || p.Schedule != "0 3 * * 0" || p.RetentionDays != 14 || !p.Enabled {
		t.Fatalf("got %+v, %v", p, err)
	}
	if p, err := findBackupPolicy(ctx, "birme-osc-postgresql", "other"); err != nil || p != nil {
		t.Errorf("an instance without a policy should have none, got %+v, %v", p, err)
	}
}

func TestFindBackupPolicyRejectsEmptyBody(t *testing.T) {
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	if _, err := findBackupPolicy(ctx, "valkey-io-valkey", "db"); err == nil {
		t.Fatal("an empty body must not read as no policy")
	}
}

func TestPutBackupPolicy(t *testing.T) {
	var bodies []map[string]interface{}
	var paths []string
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		var m map[string]interface{}
		_ = json.Unmarshal(b, &m)
		bodies, paths = append(bodies, m), append(paths, r.Method+" "+r.URL.Path)
		if strings.Contains(r.URL.Path, "free") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusPaymentRequired)
			_, _ = w.Write([]byte(`{"reason":"Automatic backups require a paid plan"}`))
			return
		}
		_, _ = w.Write([]byte(`{"message":"ok"}`))
	}))

	if err := putBackupPolicy(ctx, "valkey-io-valkey", "db", true, "0 3 * * *", 7); err != nil {
		t.Fatal(err)
	}
	// Every field is sent, since the platform keeps the ones left out.
	if paths[0] != "PUT /mybackups/valkey-io-valkey/db/policy" || bodies[0]["enabled"] != true ||
		bodies[0]["schedule"] != "0 3 * * *" || bodies[0]["retentionDays"] != float64(7) {
		t.Errorf("%s %v", paths[0], bodies[0])
	}

	if err := disableBackupPolicy(ctx, "valkey-io-valkey", "db"); err != nil {
		t.Fatal(err)
	}
	if len(bodies[1]) != 1 || bodies[1]["enabled"] != false {
		t.Errorf("disabling should keep the schedule and retention, sent %v", bodies[1])
	}

	err := putBackupPolicy(ctx, "valkey-io-valkey", "free", true, "0 2 * * *", 30)
	if err == nil || !strings.Contains(err.Error(), "paid OSC plan") || !strings.Contains(err.Error(), "require a paid plan") {
		t.Errorf("402 should explain the plan and keep the platform's reason, got %v", err)
	}
}

func TestListBackupsNewestFirst(t *testing.T) {
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mybackups/valkey-io-valkey/db" {
			t.Errorf("unexpected %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{"name":"b1","status":"Complete","createdAt":1790000000,"source":"scheduled"},
			{"name":"b3","status":"Failed","createdAt":1790002000,"source":"manual","error":"boom"},
			{"name":"b2","status":"Complete","createdAt":1790001000,"source":"scheduled"}
		]`))
	}))
	backups, err := listBackups(ctx, "valkey-io-valkey", "db")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, b := range backups {
		names = append(names, b.Name)
	}
	if strings.Join(names, ",") != "b3,b2,b1" {
		t.Errorf("order %v", names)
	}
	if got := unixTime(backups[0].CreatedAt); got != "2026-09-21T14:46:40Z" {
		t.Errorf("created at %q", got)
	}
	if unixTime(0) != "" {
		t.Error("zero should format as empty")
	}
}

func backupState(t *testing.T, r resource.Resource) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	return tfsdk.State{Schema: sr.Schema, Raw: tftypes.NewValue(sr.Schema.Type().TerraformType(ctx), nil)}
}

func TestBackupScheduleRead(t *testing.T) {
	fastReads(t)
	attrs := map[string]string{"id": "valkey-io-valkey/db", "service_id": "valkey-io-valkey", "instance_name": "db"}

	for _, f := range failures() {
		t.Run(f.name, func(t *testing.T) {
			resp := readState(t, &BackupScheduleResource{osaasContext: deployManager(t, f.handler)}, attrs)
			if resp.State.Raw.IsNull() || !resp.Diagnostics.HasError() {
				t.Fatalf("a failed read must keep the schedule and fail: %v", resp.Diagnostics)
			}
		})
	}
	t.Run("gone", func(t *testing.T) {
		resp := readState(t, &BackupScheduleResource{osaasContext: deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[]`))
		}))}, attrs)
		if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
			t.Fatalf("a schedule that is gone should leave state quietly: %v", resp.Diagnostics)
		}
	})
	t.Run("found", func(t *testing.T) {
		resp := readState(t, &BackupScheduleResource{osaasContext: deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(`[{"serviceId":"valkey-io-valkey","instanceName":"db","enabled":true,"schedule":"0 4 * * *",
				"retentionDays":7,"lastBackupAt":1790000000,"credentialStatus":"ok"}]`))
		}))}, attrs)
		var got BackupScheduleResourceModel
		resp.State.Get(context.Background(), &got)
		if resp.Diagnostics.HasError() || got.Schedule.ValueString() != "0 4 * * *" || got.RetentionDays.ValueInt64() != 7 ||
			got.LastBackupAt.ValueString() != "2026-09-21T14:13:20Z" || got.LastAttemptAt.ValueString() != "" || got.CredentialStatus.ValueString() != "ok" {
			t.Fatalf("%+v %v", got, resp.Diagnostics)
		}
	})
}

// TestBackupScheduleCreateNeedsInstance: OSC accepts a policy for any name and cannot
// remove one, so a schedule for an instance that does not exist is refused before writing.
func TestBackupScheduleCreateNeedsInstance(t *testing.T) {
	catalogSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`[]`)) }))
	base := catalogBase
	catalogBase = func(string) string { return catalogSrv.URL }
	t.Cleanup(func() { catalogBase = base; catalogSrv.Close() })

	var writes int
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut {
			writes++
		}
		_, _ = w.Write([]byte(`[]`))
	}))
	r := &BackupScheduleResource{osaasContext: ctx}
	var sr resource.SchemaResponse
	r.Schema(context.Background(), resource.SchemaRequest{}, &sr)

	plan := tfsdk.Plan{Schema: sr.Schema, Raw: backupState(t, r).Raw}
	model := BackupScheduleResourceModel{
		ID:               types.StringUnknown(),
		ServiceID:        types.StringValue("valkey-io-valkey"),
		InstanceName:     types.StringValue("missing"),
		Enabled:          types.BoolValue(true),
		Schedule:         types.StringValue(defaultBackupSchedule),
		RetentionDays:    types.Int64Value(defaultBackupRetention),
		LastBackupAt:     types.StringUnknown(),
		LastAttemptAt:    types.StringUnknown(),
		LastError:        types.StringUnknown(),
		CredentialStatus: types.StringUnknown(),
	}
	if d := plan.Set(context.Background(), &model); d.HasError() {
		t.Fatal(d)
	}
	resp := &resource.CreateResponse{State: backupState(t, r)}
	r.Create(context.Background(), resource.CreateRequest{Plan: plan}, resp)
	if !resp.Diagnostics.HasError() || writes != 0 {
		t.Fatalf("want an error and no write, got %d writes, %v", writes, resp.Diagnostics)
	}
}

func TestBackupStatus(t *testing.T) {
	for in, want := range map[string]string{"SuccessCriteriaMet": "Complete", "FailureTarget": "Failed", "Running": "Running", "Complete": "Complete", "Failed": "Failed"} {
		if got := backupStatus(in); got != want {
			t.Errorf("%s: got %s, want %s", in, got, want)
		}
	}
}
