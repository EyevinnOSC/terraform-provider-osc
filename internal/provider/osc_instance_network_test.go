package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCreateExposureRequest(t *testing.T) {
	for _, tc := range []struct {
		name    string
		access  types.Bool
		present bool
	}{
		{"private", types.BoolValue(false), true},
		{"public", types.BoolValue(true), true},
		{"unset", types.BoolNull(), false},
		{"unknown", types.BoolUnknown(), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "POST" || req.Header.Get("x-jwt") != "Bearer sat" {
					t.Errorf("unexpected request: %s", req.Method)
				}
				var body map[string]interface{}
				if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
					t.Fatal(err)
				}
				got, present := body["publicAccess"]
				if present != tc.present || (present && got != tc.access.ValueBool()) {
					t.Errorf("publicAccess = %#v, present=%v", got, present)
				}
				if body["PostgresInitDbSql"] != "SELECT 1;" {
					t.Error("lost init SQL")
				}
				_, _ = w.Write([]byte(`{"name":"db"}`))
			}))
			defer srv.Close()
			service := &catalogService{ApiUrl: srv.URL}
			body := buildInstanceBody(service, "db", map[string]string{"PostgresInitDbSql": "SELECT 1;"}, nil)
			setCreateExposure(body, tc.access)
			if _, err := createInstance(service, "sat", body, false); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestNetworkRefreshAndImport(t *testing.T) {
	for _, tc := range []struct {
		name, response string
		status         int
		prior          types.Bool
		want           types.Bool
		wantError      bool
	}{
		{"private import", `{"serviceDns":"tenant-db.svc.cluster.local","publicAccess":false}`, 200, types.BoolNull(), types.BoolValue(false), false},
		{"private drifted public", `{"serviceDns":"tenant-db.svc.cluster.local","publicAccess":true}`, 200, types.BoolValue(false), types.BoolValue(true), false},
		{"missing boolean", `{"serviceDns":"tenant-db.svc.cluster.local"}`, 200, types.BoolValue(false), types.BoolValue(false), true},
		{"unsupported legacy", `{}`, 404, types.BoolNull(), types.BoolNull(), false},
		{"managed endpoint disappeared", `{}`, 404, types.BoolValue(false), types.BoolValue(false), true},
		{"authentication failure", `{}`, 401, types.BoolNull(), types.BoolNull(), true},
		{"server failure", `{}`, 500, types.BoolValue(false), types.BoolValue(false), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				if req.Method != "GET" || req.Header.Get("x-jwt") != "Bearer sat" {
					t.Errorf("unexpected request: %s", req.Method)
				}
				switch req.URL.Path {
				case "/internal-endpoint/db":
					w.WriteHeader(tc.status)
					_, _ = w.Write([]byte(tc.response))
				case "/ports/db":
					_, _ = w.Write([]byte(`[]`))
				default:
					t.Errorf("unexpected path %s", req.URL.Path)
				}
			}))
			defer srv.Close()
			old := httpClient
			httpClient = srv.Client()
			defer func() { httpClient = old }()
			service := &catalogService{ServiceId: "postgres", ApiUrl: srv.URL + "/instances"}
			model := InstanceResourceModel{Name: types.StringValue("db"), PublicAccess: tc.prior}
			d := refreshNetwork(service, "sat", &model)
			if d.HasError() != tc.wantError || !model.PublicAccess.Equal(tc.want) {
				t.Fatalf("got %v, diagnostics %v", model.PublicAccess, d)
			}
			if !tc.wantError && tc.status == 200 {
				if model.InternalHost.ValueString() != "tenant-db.svc.cluster.local" {
					t.Error("missing internal host")
				}
				imported, diags := (&InstanceResource{}).modelFromInstance(service, "sat", map[string]interface{}{"name": "db"})
				if diags.HasError() || !imported.PublicAccess.Equal(tc.want) {
					t.Fatalf("import: %v, %v", imported.PublicAccess, diags)
				}
			}
		})
	}
}

func TestExposurePlanPreservesImportedValue(t *testing.T) {
	ctx := context.Background()
	var sr resource.SchemaResponse
	(&InstanceResource{}).Schema(ctx, resource.SchemaRequest{}, &sr)
	attribute := sr.Schema.Attributes["public_access"]
	if !attribute.IsOptional() || !attribute.IsComputed() {
		t.Fatal("public_access must be optional/computed, without a public default")
	}
	// Exercise the registered modifier, with a non-null resource state.
	mods := attribute.(interface{ BoolPlanModifiers() []planmodifier.Bool }).BoolPlanModifiers()
	for _, access := range []bool{false, true} {
		state := types.BoolValue(access)
		var tfstate tfsdk.State
		tfstate.Schema = sr.Schema
		model := networkTestModel(state)
		if d := tfstate.Set(ctx, &model); d.HasError() {
			t.Fatal(d)
		}
		req := planmodifier.BoolRequest{State: tfstate, ConfigValue: types.BoolNull(), StateValue: state, PlanValue: types.BoolUnknown()}
		resp := &planmodifier.BoolResponse{PlanValue: req.PlanValue}
		for _, m := range mods {
			m.PlanModifyBool(ctx, req, resp)
		}
		if !resp.PlanValue.Equal(state) || resp.RequiresReplace {
			t.Fatalf("omitted exposure changed: %+v", resp)
		}
	}
	if !validateExposureChange(types.BoolValue(true), types.BoolValue(false)).HasError() {
		t.Error("exposure change should be refused")
	}
	if validateExposureChange(types.BoolValue(false), types.BoolValue(false)).HasError() {
		t.Error("unchanged private exposure rejected")
	}
}

func TestSuspendedPrivateImport(t *testing.T) {
	m := modelFromSuspended(&catalogService{ServiceId: "postgres"}, "db", &suspendedInstance{Parameters: map[string]interface{}{"publicAccess": false}})
	if !m.PublicAccess.Equal(types.BoolValue(false)) {
		t.Fatal("suspended import lost private setting")
	}
}

func TestExposureUpdateRefusesBeforePlatformCalls(t *testing.T) {
	ctx := context.Background()
	r := &InstanceResource{} // No client: any platform call would panic.
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)
	state := tfsdk.State{Schema: sr.Schema}
	plan := tfsdk.Plan{Schema: sr.Schema}
	if d := state.Set(ctx, networkTestModelPtr(false)); d.HasError() {
		t.Fatal(d)
	}
	if d := plan.Set(ctx, networkTestModelPtr(true)); d.HasError() {
		t.Fatal(d)
	}
	var resp resource.UpdateResponse
	r.Update(ctx, resource.UpdateRequest{State: state, Plan: plan}, &resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("update did not refuse exposure change")
	}
}

func networkTestModel(access types.Bool) InstanceResourceModel {
	return InstanceResourceModel{PublicAccess: access, Parameters: types.MapNull(types.StringType), SensitiveParameters: types.MapNull(types.StringType), SecretNames: types.MapNull(types.StringType), Instance: types.MapNull(types.StringType), Tags: types.SetNull(types.StringType)}
}

func networkTestModelPtr(access bool) *InstanceResourceModel {
	model := networkTestModel(types.BoolValue(access))
	return &model
}
