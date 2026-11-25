package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// fakeTokenValue stands in for a token; the tests never use a real one.
const fakeTokenValue = "fake-token-value-for-tests"

const (
	policyV1 = `{"version":1,"statements":[{"actions":["deploy:read"],"resources":["deploy/*"]}]}`
	// The same value as policyV1, with other formatting and key order.
	policyV1Spaced = `{
  "statements": [ { "resources": ["deploy/*"], "actions": ["deploy:read"] } ],
  "version": 1
}`
	policyV2 = `{"version":1,"statements":[{"actions":["deploy:*"],"resources":["deploy/*"]}]}`
)

type seenRequest struct {
	Method string
	Path   string
	Body   map[string]interface{}
}

// tokenFake stands in for token-service, answering from a function and recording requests.
type tokenFake struct {
	mu   sync.Mutex
	seen []seenRequest
}

func (f *tokenFake) requests() []seenRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]seenRequest(nil), f.seen...)
}

// serveTokens points the provider at a fake token-service for the test.
func serveTokens(t *testing.T, answer func(r seenRequest, w http.ResponseWriter)) (*tokenFake, *osaasclient.Context) {
	t.Helper()
	fake := &tokenFake{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		raw, _ := io.ReadAll(req.Body)
		seen := seenRequest{Method: req.Method, Path: req.URL.Path}
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &seen.Body)
		}
		if req.Header.Get("x-pat-jwt") != "Bearer "+fakeTokenValue {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fake.mu.Lock()
		fake.seen = append(fake.seen, seen)
		fake.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		answer(seen, w)
	}))
	base, client := tokenBase, httpClient
	tokenBase = func(string) string { return srv.URL }
	httpClient = srv.Client()
	t.Cleanup(func() { tokenBase, httpClient = base, client; srv.Close() })
	ctx, err := osaasclient.NewContext(&osaasclient.ContextConfig{PersonalAccessToken: fakeTokenValue, Environment: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	return fake, ctx
}

func tokenRecordJSON(version int, status, name, policy string, extra string) string {
	return `{"patId":"pat-1","tenantId":"acme","userId":"u1","name":"` + name + `","description":"","status":"` + status +
		`","policy":` + policy + `,"policyVersion":` + strconv.Itoa(version) + `,"enforcement":"enforce","auditLevel":"default",` +
		`"createdAt":"2026-11-25T10:00:00.000Z","createdBy":"token:x","exp":1795600000` + extra + `}`
}

func reply(w http.ResponseWriter, status int, body string) {
	w.WriteHeader(status)
	_, _ = w.Write([]byte(body))
}

func tokenSchemaForTest() schema.Schema {
	var sr resource.SchemaResponse
	(&AccessTokenResource{}).Schema(context.Background(), resource.SchemaRequest{}, &sr)
	return sr.Schema
}

func nullTokenModel() AccessTokenResourceModel {
	return AccessTokenResourceModel{
		ID: types.StringNull(), Name: types.StringNull(), Description: types.StringNull(), Template: types.StringNull(),
		Policy: types.StringNull(), EffectivePolicy: types.StringNull(), ExpiresInDays: types.Int64Null(),
		ExpiresAt: types.StringNull(), Status: types.StringNull(), Enforcement: types.StringNull(),
		PolicyVersion: types.Int64Null(), LastUsedAt: types.StringNull(), Token: types.StringNull(),
	}
}

// createPlan is the plan for a new token made from the viewer template that lives one day.
func createPlan() AccessTokenResourceModel {
	m := nullTokenModel()
	m.ID, m.EffectivePolicy, m.ExpiresAt = types.StringUnknown(), types.StringUnknown(), types.StringUnknown()
	m.PolicyVersion, m.LastUsedAt, m.Token = types.Int64Unknown(), types.StringUnknown(), types.StringUnknown()
	m.Name, m.Description, m.Template = types.StringValue("ci"), types.StringValue(""), types.StringValue("viewer")
	m.ExpiresInDays, m.Status, m.Enforcement = types.Int64Value(1), types.StringValue("active"), types.StringValue("enforce")
	return m
}

// existing is the state of a token created with an explicit policy.
func existing() AccessTokenResourceModel {
	m := nullTokenModel()
	m.ID, m.Name, m.Description = types.StringValue("pat-1"), types.StringValue("ci"), types.StringValue("")
	m.Policy, m.EffectivePolicy = types.StringValue(policyV1), types.StringValue(policyV1)
	m.ExpiresAt, m.Status, m.Enforcement = types.StringValue("2026-11-26T10:00:00Z"), types.StringValue("active"), types.StringValue("enforce")
	m.PolicyVersion, m.Token = types.Int64Value(1), types.StringValue(fakeTokenValue)
	return m
}

func rawOf(t *testing.T, m AccessTokenResourceModel) tftypes.Value {
	t.Helper()
	sch := tokenSchemaForTest()
	s := tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(context.Background()), nil)}
	if d := s.Set(context.Background(), &m); d.HasError() {
		t.Fatal(d)
	}
	return s.Raw
}

func emptyState() tfsdk.State {
	sch := tokenSchemaForTest()
	return tfsdk.State{Schema: sch, Raw: tftypes.NewValue(sch.Type().TerraformType(context.Background()), nil)}
}

func stateWith(t *testing.T, m AccessTokenResourceModel) tfsdk.State {
	return tfsdk.State{Schema: tokenSchemaForTest(), Raw: rawOf(t, m)}
}

func planWith(t *testing.T, m AccessTokenResourceModel) tfsdk.Plan {
	return tfsdk.Plan{Schema: tokenSchemaForTest(), Raw: rawOf(t, m)}
}

func stateModel(t *testing.T, s tfsdk.State) AccessTokenResourceModel {
	t.Helper()
	var m AccessTokenResourceModel
	if d := s.Get(context.Background(), &m); d.HasError() {
		t.Fatal(d)
	}
	return m
}

func doCreate(t *testing.T, ctx *osaasclient.Context, plan AccessTokenResourceModel) *resource.CreateResponse {
	resp := &resource.CreateResponse{State: emptyState()}
	(&AccessTokenResource{osaasContext: ctx}).Create(context.Background(), resource.CreateRequest{Plan: planWith(t, plan)}, resp)
	return resp
}

func doUpdate(t *testing.T, ctx *osaasclient.Context, state, plan AccessTokenResourceModel) *resource.UpdateResponse {
	resp := &resource.UpdateResponse{State: stateWith(t, state)}
	(&AccessTokenResource{osaasContext: ctx}).Update(context.Background(),
		resource.UpdateRequest{Plan: planWith(t, plan), State: stateWith(t, state)}, resp)
	return resp
}

func doRead(t *testing.T, ctx *osaasclient.Context, state AccessTokenResourceModel) *resource.ReadResponse {
	resp := &resource.ReadResponse{State: stateWith(t, state)}
	(&AccessTokenResource{osaasContext: ctx}).Read(context.Background(), resource.ReadRequest{State: stateWith(t, state)}, resp)
	return resp
}

func TestAccessTokenRegistered(t *testing.T) {
	found := false
	for _, f := range RegisteredResources {
		var mr resource.MetadataResponse
		f().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "osc"}, &mr)
		found = found || mr.TypeName == "osc_access_token"
	}
	if !found {
		t.Fatal("osc_access_token is not registered")
	}
	sch := tokenSchemaForTest()
	if d := sch.ValidateImplementation(context.Background()); d.HasError() {
		t.Fatal(d)
	}
}

func TestAccessTokenCreateAndRead(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		switch r.Method + " " + r.Path {
		case "POST /tokens":
			reply(w, 201, tokenRecordJSON(1, "active", "ci", policyV1, `,"token":"`+fakeTokenValue+`"`))
		case "GET /tokens/pat-1":
			reply(w, 200, tokenRecordJSON(1, "active", "ci", policyV1, `,"lastUsedAt":"2026-11-25T11:00:00.000Z"`))
		default:
			reply(w, 500, `{"reason":"unexpected"}`)
		}
	})
	resp := doCreate(t, ctx, createPlan())
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	reqs := fake.requests()
	if len(reqs) != 1 || reqs[0].Method != "POST" || reqs[0].Path != "/tokens" {
		t.Fatalf("create must be one POST /tokens, got %+v", reqs)
	}
	body := reqs[0].Body
	if body["name"] != "ci" || body["template"] != "viewer" || body["expiresInSeconds"] != float64(86400) || body["enforcement"] != "enforce" {
		t.Errorf("create body: %+v", body)
	}
	if _, ok := body["policy"]; ok {
		t.Error("a token made from a template must not send a policy")
	}
	got := stateModel(t, resp.State)
	if got.ID.ValueString() != "pat-1" || got.Token.ValueString() != fakeTokenValue || got.PolicyVersion.ValueInt64() != 1 ||
		got.ExpiresAt.ValueString() != time.Unix(1795600000, 0).UTC().Format(time.RFC3339) ||
		got.EffectivePolicy.ValueString() != norm(policyV1) || !got.Policy.IsNull() || got.Template.ValueString() != "viewer" {
		t.Errorf("state after create: %+v", got)
	}

	// A read does not show the token value again and must not change what was stored.
	rr := doRead(t, ctx, got)
	if rr.Diagnostics.HasError() {
		t.Fatal(rr.Diagnostics)
	}
	after := stateModel(t, rr.State)
	if after.Token.ValueString() != fakeTokenValue || after.LastUsedAt.ValueString() != "2026-11-25T11:00:00.000Z" {
		t.Errorf("state after read: %+v", after)
	}
	if !after.Policy.IsNull() || after.Template.ValueString() != "viewer" {
		t.Errorf("read must keep template and a null policy: %+v", after)
	}
}

func TestAccessTokenCreateWithPolicy(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 201, tokenRecordJSON(1, "active", "ci", policyV1, `,"token":"`+fakeTokenValue+`"`))
	})
	plan := createPlan()
	plan.Template, plan.Policy, plan.ExpiresInDays = types.StringNull(), types.StringValue(policyV1Spaced), types.Int64Null()
	resp := doCreate(t, ctx, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	body := fake.requests()[0].Body
	if _, ok := body["template"]; ok {
		t.Error("template sent with a policy")
	}
	if _, ok := body["expiresInSeconds"]; ok {
		t.Error("an omitted lifetime must be left to the server")
	}
	if p, _ := json.Marshal(body["policy"]); !sameJSON(p, []byte(policyV1)) {
		t.Errorf("policy sent: %s", p)
	}
	// What the configuration wrote is kept, since the stored policy means the same.
	if got := stateModel(t, resp.State); got.Policy.ValueString() != policyV1Spaced {
		t.Errorf("policy in state: %q", got.Policy.ValueString())
	}
}

func TestAccessTokenCreateRefusesDisabled(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { reply(w, 500, `{}`) })
	plan := createPlan()
	plan.Status = types.StringValue("disabled")
	resp := doCreate(t, ctx, plan)
	if !resp.Diagnostics.HasError() || len(fake.requests()) != 0 {
		t.Fatalf("a disabled new token must be refused before any request: %v, %+v", resp.Diagnostics, fake.requests())
	}
	// The same refusal at plan time.
	mp := &resource.ModifyPlanResponse{}
	(&AccessTokenResource{}).ModifyPlan(context.Background(),
		resource.ModifyPlanRequest{Plan: planWith(t, plan), State: emptyState()}, mp)
	if !mp.Diagnostics.HasError() {
		t.Error("plan accepted a new disabled token")
	}
	// Passing path: an existing token can be disabled, and a new active one plans.
	mp = &resource.ModifyPlanResponse{}
	(&AccessTokenResource{}).ModifyPlan(context.Background(),
		resource.ModifyPlanRequest{Plan: planWith(t, existing()), State: stateWith(t, existing())}, mp)
	(&AccessTokenResource{}).ModifyPlan(context.Background(),
		resource.ModifyPlanRequest{Plan: planWith(t, createPlan()), State: emptyState()}, mp)
	if mp.Diagnostics.HasError() {
		t.Errorf("unexpected plan error: %v", mp.Diagnostics)
	}
}

// TestAccessTokenReplacementCannotBeDisabled pins what happens to an existing disabled token
// whose template or lifetime changes: that replaces it (delete, then create), a create is
// always active, so the plan is refused before anything is deleted.
func TestAccessTokenReplacementCannotBeDisabled(t *testing.T) {
	plan := func(state, planned AccessTokenResourceModel) *resource.ModifyPlanResponse {
		resp := &resource.ModifyPlanResponse{}
		(&AccessTokenResource{}).ModifyPlan(context.Background(),
			resource.ModifyPlanRequest{Plan: planWith(t, planned), State: stateWith(t, state)}, resp)
		return resp
	}
	disabled := existing()
	disabled.Status = types.StringValue("disabled")
	disabled.Policy, disabled.Template, disabled.ExpiresInDays = types.StringNull(), types.StringValue("viewer"), types.Int64Value(7)

	// Replacement by a new lifetime or a new template with `disabled` is refused.
	byLifetime := disabled
	byLifetime.ExpiresInDays = types.Int64Value(30)
	byTemplate := disabled
	byTemplate.Template = types.StringValue("deployer")
	byTemplateRemoved := disabled
	byTemplateRemoved.Template = types.StringNull()
	for name, planned := range map[string]AccessTokenResourceModel{"lifetime": byLifetime, "template": byTemplate, "template removed, no policy": byTemplateRemoved} {
		resp := plan(disabled, planned)
		if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "replaced token") {
			t.Errorf("%s: want the replacement refused at plan time: %v", name, resp.Diagnostics)
		}
	}

	// Passing paths: the same token staying disabled, and in-place changes, are fine.
	if resp := plan(disabled, disabled); resp.Diagnostics.HasError() {
		t.Errorf("unchanged disabled token: %v", resp.Diagnostics)
	}
	renamed := disabled
	renamed.Name = types.StringValue("renamed")
	if resp := plan(disabled, renamed); resp.Diagnostics.HasError() {
		t.Errorf("rename of a disabled token: %v", resp.Diagnostics)
	}
	toPolicy := disabled
	toPolicy.Template, toPolicy.Policy = types.StringNull(), types.StringValue(policyV2)
	if resp := plan(disabled, toPolicy); resp.Diagnostics.HasError() {
		t.Errorf("template replaced by policy is in place: %v", resp.Diagnostics)
	}
	// A replacement to an active token is fine.
	active := byLifetime
	active.Status = types.StringValue("active")
	if resp := plan(disabled, active); resp.Diagnostics.HasError() {
		t.Errorf("replacement to an active token: %v", resp.Diagnostics)
	}
}

func TestAccessTokenCreateNotEnabled(t *testing.T) {
	_, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { reply(w, 404, `{"reason":"not found"}`) })
	resp := doCreate(t, ctx, createPlan())
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "not enabled") {
		t.Fatalf("want the not-enabled error: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Error("a failed create must leave no state")
	}
}

func TestAccessTokenCreateTimeoutIsOutcomeUnknownAndNotRetried(t *testing.T) {
	release := make(chan struct{})
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { <-release })
	t.Cleanup(func() { close(release) })
	httpClient = &http.Client{Timeout: 150 * time.Millisecond}
	resp := doCreate(t, ctx, createPlan())
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Summary(), "outcome") {
		t.Fatalf("want the outcome-unknown error: %v", resp.Diagnostics)
	}
	if detail := resp.Diagnostics.Errors()[0].Detail(); !strings.Contains(detail, "osc token list") || strings.Contains(detail, fakeTokenValue) {
		t.Errorf("error should say how to check and carry no credential: %s", detail)
	}
	if n := len(fake.requests()); n != 1 {
		t.Fatalf("the POST must never be retried, got %d requests", n)
	}
}

func TestAccessTokenCreateServerErrorOutcome(t *testing.T) {
	cases := []struct {
		status      int
		wantUnknown bool
	}{
		// token-service can fail after it stored the record, so a token may exist.
		{500, true}, {502, true}, {504, true},
		// Its own 503s ("token store not configured", "policy cache unavailable") come
		// before the token is kept, and a 4xx is a refusal.
		{503, false}, {400, false}, {429, false},
	}
	for _, c := range cases {
		fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
			reply(w, c.status, `{"reason":"boom"}`)
		})
		resp := doCreate(t, ctx, createPlan())
		if !resp.Diagnostics.HasError() {
			t.Fatalf("%d: want an error", c.status)
		}
		e := resp.Diagnostics.Errors()[0]
		unknown := strings.Contains(e.Summary(), "outcome")
		if unknown != c.wantUnknown {
			t.Errorf("%d: outcome-unknown = %v, want %v (%s)", c.status, unknown, c.wantUnknown, e.Summary())
		}
		if unknown && !strings.Contains(e.Detail(), "osc token list") {
			t.Errorf("%d: error should say how to check: %s", c.status, e.Detail())
		}
		if n := len(fake.requests()); n != 1 {
			t.Errorf("%d: the POST must never be retried, got %d requests", c.status, n)
		}
		if !resp.State.Raw.IsNull() {
			t.Errorf("%d: a failed create must leave no state", c.status)
		}
	}
}

func TestAccessTokenCreateRejectionIsPlainError(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 409, `{"reason":"token quota reached"}`)
	})
	resp := doCreate(t, ctx, createPlan())
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "token quota reached") {
		t.Fatalf("want the platform's reason: %v", resp.Diagnostics)
	}
	if len(fake.requests()) != 1 {
		t.Errorf("requests: %+v", fake.requests())
	}
}

func TestAccessTokenPolicyChangeIsOnePut(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		if r.Method == "PUT" && r.Path == "/tokens/pat-1/policy" {
			reply(w, 200, tokenRecordJSON(2, "active", "ci", policyV2, ``))
			return
		}
		reply(w, 500, `{"reason":"unexpected"}`)
	})
	state := existing()
	plan := state
	plan.Policy = types.StringValue(policyV2)
	plan.EffectivePolicy, plan.PolicyVersion = types.StringUnknown(), types.Int64Unknown()
	resp := doUpdate(t, ctx, state, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	reqs := fake.requests()
	if len(reqs) != 1 || reqs[0].Method != "PUT" || reqs[0].Path != "/tokens/pat-1/policy" {
		t.Fatalf("want one PUT /tokens/pat-1/policy, got %+v", reqs)
	}
	if reqs[0].Body["expectedVersion"] != float64(1) {
		t.Errorf("expectedVersion: %+v", reqs[0].Body)
	}
	if _, ok := reqs[0].Body["exp"]; ok {
		t.Error("exp must never be sent")
	}
	got := stateModel(t, resp.State)
	if got.ID.ValueString() != "pat-1" || got.PolicyVersion.ValueInt64() != 2 || got.Token.ValueString() != fakeTokenValue ||
		got.EffectivePolicy.ValueString() != norm(policyV2) || got.Policy.ValueString() != policyV2 {
		t.Errorf("state after policy change: %+v", got)
	}
}

func TestAccessTokenTemplateToPolicyIsInPlace(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 200, tokenRecordJSON(2, "active", "ci", policyV2, ``))
	})
	state := existing()
	state.Policy, state.Template = types.StringNull(), types.StringValue("viewer")
	plan := state
	plan.Policy, plan.Template = types.StringValue(policyV2), types.StringNull()
	resp := doUpdate(t, ctx, state, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if reqs := fake.requests(); len(reqs) != 1 || reqs[0].Method != "PUT" {
		t.Fatalf("want one PUT, got %+v", reqs)
	}
	if got := stateModel(t, resp.State); !got.Template.IsNull() || got.Policy.ValueString() != policyV2 || got.ID.ValueString() != "pat-1" {
		t.Errorf("state: %+v", got)
	}
}

func TestAccessTokenSemanticallyEqualPolicySendsNothing(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { reply(w, 500, `{}`) })
	state := existing()
	plan := state
	plan.Policy = types.StringValue(policyV1Spaced)
	resp := doUpdate(t, ctx, state, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if len(fake.requests()) != 0 {
		t.Fatalf("a policy that means the same must change nothing, got %+v", fake.requests())
	}
	if got := stateModel(t, resp.State); got.Policy.ValueString() != policyV1Spaced || got.PolicyVersion.ValueInt64() != 1 {
		t.Errorf("state: %+v", got)
	}
}

func TestAccessTokenPolicyConflictIsOneRequestAndAnError(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 409, `{"reason":"policy version mismatch"}`)
	})
	state := existing()
	plan := state
	plan.Policy = types.StringValue(policyV2)
	resp := doUpdate(t, ctx, state, plan)
	if !resp.Diagnostics.HasError() || !strings.Contains(resp.Diagnostics.Errors()[0].Detail(), "outside Terraform since version 1") {
		t.Fatalf("want the conflict error: %v", resp.Diagnostics)
	}
	if n := len(fake.requests()); n != 1 {
		t.Fatalf("a 409 must not be retried, got %d requests", n)
	}
	// The state is untouched.
	if got := stateModel(t, resp.State); got.PolicyVersion.ValueInt64() != 1 || got.Policy.ValueString() != policyV1 {
		t.Errorf("state changed on a failed update: %+v", got)
	}
}

func TestAccessTokenNameAndStatusChangeIsOnePatchWithoutExp(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 200, tokenRecordJSON(1, "disabled", "ci-renamed", policyV1, ``))
	})
	state := existing()
	plan := state
	plan.Name, plan.Status = types.StringValue("ci-renamed"), types.StringValue("disabled")
	resp := doUpdate(t, ctx, state, plan)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	reqs := fake.requests()
	if len(reqs) != 1 || reqs[0].Method != "PATCH" || reqs[0].Path != "/tokens/pat-1" {
		t.Fatalf("want one PATCH /tokens/pat-1, got %+v", reqs)
	}
	if len(reqs[0].Body) != 2 || reqs[0].Body["name"] != "ci-renamed" || reqs[0].Body["status"] != "disabled" {
		t.Errorf("patch body must carry only what changed: %+v", reqs[0].Body)
	}
	if _, ok := reqs[0].Body["exp"]; ok {
		t.Error("exp must never be sent")
	}
	if got := stateModel(t, resp.State); got.Status.ValueString() != "disabled" || got.Name.ValueString() != "ci-renamed" ||
		got.Token.ValueString() != fakeTokenValue {
		t.Errorf("state: %+v", got)
	}
}

func TestAccessTokenDescriptionAndEnforcementPatch(t *testing.T) {
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		reply(w, 200, strings.Replace(tokenRecordJSON(1, "active", "ci", policyV1, ``), `"enforcement":"enforce"`, `"enforcement":"audit"`, 1))
	})
	state := existing()
	plan := state
	plan.Description, plan.Enforcement = types.StringValue("deploys"), types.StringValue("audit")
	if resp := doUpdate(t, ctx, state, plan); resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	body := fake.requests()[0].Body
	if len(body) != 2 || body["description"] != "deploys" || body["enforcement"] != "audit" {
		t.Errorf("patch body: %+v", body)
	}
}

func TestAccessTokenReadGoneAndDegraded(t *testing.T) {
	status, body := 404, `{"reason":"token not found"}`
	_, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { reply(w, status, body) })

	resp := doRead(t, ctx, existing())
	if resp.Diagnostics.HasError() || !resp.State.Raw.IsNull() {
		t.Fatalf("a token that is gone must leave the state: %v", resp.Diagnostics)
	}

	body = `{"reason":"role lookup unavailable"}`
	resp = doRead(t, ctx, existing())
	if !resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
		t.Fatalf("degraded mode must be an error that keeps the state: %v", resp.Diagnostics)
	}

	// A 404 for the route (the API is off) is not the token being gone either.
	body = `{"reason":"not found"}`
	resp = doRead(t, ctx, existing())
	if !resp.Diagnostics.HasError() || resp.State.Raw.IsNull() {
		t.Fatalf("a route that is off must keep the state: %v", resp.Diagnostics)
	}
}

func TestAccessTokenDelete(t *testing.T) {
	status, body := 204, ``
	fake, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) { reply(w, status, body) })
	del := func() *resource.DeleteResponse {
		resp := &resource.DeleteResponse{}
		(&AccessTokenResource{osaasContext: ctx}).Delete(context.Background(),
			resource.DeleteRequest{State: stateWith(t, existing())}, resp)
		return resp
	}
	if resp := del(); resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	if reqs := fake.requests(); len(reqs) != 1 || reqs[0].Method != "DELETE" || reqs[0].Path != "/tokens/pat-1" {
		t.Fatalf("want one DELETE /tokens/pat-1, got %+v", reqs)
	}
	status, body = 404, `{"reason":"token not found"}`
	if resp := del(); resp.Diagnostics.HasError() {
		t.Fatalf("a token already gone is a successful delete: %v", resp.Diagnostics)
	}
	status, body = 404, `{"reason":"role lookup unavailable"}`
	if resp := del(); !resp.Diagnostics.HasError() {
		t.Fatal("a token that exists but could not be looked up is not deleted")
	}
}

func TestAccessTokenImport(t *testing.T) {
	_, ctx := serveTokens(t, func(r seenRequest, w http.ResponseWriter) {
		if r.Path == "/tokens/pat-1" {
			reply(w, 200, tokenRecordJSON(3, "disabled", "ci", policyV2, `,"lastUsedAt":"2026-11-25T11:00:00.000Z"`))
			return
		}
		reply(w, 404, `{"reason":"token not found"}`)
	})
	r := &AccessTokenResource{osaasContext: ctx}
	resp := &resource.ImportStateResponse{State: emptyState()}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "pat-1"}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatal(resp.Diagnostics)
	}
	got := stateModel(t, resp.State)
	if got.ID.ValueString() != "pat-1" || got.Name.ValueString() != "ci" || got.Status.ValueString() != "disabled" ||
		got.Enforcement.ValueString() != "enforce" || got.PolicyVersion.ValueInt64() != 3 || got.ExpiresAt.IsNull() ||
		got.LastUsedAt.ValueString() != "2026-11-25T11:00:00.000Z" || got.EffectivePolicy.ValueString() != norm(policyV2) ||
		got.Policy.ValueString() != norm(policyV2) || got.Description.IsNull() {
		t.Errorf("imported state: %+v", got)
	}
	if !got.Token.IsNull() {
		t.Error("the token value cannot be imported and must be null")
	}

	// A token that does not exist is an error.
	resp = &resource.ImportStateResponse{State: emptyState()}
	r.ImportState(context.Background(), resource.ImportStateRequest{ID: "missing"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("import of an unknown id must fail")
	}
}

func TestAccessTokenValidateConfig(t *testing.T) {
	validate := func(mutate func(m *AccessTokenResourceModel)) (errs []string) {
		m := nullTokenModel()
		m.Name = types.StringValue("ci")
		mutate(&m)
		resp := &resource.ValidateConfigResponse{}
		(&AccessTokenResource{}).ValidateConfig(context.Background(),
			resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: tokenSchemaForTest(), Raw: rawOf(t, m)}}, resp)
		for _, d := range resp.Diagnostics.Errors() {
			errs = append(errs, d.Summary())
		}
		return errs
	}
	cases := []struct {
		name    string
		mutate  func(m *AccessTokenResourceModel)
		wantErr bool
	}{
		{"template only", func(m *AccessTokenResourceModel) { m.Template = types.StringValue("viewer") }, false},
		{"policy only", func(m *AccessTokenResourceModel) { m.Policy = types.StringValue(policyV1) }, false},
		{"both", func(m *AccessTokenResourceModel) {
			m.Template, m.Policy = types.StringValue("viewer"), types.StringValue(policyV1)
		}, true},
		{"neither", func(m *AccessTokenResourceModel) {}, true},
		{"policy not known yet", func(m *AccessTokenResourceModel) { m.Policy = types.StringUnknown() }, false},
		{"unknown template", func(m *AccessTokenResourceModel) { m.Template = types.StringValue("root") }, true},
		{"policy not JSON", func(m *AccessTokenResourceModel) { m.Policy = types.StringValue("allow all") }, true},
		{"policy not an object", func(m *AccessTokenResourceModel) { m.Policy = types.StringValue("[]") }, true},
		{"lifetime 90", func(m *AccessTokenResourceModel) {
			m.Template, m.ExpiresInDays = types.StringValue("full"), types.Int64Value(90)
		}, false},
		{"lifetime 0", func(m *AccessTokenResourceModel) {
			m.Template, m.ExpiresInDays = types.StringValue("full"), types.Int64Value(0)
		}, true},
		{"lifetime 91", func(m *AccessTokenResourceModel) {
			m.Template, m.ExpiresInDays = types.StringValue("full"), types.Int64Value(91)
		}, true},
		{"bad status", func(m *AccessTokenResourceModel) {
			m.Template, m.Status = types.StringValue("full"), types.StringValue("paused")
		}, true},
		{"audit enforcement", func(m *AccessTokenResourceModel) {
			m.Template, m.Enforcement = types.StringValue("full"), types.StringValue("audit")
		}, false},
		{"bad enforcement", func(m *AccessTokenResourceModel) {
			m.Template, m.Enforcement = types.StringValue("full"), types.StringValue("log")
		}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if errs := validate(c.mutate); (len(errs) > 0) != c.wantErr {
				t.Errorf("errors: %v, want error: %v", errs, c.wantErr)
			}
		})
	}
}

// runString applies an attribute's string plan modifiers the way the framework does for an
// existing resource.
func runString(t *testing.T, attr string, state, plan AccessTokenResourceModel, config AccessTokenResourceModel,
	stateValue, planValue, configValue types.String) *planmodifier.StringResponse {
	t.Helper()
	sch := tokenSchemaForTest()
	mods := sch.Attributes[attr].(interface{ StringPlanModifiers() []planmodifier.String }).StringPlanModifiers()
	req := planmodifier.StringRequest{
		Path:   path.Root(attr),
		Config: tfsdk.Config{Schema: sch, Raw: rawOf(t, config)}, ConfigValue: configValue,
		Plan: planWith(t, plan), PlanValue: planValue,
		State: stateWith(t, state), StateValue: stateValue,
	}
	resp := &planmodifier.StringResponse{PlanValue: planValue}
	for _, m := range mods {
		m.PlanModifyString(context.Background(), req, resp)
	}
	return resp
}

func TestAccessTokenPlanReplacement(t *testing.T) {
	viaTemplate := existing()
	viaTemplate.Policy, viaTemplate.Template = types.StringNull(), types.StringValue("viewer")

	// A changed template replaces the token.
	cfg := viaTemplate
	cfg.Template = types.StringValue("deployer")
	if r := runString(t, "template", viaTemplate, cfg, cfg, types.StringValue("viewer"), types.StringValue("deployer"), types.StringValue("deployer")); !r.RequiresReplace {
		t.Error("changing template must replace the token")
	}
	// Removing the template without a policy replaces it too: the scope would be unknown.
	cfg = viaTemplate
	cfg.Template = types.StringNull()
	if r := runString(t, "template", viaTemplate, cfg, cfg, types.StringValue("viewer"), types.StringNull(), types.StringNull()); !r.RequiresReplace {
		t.Error("removing template with no policy must replace the token")
	}
	// Passing path: replacing the template with a policy changes the same token in place.
	cfg = viaTemplate
	cfg.Template, cfg.Policy = types.StringNull(), types.StringValue(policyV2)
	if r := runString(t, "template", viaTemplate, cfg, cfg, types.StringValue("viewer"), types.StringNull(), types.StringNull()); r.RequiresReplace {
		t.Error("template replaced by policy must be an in-place change")
	}

	// A changed lifetime replaces the token.
	sch := tokenSchemaForTest()
	mods := sch.Attributes["expires_in_days"].(interface{ Int64PlanModifiers() []planmodifier.Int64 }).Int64PlanModifiers()
	st, pl := existing(), existing()
	st.ExpiresInDays, pl.ExpiresInDays = types.Int64Value(7), types.Int64Value(30)
	req := planmodifier.Int64Request{Path: path.Root("expires_in_days"), Plan: planWith(t, pl), State: stateWith(t, st),
		StateValue: types.Int64Value(7), PlanValue: types.Int64Value(30), ConfigValue: types.Int64Value(30)}
	resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}
	for _, m := range mods {
		m.PlanModifyInt64(context.Background(), req, resp)
	}
	if !resp.RequiresReplace {
		t.Error("changing expires_in_days must replace the token")
	}

	// Passing path: none of the in-place attributes replaces the token.
	for _, attr := range []string{"policy", "name", "description", "status", "enforcement"} {
		mods := sch.Attributes[attr].(interface{ StringPlanModifiers() []planmodifier.String }).StringPlanModifiers()
		for _, m := range mods {
			resp := &planmodifier.StringResponse{PlanValue: types.StringValue("changed")}
			m.PlanModifyString(context.Background(), planmodifier.StringRequest{Path: path.Root(attr),
				Plan: planWith(t, existing()), State: stateWith(t, existing()),
				StateValue: types.StringValue("old"), PlanValue: types.StringValue("changed"), ConfigValue: types.StringValue("changed")}, resp)
			if resp.RequiresReplace {
				t.Errorf("changing %s must not replace the token", attr)
			}
		}
	}
}

func TestAccessTokenTokenIsSensitiveAndKeptOnUpdate(t *testing.T) {
	sch := tokenSchemaForTest()
	tokenAttr := sch.Attributes["token"].(schema.StringAttribute)
	if !tokenAttr.Sensitive || !tokenAttr.Computed {
		t.Fatal("token must be computed and sensitive")
	}
	for _, name := range []string{"template", "policy", "name"} {
		if sch.Attributes[name].IsSensitive() {
			t.Errorf("%s must not be sensitive", name)
		}
	}
	// On an update the planned token is unknown; it must take the state value.
	r := runString(t, "token", existing(), existing(), existing(), types.StringValue(fakeTokenValue), types.StringUnknown(), types.StringNull())
	if r.PlanValue.ValueString() != fakeTokenValue || r.RequiresReplace {
		t.Errorf("token plan value: %+v", r)
	}
}

func TestAccessTokenPolicyDerivedPlan(t *testing.T) {
	sch := tokenSchemaForTest()
	versionMods := sch.Attributes["policy_version"].(interface{ Int64PlanModifiers() []planmodifier.Int64 }).Int64PlanModifiers()
	plan := func(configPolicy types.String) types.Int64 {
		cfg := existing()
		cfg.Policy = configPolicy
		req := planmodifier.Int64Request{Path: path.Root("policy_version"), Config: tfsdk.Config{Schema: sch, Raw: rawOf(t, cfg)},
			Plan: planWith(t, cfg), State: stateWith(t, existing()), StateValue: types.Int64Value(1), PlanValue: types.Int64Unknown()}
		resp := &planmodifier.Int64Response{PlanValue: req.PlanValue}
		for _, m := range versionMods {
			m.PlanModifyInt64(context.Background(), req, resp)
		}
		return resp.PlanValue
	}
	// A rename or a differently formatted policy cannot change the version, so it stays known.
	if v := plan(types.StringNull()); v.IsUnknown() || v.ValueInt64() != 1 {
		t.Errorf("template-based token: %v", v)
	}
	if v := plan(types.StringValue(policyV1Spaced)); v.IsUnknown() || v.ValueInt64() != 1 {
		t.Errorf("same policy, other formatting: %v", v)
	}
	// A real policy change bumps it, which is only known after the apply.
	if v := plan(types.StringValue(policyV2)); !v.IsUnknown() {
		t.Errorf("changed policy: %v", v)
	}
}

// norm is a policy as the provider stores it in effective_policy.
func norm(policy string) string {
	n, err := normalizeJSON([]byte(policy))
	if err != nil {
		panic(err)
	}
	return n
}

func TestAccessTokenHelpers(t *testing.T) {
	if !sameJSON([]byte(policyV1), []byte(policyV1Spaced)) || sameJSON([]byte(policyV1), []byte(policyV2)) {
		t.Error("sameJSON")
	}
	if sameJSON([]byte("nope"), []byte(policyV1)) {
		t.Error("invalid JSON is never equal")
	}
	n, err := normalizeJSON([]byte(policyV1Spaced))
	if err != nil || n != policyV1Spaced && n != `{"statements":[{"actions":["deploy:read"],"resources":["deploy/*"]}],"version":1}` {
		t.Errorf("normalizeJSON: %q %v", n, err)
	}
	if u := tokenURL(&osaasclient.Context{Environment: "dev"}, "/tokens/%s", "a/b"); u != "https://token.svc.dev.osaas.io/tokens/a%2Fb" {
		t.Errorf("tokenURL: %s", u)
	}
}

// TestAccessTokenDevAPI runs the API helper against token-service in dev, as the agent-run
// check of the ticket describes. It needs a personal access token for the dev test tenant.
func TestAccessTokenDevAPI(t *testing.T) {
	pat := os.Getenv("OSC_ACCESS_TOKEN")
	if os.Getenv("TF_ACC") != "1" || pat == "" {
		t.Skip("set TF_ACC=1 and OSC_ACCESS_TOKEN (a dev personal access token) to run against token.svc.dev.osaas.io")
	}
	ctx, err := osaasclient.NewContext(&osaasclient.ContextConfig{PersonalAccessToken: pat, Environment: "dev"})
	if err != nil {
		t.Fatal(err)
	}
	rec, err := createAccessToken(ctx, map[string]interface{}{
		"name": "terraform-provider-osc-acc-test", "template": "viewer", "expiresInSeconds": 86400,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	deleted := false
	defer func() {
		if !deleted {
			_ = deleteAccessToken(ctx, rec.PatID)
		}
	}()
	if rec.Token == "" {
		t.Fatal("create returned no token value")
	}
	t.Logf("created %s, token <redacted>, expires %d", rec.PatID, rec.Exp)

	got, err := getAccessToken(ctx, rec.PatID)
	if err != nil || got.PolicyVersion != 1 {
		t.Fatalf("get: %+v %v", got, err)
	}
	var policy json.RawMessage = json.RawMessage(policyV1)
	replaced, err := replaceAccessTokenPolicy(ctx, rec.PatID, policy, 1)
	if err != nil || replaced.PatID != rec.PatID || replaced.PolicyVersion != 2 {
		t.Fatalf("replace policy: %+v %v", replaced, err)
	}
	patched, err := patchAccessToken(ctx, rec.PatID, map[string]interface{}{"status": "disabled"})
	if err != nil || patched.Status != "disabled" {
		t.Fatalf("patch: %+v %v", patched, err)
	}
	if err := deleteAccessToken(ctx, rec.PatID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	deleted = true
	if _, err := getAccessToken(ctx, rec.PatID); !isTokenNotFound(err) {
		t.Fatalf("get after delete: want 404 token not found, got %v", err)
	}
}
