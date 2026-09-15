package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

const readyTimeout = 5 * time.Minute

var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithConfigure   = &InstanceResource{}
	_ resource.ResourceWithModifyPlan  = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewInstanceResource)
}

func NewInstanceResource() resource.Resource {
	return &InstanceResource{}
}

// InstanceResource manages an instance of any service in the OSC catalog. The service's
// parameter schema is read from the catalog at plan and apply time, so new services work
// without changes to the provider.
type InstanceResource struct {
	osaasContext *osaasclient.Context
}

type InstanceResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ServiceID           types.String `tfsdk:"service_id"`
	Name                types.String `tfsdk:"name"`
	Parameters          types.Map    `tfsdk:"parameters"`
	SensitiveParameters types.Map    `tfsdk:"sensitive_parameters"`
	WaitForReady        types.Bool   `tfsdk:"wait_for_ready"`
	UseLatest           types.Bool   `tfsdk:"use_latest"`
	URL                 types.String `tfsdk:"url"`
	ExternalIP          types.String `tfsdk:"external_ip"`
	ExternalPort        types.Int64  `tfsdk:"external_port"`
	Instance            types.Map    `tfsdk:"instance"`
}

func (r *InstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *InstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an instance of any service in the Open Source Cloud catalog.\n\n" +
			"The service's parameter schema is fetched from the OSC catalog when planning and applying, " +
			"so this single resource covers every current and future service without a provider update. " +
			"Parameter names, required parameters, enum values and value patterns are validated against the " +
			"catalog at plan time whenever the tenant is already subscribed to the service. Use the `osc_service` " +
			"data source to inspect a service's parameters from Terraform.\n\n" +
			"Changing `service_id`, `name` or `use_latest` replaces the instance. Changing parameters updates the instance " +
			"in place with a rolling restart when the service supports it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Identifier in the form `service_id/name`. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service_id": schema.StringAttribute{
				Required: true,
				Description: "The OSC service id, e.g. `valkey-io-valkey` or `eyevinn-encore-callback-listener`. " +
					"Ids have the form {contributor}-{name} and cannot be guessed; look them up in the OSC catalog. " +
					"The tenant is subscribed to the service automatically on first use.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Instance name. Must be unique per service within the tenant and usually matches `^\\w+$`.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"parameters": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Service specific parameters as a map of strings, keyed by the parameter name from the service's " +
					"schema (case sensitive, e.g. `RedisUrl`). Boolean parameters take \"true\" or \"false\". " +
					"Do not include `name` here.",
			},
			"sensitive_parameters": schema.MapAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Sensitive:   true,
				Description: "Same as `parameters` but hidden from plan output. Use for passwords, tokens and keys. " +
					"A parameter must be set in either `parameters` or `sensitive_parameters`, not both.",
			},
			"wait_for_ready": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Wait for the instance to report a running health status after create and update, " +
					"up to five minutes. Disable for services that never report health.",
			},
			"use_latest": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Provision the instance from the latest built image of the service instead of the pinned " +
					"stable catalog release. Meant for testing pre-release builds, not for production. Changing it " +
					"replaces the instance, since OSC decides the image at creation time.",
				PlanModifiers: []planmodifier.Bool{
					// State written before this attribute existed has it null. Treat that as
					// "unknown image" and do not replace just to record the default.
					boolplanmodifier.RequiresReplaceIf(func(_ context.Context, req planmodifier.BoolRequest, resp *boolplanmodifier.RequiresReplaceIfFuncResponse) {
						resp.RequiresReplace = !req.StateValue.IsNull()
					}, "Replaces the instance when use_latest changes.", "Replaces the instance when `use_latest` changes."),
				},
			},
			"url": schema.StringAttribute{
				Computed:    true,
				Description: "URL of the created instance.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_ip": schema.StringAttribute{
				Computed:    true,
				Description: "External IP of the instance for services exposing a TCP/UDP port. Empty otherwise.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_port": schema.Int64Attribute{
				Computed:    true,
				Description: "External port of the instance for services exposing a TCP/UDP port. Zero otherwise.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"instance": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "The full instance document as returned by the service API, as a map of strings. " +
					"Nested values are JSON encoded. Values of parameters set in `sensitive_parameters`, or marked " +
					"sensitive by the service, are replaced with \"(sensitive)\".",
			},
		},
	}
}

func (r *InstanceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	osaasContext, ok := req.ProviderData.(*osaasclient.Context)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *osaasclient.Context, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	r.osaasContext = osaasContext
}

// mapToParamValues extracts known and unknown values from a Terraform map attribute.
func mapToParamValues(m types.Map) paramValues {
	pv := newParamValues()
	if m.IsNull() || m.IsUnknown() {
		return pv
	}
	for k, v := range m.Elements() {
		s, ok := v.(types.String)
		if !ok || s.IsUnknown() {
			pv.unknown[k] = true
			continue
		}
		if s.IsNull() {
			continue
		}
		pv.values[k] = s.ValueString()
	}
	return pv
}

func mapToStrings(m types.Map) map[string]string {
	out := map[string]string{}
	if m.IsNull() || m.IsUnknown() {
		return out
	}
	for k, v := range m.Elements() {
		if s, ok := v.(types.String); ok && !s.IsNull() && !s.IsUnknown() {
			out[k] = s.ValueString()
		}
	}
	return out
}

func stringsToMap(m map[string]string) types.Map {
	elems := make(map[string]attr.Value, len(m))
	for k, v := range m {
		elems[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, elems)
}

// ModifyPlan validates the configuration against the service's catalog schema so that
// mistakes surface at plan time with the accepted schema in the error message.
func (r *InstanceResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() || r.osaasContext == nil {
		return
	}
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() || plan.ServiceID.IsUnknown() {
		return
	}

	service, err := findSubscribedService(r.osaasContext, plan.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OSC catalog", err.Error())
		return
	}
	if service == nil {
		// Not subscribed yet: validate against the published catalog mirror instead.
		mirrored, suggestions, merr := mirrorService(ctx, plan.ServiceID.ValueString())
		switch {
		case merr != nil:
			resp.Diagnostics.AddAttributeWarning(path.Root("service_id"), "Service not yet subscribed",
				fmt.Sprintf("The workspace is not subscribed to service %q and the catalog mirror could not be read (%s), "+
					"so parameters cannot be validated at plan time. The subscription is created and parameters are "+
					"validated when applying.", plan.ServiceID.ValueString(), merr.Error()))
			return
		case mirrored == nil:
			resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Unknown service",
				unknownServiceDetail(plan.ServiceID.ValueString(), suggestions, nil)+
					" If the service was published very recently the mirror may not list it yet; in that case apply will subscribe and validate.")
			return
		}
		resp.Diagnostics.AddAttributeWarning(path.Root("service_id"), "Service not yet subscribed",
			fmt.Sprintf("The workspace is not subscribed to service %q. Apply subscribes it first. "+
				"Parameters were validated against the catalog mirror generated %s.",
				plan.ServiceID.ValueString(), mirror.GeneratedAt.Format("2006-01-02")))
		service = mirrored
	}

	resp.Diagnostics.Append(r.validateAgainstService(service, plan)...)
}

func (r *InstanceResource) validateAgainstService(service *catalogService, plan InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if service.ServiceType != "" && service.ServiceType != "instance" {
		diags.AddAttributeError(path.Root("service_id"), "Unsupported service type",
			fmt.Sprintf("Service %q is of type %q. osc_instance only manages long running services of type \"instance\".",
				service.ServiceId, service.ServiceType))
		return diags
	}
	if !plan.Name.IsUnknown() && !plan.Name.IsNull() {
		diags.Append(validateInstanceName(service, plan.Name.ValueString())...)
	}
	if plan.Parameters.IsUnknown() || plan.SensitiveParameters.IsUnknown() {
		return diags
	}
	diags.Append(validateParameters(service, mapToParamValues(plan.Parameters), mapToParamValues(plan.SensitiveParameters))...)
	return diags
}

// refreshComputed fills the computed attributes from the live instance and its ports.
func (r *InstanceResource) refreshComputed(service *catalogService, token string, instance map[string]interface{}, model *InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	model.ID = types.StringValue(service.ServiceId + "/" + model.Name.ValueString())
	if u, ok := instance["url"].(string); ok {
		model.URL = types.StringValue(u)
	} else {
		model.URL = types.StringValue("")
	}
	// Services echo their configuration back, so hide anything the user marked sensitive.
	flat := flattenInstance(instance)
	for k := range mapToStrings(model.SensitiveParameters) {
		if _, ok := flat[k]; ok {
			flat[k] = "(sensitive)"
		}
	}
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Sensitive {
			if _, ok := flat[opt.Name]; ok {
				flat[opt.Name] = "(sensitive)"
			}
		}
	}
	model.Instance = stringsToMap(flat)

	externalIP, externalPort := "", int64(0)
	ports, err := getPorts(service, model.Name.ValueString(), token)
	if err != nil && !isNotFound(err) {
		diags.AddWarning("Could not read instance ports", err.Error())
	}
	if len(ports) > 0 {
		externalIP = ports[0].ExternalIP
		externalPort = int64(ports[0].ExternalPort)
	}
	model.ExternalIP = types.StringValue(externalIP)
	model.ExternalPort = types.Int64Value(externalPort)
	return diags
}

func (r *InstanceResource) waitReady(service *catalogService, token string, model InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if !model.WaitForReady.ValueBool() {
		return diags
	}
	ready, err := waitForInstanceReady(service, model.Name.ValueString(), token, readyTimeout)
	if err != nil {
		diags.AddWarning("Could not check instance health", err.Error())
		return diags
	}
	if !ready {
		diags.AddWarning("Instance not confirmed running",
			fmt.Sprintf("Instance %q of service %q did not report a running health status within %s. "+
				"It may still be starting, or the service may not implement health checks. "+
				"Set wait_for_ready = false to skip this check.", model.Name.ValueString(), service.ServiceId, readyTimeout))
	}
	return diags
}

func (r *InstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := ensureSubscribed(r.osaasContext, plan.ServiceID.ValueString())
	if err != nil {
		_, suggestions, merr := mirrorService(ctx, plan.ServiceID.ValueString())
		resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Unknown service",
			err.Error()+"\n\n"+unknownServiceDetail(plan.ServiceID.ValueString(), suggestions, merr))
		return
	}

	// Values are all known now, so this catches anything plan time could not.
	resp.Diagnostics.Append(r.validateAgainstService(service, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	body := buildInstanceBody(service, plan.Name.ValueString(), mapToStrings(plan.Parameters), mapToStrings(plan.SensitiveParameters))
	instance, err := createInstance(service, token, body, plan.UseLatest.ValueBool())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance",
			fmt.Sprintf("Service %q rejected creation of instance %q: %s\n\n%s",
				service.ServiceId, plan.Name.ValueString(), err.Error(), describeOptions(service)))
		return
	}

	resp.Diagnostics.Append(r.waitReady(service, token, plan)...)

	// Re-read so the state reflects what the service stored.
	if current, err := getInstance(service, plan.Name.ValueString(), token); err == nil && current != nil {
		instance = current
	}

	state := plan
	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := findSubscribedService(r.osaasContext, state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OSC catalog", err.Error())
		return
	}
	if service == nil {
		// No subscription means no instances of the service can exist for this tenant.
		resp.State.RemoveResource(ctx)
		return
	}

	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := getInstance(service, state.Name.ValueString(), token)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	if instance == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *InstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state InstanceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := ensureSubscribed(r.osaasContext, plan.ServiceID.ValueString())
	if err != nil {
		_, suggestions, merr := mirrorService(ctx, plan.ServiceID.ValueString())
		resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Unknown service",
			err.Error()+"\n\n"+unknownServiceDetail(plan.ServiceID.ValueString(), suggestions, merr))
		return
	}
	resp.Diagnostics.Append(r.validateAgainstService(service, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	newParams := mapToStrings(plan.Parameters)
	newSensitive := mapToStrings(plan.SensitiveParameters)
	changed := !plan.Parameters.Equal(state.Parameters) || !plan.SensitiveParameters.Equal(state.SensitiveParameters)

	var instance map[string]interface{}
	if changed {
		body := buildInstanceBody(service, plan.Name.ValueString(), newParams, newSensitive)
		delete(body, "name")
		instance, err = patchInstance(service, plan.Name.ValueString(), token, body)
		if err != nil {
			var ae *apiError
			hint := ""
			if isNotFound(err) || (asAPIError(err, &ae) && (ae.StatusCode == http.StatusMethodNotAllowed || ae.StatusCode == http.StatusNotImplemented)) {
				hint = fmt.Sprintf("\n\nService %q does not support in-place updates. Recreate the instance with:\n"+
					"  terraform apply -replace=\"<address of this osc_instance resource>\"", service.ServiceId)
			}
			resp.Diagnostics.AddError("Failed to update instance",
				fmt.Sprintf("Service %q rejected the update of instance %q: %s%s\n\n%s",
					service.ServiceId, plan.Name.ValueString(), err.Error(), hint, describeOptions(service)))
			return
		}
		resp.Diagnostics.Append(r.waitReady(service, token, plan)...)
	}

	current, err := getInstance(service, plan.Name.ValueString(), token)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance after update", err.Error())
		return
	}
	if current != nil {
		instance = current
	}
	if instance == nil {
		resp.Diagnostics.AddError("Instance disappeared", fmt.Sprintf("Instance %q of service %q no longer exists.", plan.Name.ValueString(), service.ServiceId))
		return
	}

	newState := plan
	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &newState)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *InstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	service, err := findSubscribedService(r.osaasContext, state.ServiceID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OSC catalog", err.Error())
		return
	}
	if service == nil {
		return
	}

	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	if err := removeInstance(service, state.Name.ValueString(), token); err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
	}
}

// ImportState accepts "service_id/name".
func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("Expected \"service_id/name\", e.g. \"valkey-io-valkey/mycache\", got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("service_id"), parts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("wait_for_ready"), true)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("use_latest"), false)...)
}
