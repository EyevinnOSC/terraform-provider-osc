package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &ParameterResource{}
	_ resource.ResourceWithConfigure      = &ParameterResource{}
	_ resource.ResourceWithImportState    = &ParameterResource{}
	_ resource.ResourceWithValidateConfig = &ParameterResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewParameterResource)
}

func NewParameterResource() resource.Resource {
	return &ParameterResource{}
}

// ParameterResource manages one value in a parameter store.
type ParameterResource struct {
	osaasContext *osaasclient.Context
}

type ParameterResourceModel struct {
	ID             types.String `tfsdk:"id"`
	ParameterStore types.String `tfsdk:"parameter_store"`
	Key            types.String `tfsdk:"key"`
	Value          types.String `tfsdk:"value"`
	SecretValue    types.String `tfsdk:"secret_value"`
}

func (r *ParameterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter"
}

func (r *ParameterResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages one value in a parameter store. Every value in the store a My App is bound to becomes " +
			"an environment variable of the app, named by `key`.\n\n" +
			"Set exactly one of `value` and `secret_value`. A secret value is encrypted at rest, masked everywhere but in " +
			"the bound app, and hidden from plan output; it needs a store with `enable_secrets`. A value that has been " +
			"stored as a secret cannot be turned back into a plain value in place.\n\n" +
			"Apps read their environment when they start, so a changed value reaches a running app on its next restart. " +
			"To restart it from Terraform, include the values in the app's `rebuild_trigger`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier in the form `parameter_store/key`. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"parameter_store": schema.StringAttribute{
				Required:      true,
				Description:   "Name of the parameter store, e.g. `osc_parameter_store.<name>.name`. Changing it replaces the value.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"key": schema.StringAttribute{
				Required:      true,
				Description:   "The key, which is also the environment variable name in a bound app, e.g. `DATABASE_URL`. Changing it replaces the value.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"value": schema.StringAttribute{
				Optional:    true,
				Description: "A plain value, readable by anyone with access to the store.",
			},
			"secret_value": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "A secret value, encrypted at rest.",
				PlanModifiers: []planmodifier.String{
					// The store refuses to turn a secret into a plain value.
					stringplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.StringRequest, resp *stringplanmodifier.RequiresReplaceIfFuncResponse) {
						resp.RequiresReplace = !req.StateValue.IsNull() && req.PlanValue.IsNull()
					}, "Replaces the value when it stops being a secret.", "Replaces the value when it stops being a secret."),
				},
			},
		},
	}
}

func (r *ParameterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *ParameterResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config ParameterResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || config.Value.IsUnknown() || config.SecretValue.IsUnknown() {
		return
	}
	if config.Value.IsNull() == config.SecretValue.IsNull() {
		resp.Diagnostics.AddAttributeError(path.Root("value"), "Exactly one value required",
			"Set exactly one of value and secret_value. After importing a secret, set secret_value in the configuration: "+
				"Terraform never writes sensitive values into generated configuration.")
	}
}

func (r *ParameterResource) write(plan ParameterResourceModel) error {
	client, err := newParameterClient(r.osaasContext, plan.ParameterStore.ValueString())
	if err != nil {
		return err
	}
	secret := !plan.SecretValue.IsNull()
	value := plan.Value.ValueString()
	if secret {
		value = plan.SecretValue.ValueString()
	}
	return client.put(plan.Key.ValueString(), value, secret)
}

func (r *ParameterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ParameterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(plan); err != nil {
		resp.Diagnostics.AddError("Failed to set parameter",
			fmt.Sprintf("Could not set %q in parameter store %q: %s", plan.Key.ValueString(), plan.ParameterStore.ValueString(), err.Error()))
		return
	}
	plan.ID = types.StringValue(plan.ParameterStore.ValueString() + "/" + plan.Key.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ParameterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ParameterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := newParameterClient(r.osaasContext, state.ParameterStore.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read parameter store", err.Error())
		return
	}
	obj, err := client.get(state.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read parameter", err.Error())
		return
	}
	if obj == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyParameter(obj, client.apiKey != "", &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// applyParameter records the stored value. A secret read without the store's API key comes
// back masked, so the value in state is kept rather than replaced by the mask.
func applyParameter(obj *parameterObject, plaintext bool, model *ParameterResourceModel) {
	if obj.Secret {
		model.Value = types.StringNull()
		if plaintext {
			model.SecretValue = types.StringValue(obj.Value)
		} else if model.SecretValue.IsNull() {
			model.SecretValue = types.StringValue("")
		}
		return
	}
	model.SecretValue = types.StringNull()
	model.Value = types.StringValue(obj.Value)
}

func (r *ParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state ParameterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(plan); err != nil {
		resp.Diagnostics.AddError("Failed to update parameter",
			fmt.Sprintf("Could not update %q in parameter store %q: %s", plan.Key.ValueString(), plan.ParameterStore.ValueString(), err.Error()))
		return
	}
	plan.ID = state.ID
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ParameterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ParameterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := newParameterClient(r.osaasContext, state.ParameterStore.ValueString())
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			return
		}
		resp.Diagnostics.AddError("Failed to read parameter store", err.Error())
		return
	}
	if err := client.delete(state.Key.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete parameter", err.Error())
	}
}

func (r *ParameterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("Expected \"parameter_store/key\", e.g. \"mystore/DATABASE_URL\", got %q.", req.ID))
		return
	}
	client, err := newParameterClient(r.osaasContext, parts[0])
	if err != nil {
		resp.Diagnostics.AddError("Failed to read parameter store", err.Error())
		return
	}
	obj, err := client.get(parts[1])
	if err != nil {
		resp.Diagnostics.AddError("Failed to read parameter", err.Error())
		return
	}
	if obj == nil {
		resp.Diagnostics.AddError("Parameter not found", fmt.Sprintf("Parameter store %q has no key %q.", parts[0], parts[1]))
		return
	}
	model := ParameterResourceModel{
		ID:             types.StringValue(req.ID),
		ParameterStore: types.StringValue(parts[0]),
		Key:            types.StringValue(parts[1]),
		Value:          types.StringNull(),
		SecretValue:    types.StringNull(),
	}
	applyParameter(obj, client.apiKey != "", &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
