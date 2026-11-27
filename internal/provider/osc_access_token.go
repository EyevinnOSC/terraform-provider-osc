package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &AccessTokenResource{}
	_ resource.ResourceWithConfigure      = &AccessTokenResource{}
	_ resource.ResourceWithImportState    = &AccessTokenResource{}
	_ resource.ResourceWithValidateConfig = &AccessTokenResource{}
	_ resource.ResourceWithModifyPlan     = &AccessTokenResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewAccessTokenResource)
}

func NewAccessTokenResource() resource.Resource {
	return &AccessTokenResource{}
}

// AccessTokenResource manages a fine-grained access token (ADR-0068).
type AccessTokenResource struct {
	osaasContext *osaasclient.Context
}

type AccessTokenResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Template        types.String `tfsdk:"template"`
	Policy          types.String `tfsdk:"policy"`
	EffectivePolicy types.String `tfsdk:"effective_policy"`
	ExpiresInDays   types.Int64  `tfsdk:"expires_in_days"`
	ExpiresAt       types.String `tfsdk:"expires_at"`
	Status          types.String `tfsdk:"status"`
	Enforcement     types.String `tfsdk:"enforcement"`
	PolicyVersion   types.Int64  `tfsdk:"policy_version"`
	LastUsedAt      types.String `tfsdk:"last_used_at"`
	Token           types.String `tfsdk:"token"`
}

// accessTokenTemplates are the policy templates token-service expands at creation.
var accessTokenTemplates = []string{"viewer", "deployer", "ci-publisher", "agent-runner", "mcp-readonly", "full"}

const maxAccessTokenDays = 90

func (r *AccessTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_token"
}

// templateRequiresReplace replaces the token when the template changes, except when the
// template is removed in favour of a policy: that is the way to change the scope of the same
// token, as a policy replacement.
func templateRequiresReplace(ctx context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
	if req.PlanValue.IsNull() {
		var policy types.String
		if diags := req.Config.GetAttribute(ctx, path.Root("policy"), &policy); !diags.HasError() && !policy.IsNull() {
			return
		}
	}
	resp.RequiresReplace = true
}

func (r *AccessTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a fine-grained access token: a token for one workspace whose permissions are a policy " +
			"kept by the platform, not in the token. Use it for CI, agents and other automation that should hold less " +
			"than a full personal access token. Create it from a `template` or an explicit `policy`.\n\n" +
			"The token value is returned by the platform once, when the token is created. It is exposed as the " +
			"sensitive attribute `token` and stored in the Terraform state, so protect the state (use an encrypted remote " +
			"backend). The expiry of a token cannot be extended: to rotate it, run " +
			"`terraform apply -replace=osc_access_token.<name>`.\n\n" +
			"The policy, name, description, status and enforcement change in place and the token value stays the same. " +
			"A change of `template` or `expires_in_days` replaces the token.\n\n" +
			"An imported token has `template` and `expires_in_days` null in the state, because the platform does not report " +
			"them. Setting either of them in the configuration afterwards replaces the token: a new token value is " +
			"created and the old token is revoked. Describe an imported token with `policy` only.\n\n" +
			"A token stays readable in the store after its expiry, and the plan shows no change, although the platform no " +
			"longer accepts it. The provider does not rotate it: replace it yourself with " +
			"`terraform apply -replace=osc_access_token.<name>`.\n\n" +
			"Creating tokens is not enabled in every environment yet; where it is not, creating the resource fails with an " +
			"error saying so.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The token id (`patId`). Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of the token, shown in the console and in audit events. Changed in place.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
				Description: "What the token is for. Changed in place.",
			},
			"template": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "A named policy the platform expands into a policy when the token is created: one of `viewer`, " +
					"`deployer`, `ci-publisher`, `agent-runner`, `mcp-readonly` or `full`. Set exactly one of `template` and " +
					"`policy`. The template is not stored as a reference, so changing it replaces the token. To change the " +
					"scope of the same token, use `policy`: replacing `template` with `policy` in the configuration " +
					"replaces the policy of the existing token in place.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplaceIf(templateRequiresReplace,
					"Replaces the token when the template changes, but not when it is removed in favour of a policy.",
					"Replaces the token when the template changes, but not when it is removed in favour of a policy.")},
			},
			"policy": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "The policy of the token as a JSON string, normally `jsonencode(...)`: " +
					"`{\"version\": 1, \"statements\": [{\"actions\": [\"myapps:read\"], \"resources\": [\"myapps/*\"]}]}`. " +
					"Set exactly one of `template` and `policy`. A change replaces the policy of the same token in place, " +
					"guarded by `policy_version`: if the policy was changed outside Terraform since it was last read, the " +
					"apply fails and a new plan is needed. A policy that means the same as the stored one, such as one " +
					"with other formatting, changes nothing. A policy cannot give more permissions than the caller of the " +
					"API has.",
			},
			"effective_policy": schema.StringAttribute{
				Computed:    true,
				Description: "The policy as the platform stores it, as normalised JSON. For a token created from a template it is the policy the template expanded to.",
				PlanModifiers: []planmodifier.String{
					policyDerivedString{},
				},
			},
			"expires_in_days": schema.Int64Attribute{
				Optional: true,
				MarkdownDescription: "Lifetime of the token in days, 1 to 90. Omitted means 90. The expiry is fixed when the " +
					"token is created, so changing this replaces the token.",
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"expires_at": schema.StringAttribute{
				Computed:      true,
				Description:   "When the token expires (RFC 3339).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("active"),
				MarkdownDescription: "`active` or `disabled`. A disabled token is refused by the platform until it is activated " +
					"again. Defaults to `active`, and a new token is always created active, so a change that replaces the token " +
					"(`template` or `expires_in_days`) cannot also set `disabled`: apply it first, then disable the token. " +
					"Changed in place.",
			},
			"enforcement": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString("enforce"),
				MarkdownDescription: "`enforce` or `audit`. With `audit` the platform records what the policy would have refused " +
					"but allows it. Defaults to `enforce`. Changed in place.",
			},
			"policy_version": schema.Int64Attribute{
				Computed:    true,
				Description: "Version of the stored policy, raised by every policy change.",
				PlanModifiers: []planmodifier.Int64{
					policyDerivedInt64{},
				},
			},
			"last_used_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the platform last saw the token used (RFC 3339), refreshed on every read. Null while it has not been used.",
			},
			"token": schema.StringAttribute{
				Computed:  true,
				Sensitive: true,
				MarkdownDescription: "The token value, to use as an OSC access token. The platform returns it only when the token is " +
					"created, so it is stored in the Terraform state and a read never changes it: protect the state, for " +
					"example with an encrypted remote backend. It is null for an imported token, because the value " +
					"cannot be read back. The expiry cannot be extended: to rotate the token, run " +
					"`terraform apply -replace=osc_access_token.<name>`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

// policyWillChange reports whether applying the plan sends a policy replacement: the
// configuration sets a policy that does not mean the same as the one stored.
func policyWillChange(ctx context.Context, config tfsdk.Config, state tfsdk.State) bool {
	if state.Raw.IsNull() {
		return true
	}
	var policy, effective types.String
	if config.GetAttribute(ctx, path.Root("policy"), &policy).HasError() ||
		state.GetAttribute(ctx, path.Root("effective_policy"), &effective).HasError() {
		return true
	}
	if policy.IsNull() {
		return false
	}
	if policy.IsUnknown() || effective.IsNull() || effective.IsUnknown() {
		return true
	}
	return !sameJSON([]byte(policy.ValueString()), []byte(effective.ValueString()))
}

// policyDerivedString keeps a value taken from the stored policy while no policy
// replacement is planned, so that a plan which cannot change it does not show it as unknown.
type policyDerivedString struct{}

func (policyDerivedString) Description(context.Context) string {
	return "Keeps the state value unless the policy is replaced."
}
func (m policyDerivedString) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (policyDerivedString) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !resp.PlanValue.IsUnknown() || policyWillChange(ctx, req.Config, req.State) {
		return
	}
	resp.PlanValue = req.StateValue
}

type policyDerivedInt64 struct{}

func (policyDerivedInt64) Description(context.Context) string {
	return "Keeps the state value unless the policy is replaced."
}
func (m policyDerivedInt64) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}
func (policyDerivedInt64) PlanModifyInt64(ctx context.Context, req planmodifier.Int64Request, resp *planmodifier.Int64Response) {
	if req.StateValue.IsNull() || !resp.PlanValue.IsUnknown() || policyWillChange(ctx, req.Config, req.State) {
		return
	}
	resp.PlanValue = req.StateValue
}

func (r *AccessTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *AccessTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config AccessTokenResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !config.Template.IsUnknown() && !config.Policy.IsUnknown() {
		switch {
		case config.Template.IsNull() && config.Policy.IsNull():
			resp.Diagnostics.AddError("Template or policy required",
				"Set exactly one of `template` and `policy`: the template or policy decides what the token may do.")
		case !config.Template.IsNull() && !config.Policy.IsNull():
			resp.Diagnostics.AddError("Template and policy are exclusive",
				"Set exactly one of `template` and `policy`. A template expands to a policy when the token is created; "+
					"to change the scope of an existing token, replace `template` with `policy`.")
		}
	}
	if isSet(config.Template) {
		valid := false
		for _, t := range accessTokenTemplates {
			valid = valid || t == config.Template.ValueString()
		}
		if !valid {
			resp.Diagnostics.AddAttributeError(path.Root("template"), "Unknown template",
				fmt.Sprintf("The template must be one of %v, got %q.", accessTokenTemplates, config.Template.ValueString()))
		}
	}
	if !config.Policy.IsNull() && !config.Policy.IsUnknown() {
		var v map[string]interface{}
		if err := json.Unmarshal([]byte(config.Policy.ValueString()), &v); err != nil || v == nil {
			resp.Diagnostics.AddAttributeError(path.Root("policy"), "Invalid policy",
				"The policy must be a JSON object, for example `jsonencode({version = 1, statements = [...]})`.")
		}
	}
	if !config.ExpiresInDays.IsNull() && !config.ExpiresInDays.IsUnknown() {
		if d := config.ExpiresInDays.ValueInt64(); d < 1 || d > maxAccessTokenDays {
			resp.Diagnostics.AddAttributeError(path.Root("expires_in_days"), "Invalid lifetime",
				fmt.Sprintf("expires_in_days must be between 1 and %d, got %d.", maxAccessTokenDays, d))
		}
	}
	if isSet(config.Status) && config.Status.ValueString() != "active" && config.Status.ValueString() != "disabled" {
		resp.Diagnostics.AddAttributeError(path.Root("status"), "Invalid status",
			fmt.Sprintf("status must be `active` or `disabled`, got %q.", config.Status.ValueString()))
	}
	if isSet(config.Enforcement) && config.Enforcement.ValueString() != "enforce" && config.Enforcement.ValueString() != "audit" {
		resp.Diagnostics.AddAttributeError(path.Root("enforcement"), "Invalid enforcement",
			fmt.Sprintf("enforcement must be `enforce` or `audit`, got %q.", config.Enforcement.ValueString()))
	}
}

const createDisabledMessage = "A new access token is always created active. Create it without `status`, then set " +
	"`status = \"disabled\"` in a following apply."

// ModifyPlan refuses a new token that is to be disabled at once: creating one is a single
// request, and a second request to disable it could fail after the token was minted.
//
// It refuses the replacement of a token the same way. A replacement deletes the old token
// and then creates a new one, and a create with `disabled` would be refused after the
// delete; so a plan that both replaces the token (template or expires_in_days changes) and
// sets `disabled` is refused here, before anything is deleted. Terraform core also plans a
// replacement again as a create, which the first check covers; the second check does not
// rely on that.
func (r *AccessTokenResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}
	var plan AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || !isSet(plan.Status) || plan.Status.ValueString() != "disabled" {
		return
	}
	if req.State.Raw.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("status"), "A new token cannot be disabled", createDisabledMessage)
		return
	}
	var state AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// A value that comes from another resource is unknown at plan time, so whether the token
	// is replaced is not known yet; refusing would blame a change that may not happen.
	if plan.Template.IsUnknown() || plan.ExpiresInDays.IsUnknown() {
		return
	}
	templateReplaced := !plan.Template.Equal(state.Template) && !(plan.Template.IsNull() && !plan.Policy.IsNull())
	if templateReplaced || !plan.ExpiresInDays.Equal(state.ExpiresInDays) {
		resp.Diagnostics.AddAttributeError(path.Root("status"), "A replaced token cannot be disabled",
			"Changing `template` or `expires_in_days` replaces the access token, and a new token is always created active, "+
				"so the replacement cannot also be disabled. Nothing was changed. Apply the change with `status` omitted "+
				"(or `\"active\"`), then set `status = \"disabled\"` in a following apply.")
	}
}

// applyAccessToken copies what the platform reports into the model. Template, policy,
// expires_in_days and token are not reported and are left as they are, except that a policy
// that no longer means what the configuration says is replaced by the stored one.
func applyAccessToken(rec *accessTokenRecord, model *AccessTokenResourceModel) error {
	effective, err := normalizeJSON(rec.Policy)
	if err != nil {
		return fmt.Errorf("the platform returned a policy that is not valid JSON: %w", err)
	}
	model.ID = types.StringValue(rec.PatID)
	model.Name = types.StringValue(rec.Name)
	model.Description = types.StringValue(rec.Description)
	model.Status = types.StringValue(rec.Status)
	model.Enforcement = types.StringValue(rec.Enforcement)
	model.PolicyVersion = types.Int64Value(rec.PolicyVersion)
	model.EffectivePolicy = types.StringValue(effective)
	model.ExpiresAt = types.StringValue(time.Unix(rec.Exp, 0).UTC().Format(time.RFC3339))
	if rec.LastUsedAt == "" {
		model.LastUsedAt = types.StringNull()
	} else {
		model.LastUsedAt = types.StringValue(rec.LastUsedAt)
	}
	if !model.Policy.IsNull() && !model.Policy.IsUnknown() && !sameJSON([]byte(model.Policy.ValueString()), rec.Policy) {
		model.Policy = types.StringValue(effective)
	}
	return nil
}

func (r *AccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(plan.Status) && plan.Status.ValueString() != "active" {
		resp.Diagnostics.AddAttributeError(path.Root("status"), "A new token cannot be disabled", createDisabledMessage)
		return
	}
	body := map[string]interface{}{"name": plan.Name.ValueString()}
	if isSet(plan.Description) {
		body["description"] = plan.Description.ValueString()
	}
	if isSet(plan.Template) {
		body["template"] = plan.Template.ValueString()
	}
	if isSet(plan.Policy) {
		body["policy"] = json.RawMessage(plan.Policy.ValueString())
	}
	if !plan.ExpiresInDays.IsNull() && !plan.ExpiresInDays.IsUnknown() {
		body["expiresInSeconds"] = plan.ExpiresInDays.ValueInt64() * 24 * 3600
	}
	if isSet(plan.Enforcement) {
		body["enforcement"] = plan.Enforcement.ValueString()
	}

	// One request, never repeated, and nothing after it that can fail before the state is
	// written: the response is the only time the token value is given.
	rec, err := createAccessToken(r.osaasContext, body)
	if err != nil {
		var ae *apiError
		switch {
		case errors.As(err, &ae) && ae.StatusCode == http.StatusNotFound:
			resp.Diagnostics.AddError("Access tokens are not enabled in this environment",
				fmt.Sprintf("Creating access tokens is not enabled in the %q environment yet (%s). Nothing was created.",
					r.osaasContext.GetEnvironment(), err.Error()))
		case isTokenOutcomeUnknown(err):
			resp.Diagnostics.AddError("The outcome of creating the access token is unknown",
				fmt.Sprintf("The request to create the access token %q was sent but no answer came back (%s), so a token may "+
					"have been created. The request is not repeated. Check with `osc token list` (or GET /tokens, looking for "+
					"the name) and delete the token if it exists before applying again.", plan.Name.ValueString(), err.Error()))
		default:
			resp.Diagnostics.AddError("Failed to create access token", err.Error())
		}
		return
	}

	state := plan
	if err := applyAccessToken(rec, &state); err != nil {
		resp.Diagnostics.AddError("Failed to read the created access token", err.Error())
		return
	}
	if rec.Token == "" {
		state.Token = types.StringNull()
	} else {
		state.Token = types.StringValue(rec.Token)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if rec.Token == "" {
		resp.Diagnostics.AddError("The platform did not return the token value",
			fmt.Sprintf("The access token %s was created but the answer had no token value, which is only given once. "+
				"Destroy it and create it again.", rec.PatID))
	}
}

func (r *AccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Gone is confirmed over several attempts, so one transient "token not found" does not
	// drop the token from state and plan a second one. Only the exact reason counts as gone.
	rec, exists, err := readConfirmed(func() (*accessTokenRecord, bool, error) {
		rec, err := getAccessToken(r.osaasContext, state.ID.ValueString())
		if isTokenNotFound(err) {
			return nil, false, nil
		}
		return rec, err == nil && rec != nil, err
	})
	if err != nil || !exists {
		switch {
		case err == nil:
			resp.State.RemoveResource(ctx)
		case isRoleLookupUnavailable(err):
			// The token exists; dropping it from state would plan a second one.
			resp.Diagnostics.AddError("Access token cannot be read right now",
				fmt.Sprintf("The platform could not look up your role and so did not show access token %s. The token is kept in "+
					"the state. Try again shortly.", state.ID.ValueString()))
		default:
			resp.Diagnostics.AddError("Failed to read access token", err.Error())
		}
		return
	}
	if err := applyAccessToken(rec, &state); err != nil {
		resp.Diagnostics.AddError("Failed to read access token", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state AccessTokenResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	// What is kept whatever happens: the token value, and what the configuration says about
	// the template, the policy and the lifetime.
	next := state
	next.Template, next.Policy, next.ExpiresInDays = plan.Template, plan.Policy, plan.ExpiresInDays

	fail := func(summary string, err error) {
		// Keep what was already applied, so that the state does not claim otherwise.
		resp.Diagnostics.Append(resp.State.Set(ctx, &next)...)
		resp.Diagnostics.AddError(summary, err.Error())
	}

	if isSet(plan.Policy) && (state.EffectivePolicy.IsNull() || !sameJSON([]byte(plan.Policy.ValueString()), []byte(state.EffectivePolicy.ValueString()))) {
		rec, err := replaceAccessTokenPolicy(r.osaasContext, id, json.RawMessage(plan.Policy.ValueString()), state.PolicyVersion.ValueInt64())
		if err != nil {
			var ae *apiError
			if errors.As(err, &ae) && ae.StatusCode == http.StatusConflict {
				resp.Diagnostics.AddError("The policy changed outside Terraform",
					fmt.Sprintf("The policy of access token %s changed outside Terraform since version %d. Nothing was changed. "+
						"Run terraform plan again to see the difference.", id, state.PolicyVersion.ValueInt64()))
				return
			}
			resp.Diagnostics.AddError("Failed to replace the policy of the access token", err.Error())
			return
		}
		if err := applyAccessToken(rec, &next); err != nil {
			resp.Diagnostics.AddError("Failed to read the access token", err.Error())
			return
		}
	}

	fields := map[string]interface{}{}
	if !plan.Name.Equal(state.Name) {
		fields["name"] = plan.Name.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		fields["description"] = plan.Description.ValueString()
	}
	if !plan.Status.Equal(state.Status) {
		fields["status"] = plan.Status.ValueString()
	}
	if !plan.Enforcement.Equal(state.Enforcement) {
		fields["enforcement"] = plan.Enforcement.ValueString()
	}
	if len(fields) > 0 {
		rec, err := patchAccessToken(r.osaasContext, id, fields)
		if err != nil {
			fail("Failed to update the access token", err)
			return
		}
		if err := applyAccessToken(rec, &next); err != nil {
			fail("Failed to read the access token", err)
			return
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &next)...)
}

func (r *AccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state AccessTokenResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteAccessToken(r.osaasContext, state.ID.ValueString()); err != nil && !isTokenNotFound(err) {
		resp.Diagnostics.AddError("Failed to delete access token", err.Error())
	}
}

// ImportState reads the token with the given id (`patId`) into the state. The token value
// is not readable, so `token` stays null; neither is the template, nor the lifetime, kept.
func (r *AccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	rec, err := getAccessToken(r.osaasContext, req.ID)
	if err != nil {
		if isTokenNotFound(err) {
			resp.Diagnostics.AddError("Access token not found",
				fmt.Sprintf("The workspace has no access token with id %q that you can see. List them with `osc token list`.", req.ID))
			return
		}
		resp.Diagnostics.AddError("Failed to read access token", err.Error())
		return
	}
	model := AccessTokenResourceModel{
		Template:      types.StringNull(),
		Policy:        types.StringNull(),
		ExpiresInDays: types.Int64Null(),
		Token:         types.StringNull(),
	}
	if err := applyAccessToken(rec, &model); err != nil {
		resp.Diagnostics.AddError("Failed to read access token", err.Error())
		return
	}
	model.Policy = model.EffectivePolicy
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
