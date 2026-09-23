package provider

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

const myAppBuildTimeout = 15 * time.Minute

var (
	_ resource.Resource                   = &MyAppResource{}
	_ resource.ResourceWithConfigure      = &MyAppResource{}
	_ resource.ResourceWithImportState    = &MyAppResource{}
	_ resource.ResourceWithValidateConfig = &MyAppResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewMyAppResource)
}

func NewMyAppResource() resource.Resource {
	return &MyAppResource{}
}

// MyAppResource manages a My App: the tenant's own application, built from a git
// repository and run on Web Runner.
type MyAppResource struct {
	osaasContext *osaasclient.Context
}

type MyAppResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Type             types.String `tfsdk:"type"`
	GitURL           types.String `tfsdk:"git_url"`
	SourceRef        types.String `tfsdk:"source_ref"`
	SubPath          types.String `tfsdk:"sub_path"`
	GitToken         types.String `tfsdk:"git_token"`
	GitCredential    types.String `tfsdk:"git_credential"`
	ConfigService    types.String `tfsdk:"config_service"`
	ConfigAPIKey     types.String `tfsdk:"config_api_key"`
	HighAvailability types.Bool   `tfsdk:"high_availability"`
	RebuildTrigger   types.String `tfsdk:"rebuild_trigger"`
	WaitForReady     types.Bool   `tfsdk:"wait_for_ready"`
	URL              types.String `tfsdk:"url"`
	ManagedDomain    types.String `tfsdk:"managed_domain"`
	DomainServiceID  types.String `tfsdk:"domain_service_id"`
	BuildStatus      types.String `tfsdk:"build_status"`
}

var gitCredentialName = regexp.MustCompile(`^[a-z0-9-]{1,40}$`)

func (r *MyAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_my_app"
}

func (r *MyAppResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a My App: your own application, built from a git repository and run on Web Runner.\n\n" +
			"On every build the platform clones the repository, then runs `npm install`, `npm run build` and `npm start` " +
			"(or the equivalent for the runtime) from the repository root, or from `sub_path`. The app must listen on the " +
			"`PORT` it is given. Environment variables come from the bound parameter store (`config_service`); manage the " +
			"store with `osc_parameter_store` and its values with `osc_parameter`.\n\n" +
			"Terraform manages the app, not its releases: pushing to the tracked branch does not redeploy. Change " +
			"`source_ref` to deploy a tag, or change `rebuild_trigger` to rebuild from the current head.\n\n" +
			"Attach a custom domain with `osc_domain`, using `domain_service_id` and `id` of this resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The app id, which the platform derives from the name. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "App name, unique within the workspace. Changing it replaces the app.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Optional:      true,
				Computed:      true,
				Default:       stringdefault.StaticString("nodejs"),
				Description:   "Runtime: `nodejs`, `python`, `wasm`, `golang` or `dotnet`. Changing it replaces the app.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"git_url": schema.StringAttribute{
				Required: true,
				Description: "HTTPS URL of the git repository, on any host (GitHub, OSC managed Gitea, GitLab, ...), without a " +
					"`#ref`. Changing it replaces the app.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"source_ref": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: "Branch or tag to build. Commit SHAs are not supported; tag the commit instead. Changing it " +
					"deploys that ref with a rolling restart. When unset the app builds the default branch, and the ref " +
					"the platform reports is kept in state.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"sub_path": schema.StringAttribute{
				Optional: true,
				Description: "Directory within the repository to build and start from, for monorepos. Note that the platform " +
					"does not load the parameter store for an app with a `sub_path`; the app must fetch it itself from " +
					"`APP_CONFIG_URL`. Changing it replaces the app.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"git_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Access token for a private repository. Conflicts with `git_credential`.",
			},
			"git_credential": schema.StringAttribute{
				Optional: true,
				Description: "Name of a git credential stored in the workspace, used instead of `git_token` so that the token " +
					"never passes through Terraform. Conflicts with `git_token`.",
			},
			"config_service": schema.StringAttribute{
				Optional: true,
				Description: "Name of the parameter store whose values become the app's environment variables. Changing it " +
					"rebinds the app with a rolling restart.",
			},
			"config_api_key": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "API key of the parameter store, needed to decrypt its secret values. When unset it is read from " +
					"the parameter store, which is what you want unless the key was rotated outside Terraform.",
			},
			"high_availability": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Run a standby instance so that rebuilds are blue-green deployments with no downtime. Only " +
					"for `nodejs` and `python` apps, and only on paid plans.",
			},
			"rebuild_trigger": schema.StringAttribute{
				Optional: true,
				Description: "Any value. Changing it rebuilds the app from the head of `source_ref` with a fresh image, " +
					"for example a commit SHA or a release version from CI. For an app with `high_availability` the " +
					"rebuild is a blue-green deployment.",
			},
			"wait_for_ready": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Wait for the build to finish after create and after any change that restarts the app, up to " +
					"fifteen minutes, and fail if the build fails.",
			},
			"url": schema.StringAttribute{
				Computed:      true,
				Description:   "The app's URL on the Web Runner host.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"managed_domain": schema.StringAttribute{
				Computed:      true,
				Description:   "The hostname the platform assigns the app, `<hash>.apps.osaas.io`. Empty until assigned.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain_service_id": schema.StringAttribute{
				Computed:      true,
				Description:   "The `service_id` to use in an `osc_domain` for this app, together with `id` as `instance_name`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"build_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the latest build: `building`, `running`, `failed` or `unknown`.",
			},
		},
	}
}

func (r *MyAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *MyAppResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config MyAppResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.GitToken) && isSet(config.GitCredential) {
		resp.Diagnostics.AddAttributeError(path.Root("git_credential"), "Conflicting git credentials",
			"Set either git_token or git_credential, not both.")
	}
	if isSet(config.GitCredential) && !gitCredentialName.MatchString(config.GitCredential.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("git_credential"), "Invalid git credential name",
			"A git credential name is 1-40 lowercase letters, digits and hyphens, as listed in the workspace's git credentials.")
	}
	if isSet(config.Type) {
		switch config.Type.ValueString() {
		case "nodejs", "python", "wasm", "golang", "dotnet":
		default:
			resp.Diagnostics.AddAttributeError(path.Root("type"), "Unsupported runtime",
				fmt.Sprintf("type must be one of nodejs, python, wasm, golang or dotnet, got %q.", config.Type.ValueString()))
		}
	}
	if isSet(config.GitURL) && strings.Contains(config.GitURL.ValueString(), "#") {
		resp.Diagnostics.AddAttributeError(path.Root("git_url"), "Ref in git_url",
			"Put the branch or tag in source_ref rather than after # in git_url.")
	}
	if config.HighAvailability.ValueBool() && isSet(config.Type) {
		if t := config.Type.ValueString(); t != "nodejs" && t != "python" {
			resp.Diagnostics.AddAttributeError(path.Root("high_availability"), "High availability not supported",
				fmt.Sprintf("High availability is only available for nodejs and python apps, not %s.", t))
		}
	}
}

// configAPIKey is the key to send when binding the app to its parameter store: the
// configured one, or the store's own when the store has secrets enabled.
func (r *MyAppResource) configAPIKey(plan MyAppResourceModel) (string, error) {
	store := plan.ConfigService.ValueString()
	if store == "" {
		return "", nil
	}
	if isSet(plan.ConfigAPIKey) {
		return plan.ConfigAPIKey.ValueString(), nil
	}
	info, err := getParameterStore(r.osaasContext, store)
	if err != nil {
		return "", fmt.Errorf("could not read the API key of parameter store %q: %w", store, err)
	}
	return info.ConfigAPIKey, nil
}

func gitCredentialRef(name string) string {
	if name == "" {
		return ""
	}
	return "user.gitcred." + name
}

// refresh copies what the platform reports into the model.
func (r *MyAppResource) refresh(app *myApp, model *MyAppResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	model.ID = types.StringValue(app.ID)
	model.Name = types.StringValue(app.Name)
	if app.Type != "" {
		model.Type = types.StringValue(app.Type)
	}
	// Keep the configured spelling unless the repository actually changed.
	if src := app.sourceURL(); src != "" && normalizeGitURL(src) != normalizeGitURL(model.GitURL.ValueString()) {
		model.GitURL = types.StringValue(src)
	}
	if app.SourceRef != "" || !model.SourceRef.IsNull() {
		model.SourceRef = types.StringValue(app.SourceRef)
	}
	if app.SubPath != "" {
		model.SubPath = types.StringValue(app.SubPath)
	}
	if bound := app.boundConfigService(); bound != "" || !model.ConfigService.IsNull() {
		model.ConfigService = types.StringValue(bound)
	}
	model.HighAvailability = types.BoolValue(app.HAEnabled)
	model.URL = types.StringValue(app.URL)
	model.BuildStatus = types.StringValue(app.BuildStatus)
	model.DomainServiceID = types.StringValue(webRunnerServiceID)
	managed := app.AppDNS
	if managed == "" {
		var err error
		if managed, err = managedDomain(r.osaasContext, webRunnerServiceID, app.ID); err != nil {
			diags.AddWarning("Could not read the app's domains", err.Error())
		}
	}
	model.ManagedDomain = types.StringValue(managed)
	return diags
}

func normalizeGitURL(u string) string {
	u = strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(u), "/"), ".git")
	return strings.ToLower(u)
}

// waitBuild waits for the build when the configuration asks for it.
func (r *MyAppResource) waitBuild(model MyAppResourceModel, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	if !model.WaitForReady.ValueBool() {
		return diags
	}
	if _, err := waitForMyAppBuild(r.osaasContext, id, myAppBuildTimeout); err != nil {
		diags.AddError("App build did not succeed", err.Error())
	}
	return diags
}

func (r *MyAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MyAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := map[string]interface{}{
		"name":   plan.Name.ValueString(),
		"type":   plan.Type.ValueString(),
		"gitUrl": plan.GitURL.ValueString(),
	}
	if isSet(plan.SourceRef) {
		body["sourceRef"] = plan.SourceRef.ValueString()
	}
	if isSet(plan.SubPath) {
		body["subPath"] = plan.SubPath.ValueString()
	}
	if isSet(plan.GitToken) {
		body["gitToken"] = plan.GitToken.ValueString()
	}
	if isSet(plan.GitCredential) {
		body["gitCredentialRef"] = gitCredentialRef(plan.GitCredential.ValueString())
	}
	if isSet(plan.ConfigService) {
		body["configService"] = plan.ConfigService.ValueString()
		key, err := r.configAPIKey(plan)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config_service"), "Failed to read parameter store", err.Error())
			return
		}
		if key != "" {
			body["configApiKey"] = key
		}
	}

	created, err := createMyApp(r.osaasContext, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create app", fmt.Sprintf("Could not create app %q: %s", plan.Name.ValueString(), err.Error()))
		return
	}
	id := created.ID
	if id == "" {
		id = plan.Name.ValueString()
	}

	// Record the app before waiting, so a failed build leaves it in state to fix or destroy.
	state := plan
	state.ID = types.StringValue(id)
	state.DomainServiceID = types.StringValue(webRunnerServiceID)
	state.URL = types.StringValue(created.URL)
	state.ManagedDomain = types.StringValue("")
	state.BuildStatus = types.StringValue(created.BuildStatus)
	if state.SourceRef.IsUnknown() {
		state.SourceRef = types.StringValue("")
	}
	state.HighAvailability = types.BoolValue(false)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	resp.Diagnostics.Append(r.waitBuild(plan, id)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.HighAvailability.ValueBool() {
		if err := setMyAppHA(r.osaasContext, id, true); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("high_availability"), "Failed to enable high availability", err.Error())
			return
		}
	}

	app, err := getMyApp(r.osaasContext, id)
	if err != nil || app == nil {
		resp.Diagnostics.AddError("Failed to read app after create", fmt.Sprintf("app %q: %v", id, err))
		return
	}
	resp.Diagnostics.Append(r.refresh(app, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MyAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MyAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	app, err := getMyApp(r.osaasContext, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read app", err.Error())
		return
	}
	if app == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(r.refresh(app, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MyAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MyAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	// Each of these restarts the app, so wait for one to finish before starting the next.
	if !plan.GitToken.Equal(state.GitToken) || !plan.GitCredential.Equal(state.GitCredential) {
		if err := setMyAppGitToken(r.osaasContext, id, plan.GitToken.ValueString(), gitCredentialRef(plan.GitCredential.ValueString())); err != nil {
			resp.Diagnostics.AddError("Failed to update git credentials", err.Error())
			return
		}
		resp.Diagnostics.Append(r.waitBuild(plan, id)...)
	}

	if !plan.ConfigService.Equal(state.ConfigService) || !plan.ConfigAPIKey.Equal(state.ConfigAPIKey) {
		key, err := r.configAPIKey(plan)
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("config_service"), "Failed to read parameter store", err.Error())
			return
		}
		if err := setMyAppConfig(r.osaasContext, id, plan.ConfigService.ValueString(), key); err != nil {
			resp.Diagnostics.AddError("Failed to bind parameter store", err.Error())
			return
		}
		resp.Diagnostics.Append(r.waitBuild(plan, id)...)
	}

	if !plan.SourceRef.IsUnknown() && !plan.SourceRef.Equal(state.SourceRef) {
		if err := setMyAppSourceRef(r.osaasContext, id, plan.SourceRef.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("source_ref"), "Failed to change source ref", err.Error())
			return
		}
		resp.Diagnostics.Append(r.waitBuild(plan, id)...)
	}

	if !plan.RebuildTrigger.Equal(state.RebuildTrigger) && !plan.RebuildTrigger.IsNull() {
		if err := restartMyApp(r.osaasContext, id, true); err != nil {
			resp.Diagnostics.AddError("Failed to rebuild app", err.Error())
			return
		}
		resp.Diagnostics.Append(r.waitBuild(plan, id)...)
	}

	if !plan.HighAvailability.Equal(state.HighAvailability) {
		if err := setMyAppHA(r.osaasContext, id, plan.HighAvailability.ValueBool()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("high_availability"), "Failed to change high availability", err.Error())
			return
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	app, err := getMyApp(r.osaasContext, id)
	if err != nil || app == nil {
		resp.Diagnostics.AddError("Failed to read app after update", fmt.Sprintf("app %q: %v", id, err))
		return
	}
	newState := plan
	newState.ID = state.ID
	resp.Diagnostics.Append(r.refresh(app, &newState)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *MyAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MyAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteMyApp(r.osaasContext, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete app", err.Error())
	}
}

// ImportState takes the app id, as listed by the OSC CLI or dashboard. Credentials are
// not readable from the platform, so git_token and config_api_key start out unset.
func (r *MyAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	app, err := getMyApp(r.osaasContext, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read app", err.Error())
		return
	}
	if app == nil {
		resp.Diagnostics.AddError("App not found", fmt.Sprintf("The workspace has no app with id %q.", req.ID))
		return
	}
	model := MyAppResourceModel{
		GitURL:        types.StringValue(app.sourceURL()),
		SourceRef:     types.StringValue(app.SourceRef),
		SubPath:       types.StringNull(),
		GitToken:      types.StringNull(),
		GitCredential: types.StringNull(),
		ConfigService: types.StringNull(),
		ConfigAPIKey:  types.StringNull(),
		WaitForReady:  types.BoolValue(true),
	}
	resp.Diagnostics.Append(r.refresh(app, &model)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
