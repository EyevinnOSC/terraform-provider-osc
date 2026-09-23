package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &ParameterStoreResource{}
	_ resource.ResourceWithConfigure      = &ParameterStoreResource{}
	_ resource.ResourceWithImportState    = &ParameterStoreResource{}
	_ resource.ResourceWithValidateConfig = &ParameterStoreResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewParameterStoreResource)
}

func NewParameterStoreResource() resource.Resource {
	return &ParameterStoreResource{}
}

// ParameterStoreResource manages a parameter store: an app-config-svc instance whose
// encryption and API keys the platform holds, so that a My App can be bound to it.
type ParameterStoreResource struct {
	osaasContext *osaasclient.Context
}

type ParameterStoreResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	RedisURL      types.String `tfsdk:"redis_url"`
	EnableSecrets types.Bool   `tfsdk:"enable_secrets"`
	URL           types.String `tfsdk:"url"`
	ConfigAPIKey  types.String `tfsdk:"config_api_key"`
}

var parameterStoreName = regexp.MustCompile(`^[A-Za-z0-9]+$`)

func (r *ParameterStoreResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter_store"
}

func (r *ParameterStoreResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a parameter store, the source of a My App's environment variables. Bind an app to it " +
			"with `config_service` on `osc_my_app`, and manage its values with `osc_parameter`.\n\n" +
			"This is not the same as an `osc_instance` of `eyevinn-app-config-svc`: the platform creates the store with an " +
			"encryption key and an API key it keeps as secrets, so secret values can be stored and a bound app can " +
			"decrypt them.\n\n" +
			"Destroying the store deletes every value in it, and the Valkey instance OSC created for it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The store name. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Store name: letters and digits only. Changing it replaces the store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"redis_url": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Valkey or Redis URL to keep the values in. When unset OSC creates a Valkey instance for the " +
					"store, which is the usual choice. Changing it replaces the store.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enable_secrets": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Create the store with an encryption key and an API key, so that it can hold secret values. " +
					"Changing it replaces the store.",
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
			},
			"url": schema.StringAttribute{
				Computed:      true,
				Description:   "URL of the store's API.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"config_api_key": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "The store's API key, which reads secret values in plain text. Empty when secrets are disabled.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *ParameterStoreResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *ParameterStoreResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ParameterStoreResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.Name) && !parameterStoreName.MatchString(config.Name.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid parameter store name",
			"A parameter store name is letters and digits only; no hyphens or underscores.")
	}
}

// refresh reads the store's instance and keys. It returns false if the store is gone.
func (r *ParameterStoreResource) refresh(model *ParameterStoreResourceModel) (bool, error) {
	name := model.Name.ValueString()
	instance, _, _, err := parameterStoreInstance(r.osaasContext, name)
	if err != nil {
		return false, err
	}
	if instance == nil {
		return false, nil
	}
	info, err := getParameterStore(r.osaasContext, name)
	if err != nil {
		return false, err
	}
	model.ID = types.StringValue(name)
	u, _ := instance["url"].(string)
	model.URL = types.StringValue(u)
	model.EnableSecrets = types.BoolValue(info.IsSecure)
	if info.ConfigAPIKey != "" || model.ConfigAPIKey.IsNull() || model.ConfigAPIKey.IsUnknown() {
		model.ConfigAPIKey = types.StringValue(info.ConfigAPIKey)
	}
	return true, nil
}

func (r *ParameterStoreResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ParameterStoreResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	body := map[string]interface{}{
		"name":          name,
		"enableSecrets": plan.EnableSecrets.ValueBool(),
	}
	if isSet(plan.RedisURL) {
		body["redisUrl"] = plan.RedisURL.ValueString()
	}
	created, err := createParameterStore(r.osaasContext, body)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create parameter store", fmt.Sprintf("Could not create parameter store %q: %s", name, err.Error()))
		return
	}

	state := plan
	state.ID = types.StringValue(name)
	state.URL = types.StringValue("")
	state.ConfigAPIKey = types.StringValue(created.ConfigAPIKey)

	// The store is created asynchronously; wait for its instance to answer.
	instance, service, token, err := parameterStoreInstance(r.osaasContext, name)
	if err == nil && instance != nil {
		if ready, werr := waitForInstanceReady(service, name, token, readyTimeout); werr != nil || !ready {
			resp.Diagnostics.AddWarning("Parameter store not confirmed running",
				fmt.Sprintf("Parameter store %q did not report running within %s. It may still be starting.", name, readyTimeout))
		}
	}
	if _, err := r.refresh(&state); err != nil {
		resp.Diagnostics.AddWarning("Could not read parameter store after create", err.Error())
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	// Values are written over HTTPS right after this, so wait for the certificate.
	if u := state.URL.ValueString(); u != "" {
		if err := waitForHTTPS(u, readyTimeout); err != nil {
			resp.Diagnostics.AddWarning("Parameter store not reachable yet", err.Error())
		}
	}
}

func (r *ParameterStoreResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ParameterStoreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	exists, err := r.refresh(&state)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read parameter store", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *ParameterStoreResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every configurable attribute requires replacement.
	var plan ParameterStoreResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ParameterStoreResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ParameterStoreResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteParameterStore(r.osaasContext, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete parameter store", err.Error())
	}
}

func (r *ParameterStoreResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	model := ParameterStoreResourceModel{
		Name:         types.StringValue(req.ID),
		RedisURL:     types.StringNull(),
		ConfigAPIKey: types.StringNull(),
	}
	exists, err := r.refresh(&model)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read parameter store", err.Error())
		return
	}
	if !exists {
		resp.Diagnostics.AddError("Parameter store not found", fmt.Sprintf("The workspace has no parameter store %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
