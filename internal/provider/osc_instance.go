package provider

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

const readyTimeout = 5 * time.Minute

// createRetryTimeout is how long creating an instance is retried while the service is
// unavailable.
const createRetryTimeout = 3 * time.Minute

var (
	_ resource.Resource                = &InstanceResource{}
	_ resource.ResourceWithConfigure   = &InstanceResource{}
	_ resource.ResourceWithModifyPlan  = &InstanceResource{}
	_ resource.ResourceWithImportState = &InstanceResource{}
	_ resource.ResourceWithIdentity    = &InstanceResource{}
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
	AsSecrets           types.Bool   `tfsdk:"sensitive_parameters_as_secrets"`
	SecretNames         types.Map    `tfsdk:"secret_names"`
	WaitForReady        types.Bool   `tfsdk:"wait_for_ready"`
	AllowSuspend        types.Bool   `tfsdk:"allow_suspend"`
	Suspended           types.Bool   `tfsdk:"suspended"`
	UseLatest           types.Bool   `tfsdk:"use_latest"`
	URL                 types.String `tfsdk:"url"`
	ExternalIP          types.String `tfsdk:"external_ip"`
	ExternalPort        types.Int64  `tfsdk:"external_port"`
	Instance            types.Map    `tfsdk:"instance"`
}

// InstanceIdentityModel is the resource identity: the pair that uniquely names an
// instance within a workspace. It is what `terraform query` and identity based import use.
type InstanceIdentityModel struct {
	ServiceID types.String `tfsdk:"service_id"`
	Name      types.String `tfsdk:"name"`
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
				Computed:    true,
				Sensitive:   true,
				Description: "Same as `parameters` but hidden from plan output. Use for passwords, tokens and keys. " +
					"Each value is stored in an OSC service secret and the instance is given a `{{secrets.<name>}}` " +
					"reference to it, so the service API never returns the value; see `secret_names`. A value that " +
					"already is a `{{secrets.<name>}}` reference is passed on as it is. " +
					"A parameter must be set in either `parameters` or `sensitive_parameters`, not both. " +
					"When left unset, the values already in state are kept, so an imported instance keeps its " +
					"passwords without them appearing in the configuration; set it to `{}` to remove them.",
				PlanModifiers: []planmodifier.Map{
					keepStateWhenUnset{},
				},
			},
			"sensitive_parameters_as_secrets": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Store the values of `sensitive_parameters` in OSC service secrets. Disable only for a service " +
					"that does not resolve `{{secrets.<name>}}` references; its sensitive values are then stored in the " +
					"instance configuration in plain text, readable by anyone with access to the workspace.",
			},
			"secret_names": schema.MapAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "The OSC service secret holding each sensitive parameter, keyed by parameter name. The provider " +
					"creates the secrets, updates them when a value changes and restarts the instance so it reads the new " +
					"value, and deletes them with the instance. Names are derived from the instance and parameter name. " +
					"OSC never returns secret values, so a secret changed outside Terraform is not detected.",
			},
			"wait_for_ready": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(true),
				Description: "Wait for the instance to report a running health status after create and update, " +
					"up to five minutes. Disable for services that never report health.",
			},
			"allow_suspend": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
				Description: "Accept the instance being suspended, e.g. by an application that suspends it while idle " +
					"and resumes it when needed. A suspended instance then plans no changes and keeps its `url` and other " +
					"attributes, so resources that depend on them are not changed. When false, a suspended instance is " +
					"resumed. Either way a suspended instance is never planned for recreation. Suspending an instance of a " +
					"stateful service such as MinIO, Valkey or PostgreSQL deletes its data.",
			},
			"suspended": schema.BoolAttribute{
				Computed: true,
				Description: "Whether the instance is suspended. A change to its parameters while it is suspended " +
					"resumes it, applies the change, and leaves it running.",
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

func (r *InstanceResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"service_id": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "The OSC service id, e.g. `valkey-io-valkey`.",
			},
			"name": identityschema.StringAttribute{
				RequiredForImport: true,
				Description:       "The instance name.",
			},
		},
	}
}

// setIdentity records the resource identity alongside the state. Terraform versions
// without identity support leave the identity nil, in which case there is nothing to set.
func setIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, model InstanceResourceModel) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, InstanceIdentityModel{ServiceID: model.ServiceID, Name: model.Name})
}

// parametersFromInstance recovers the configurable parameters from an instance document,
// keyed by catalog option name and split into plain and sensitive parameters the way the
// resource models them. Fields that are not catalog options (name, url, status, ...) are
// left out so the result passes plan time validation unchanged.
func parametersFromInstance(service *catalogService, instance map[string]interface{}) (params, sensitive map[string]string) {
	params, sensitive = map[string]string{}, map[string]string{}
	flat := flattenInstance(instance)
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Name == "name" {
			continue
		}
		v, ok := flat[opt.Name]
		if !ok {
			continue
		}
		if opt.Sensitive {
			sensitive[opt.Name] = v
		} else {
			params[opt.Name] = v
		}
	}
	return params, sensitive
}

// modelFromInstance builds a complete resource model for an instance that exists in OSC
// but not yet in state, as needed by import and by `terraform query`.
func (r *InstanceResource) modelFromInstance(service *catalogService, token string, instance map[string]interface{}) (InstanceResourceModel, diag.Diagnostics) {
	name, _ := instance["name"].(string)
	params, sensitive := parametersFromInstance(service, instance)
	model := InstanceResourceModel{
		ServiceID:           types.StringValue(service.ServiceId),
		Name:                types.StringValue(name),
		Parameters:          types.MapNull(types.StringType),
		SensitiveParameters: types.MapNull(types.StringType),
		AsSecrets:           types.BoolValue(true),
		SecretNames:         stringsToMap(nil),
		WaitForReady:        types.BoolValue(true),
		AllowSuspend:        types.BoolValue(false),
		Suspended:           types.BoolValue(false),
		UseLatest:           types.BoolValue(false),
	}
	if len(params) > 0 {
		model.Parameters = stringsToMap(params)
	}
	if len(sensitive) > 0 {
		model.SensitiveParameters = stringsToMap(sensitive)
	}
	diags := r.refreshComputed(service, token, instance, &model)
	return model, diags
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

// keepStateWhenUnset plans the state value of a map attribute when the configuration leaves
// it unset. Terraform never writes sensitive values into generated configuration, so this is
// what lets an imported instance keep its passwords: they are read into state on import and
// stay there until the configuration says otherwise. `{}` in the configuration clears them.
type keepStateWhenUnset struct{}

func (keepStateWhenUnset) Description(_ context.Context) string {
	return "Keeps the value from state when the attribute is not set in the configuration."
}

func (m keepStateWhenUnset) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (keepStateWhenUnset) PlanModifyMap(_ context.Context, req planmodifier.MapRequest, resp *planmodifier.MapResponse) {
	if !req.ConfigValue.IsNull() {
		return
	}
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		resp.PlanValue = types.MapNull(types.StringType)
		return
	}
	resp.PlanValue = req.StateValue
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
	if resp.Diagnostics.HasError() {
		return
	}
	plan.SecretNames = plannedSecretNames(plan)
	resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, path.Root("secret_names"), plan.SecretNames)...)
	resp.Diagnostics.Append(r.planSuspended(ctx, req, resp, plan)...)
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
	if plan.AllowSuspend.ValueBool() && isStateful(service) {
		resp.Diagnostics.AddAttributeWarning(path.Root("allow_suspend"), "Suspending this service deletes its data",
			fmt.Sprintf("Service %q keeps its data on a volume. Suspending an instance of it deletes the volume, and resuming "+
				"it starts with an empty one. Only allow suspension if the data can be lost.", service.ServiceId))
	}
}

// planSuspended plans the suspended attribute. A suspended instance stays suspended only
// when suspension is allowed and nothing that OSC is sent changes; otherwise the apply
// resumes it.
func (r *InstanceResource) planSuspended(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse, plan InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if req.State.Raw.IsNull() {
		return resp.Plan.SetAttribute(ctx, path.Root("suspended"), types.BoolValue(false))
	}
	var state InstanceResourceModel
	diags.Append(req.State.Get(ctx, &state)...)
	if diags.HasError() {
		return diags
	}
	if !state.Suspended.ValueBool() {
		diags.Append(resp.Plan.SetAttribute(ctx, path.Root("suspended"), types.BoolValue(false))...)
		return diags
	}
	changes := instanceChanges(plan, state)
	stay := plan.AllowSuspend.ValueBool() && !changes
	diags.Append(resp.Plan.SetAttribute(ctx, path.Root("suspended"), types.BoolValue(stay))...)
	if stay {
		return diags
	}
	// A resumed instance may be given another external address.
	diags.Append(resp.Plan.SetAttribute(ctx, path.Root("external_ip"), types.StringUnknown())...)
	diags.Append(resp.Plan.SetAttribute(ctx, path.Root("external_port"), types.Int64Unknown())...)
	if changes && plan.AllowSuspend.ValueBool() {
		diags.AddWarning("Suspended instance will be resumed",
			fmt.Sprintf("Instance %q is suspended and its parameters change. OSC cannot change a suspended instance, so "+
				"the apply resumes it, applies the change and leaves it running.", plan.Name.ValueString()))
	}
	return diags
}

// instanceChanges reports whether the plan changes what OSC is sent for the instance.
func instanceChanges(plan, state InstanceResourceModel) bool {
	return !plan.Parameters.Equal(state.Parameters) || !plan.SensitiveParameters.Equal(state.SensitiveParameters) ||
		!plan.SecretNames.Equal(state.SecretNames) || !plan.AsSecrets.Equal(state.AsSecrets)
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

	if suspended, err := findSuspended(r.osaasContext, service.ServiceId, plan.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to check for a suspended instance", err.Error())
		return
	} else if suspended != nil {
		resp.Diagnostics.AddError("Instance is suspended",
			fmt.Sprintf("Service %q already has a suspended instance named %q. Import it to manage it with Terraform:\n"+
				"  terraform import <address of this osc_instance resource> %s/%s\n"+
				"or discard the suspended instance in OSC if it is no longer needed.",
				service.ServiceId, plan.Name.ValueString(), service.ServiceId, plan.Name.ValueString()))
		return
	}

	sensitive, secretNames := mapToStrings(plan.SensitiveParameters), mapToStrings(plan.SecretNames)
	if err := putSecrets(r.osaasContext, service.ServiceId, secretValues(sensitive, secretNames)); err != nil {
		resp.Diagnostics.AddError("Failed to store sensitive parameters as secrets",
			fmt.Sprintf("Could not write the secrets of instance %q to service %q: %s", plan.Name.ValueString(), service.ServiceId, err.Error()))
		return
	}

	body := buildInstanceBody(service, plan.Name.ValueString(), mapToStrings(plan.Parameters), withSecretRefs(sensitive, secretNames))
	instance, err := createInstanceRetrying(service, token, body, plan.UseLatest.ValueBool(), createRetryTimeout, 10*time.Second)
	if err != nil {
		if isRejection(err) {
			// Nothing refers to the secrets. When the service could not be reached they are
			// kept, since the instance may have been created after all.
			resp.Diagnostics.Append(r.deleteSecrets(service.ServiceId, secretNames, nil)...)
			resp.Diagnostics.AddError("Failed to create instance",
				fmt.Sprintf("Service %q rejected creation of instance %q: %s\n\n%s",
					service.ServiceId, plan.Name.ValueString(), err.Error(), describeOptions(service)))
		} else {
			resp.Diagnostics.AddError("Failed to create instance",
				fmt.Sprintf("Could not reach service %q to create instance %q, retried for %s: %s. "+
					"This is not a problem with the parameters; apply again once the service answers.",
					service.ServiceId, plan.Name.ValueString(), createRetryTimeout, err.Error()))
		}
		return
	}

	resp.Diagnostics.Append(r.waitReady(service, token, plan)...)

	// Re-read so the state reflects what the service stored.
	if current, err := getInstance(service, plan.Name.ValueString(), token); err == nil && current != nil {
		instance = current
	}

	state := plan
	state.Suspended = types.BoolValue(false)
	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, state)...)
}

func (r *InstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state InstanceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	type current struct {
		service   *catalogService
		token     string
		instance  map[string]interface{}
		suspended *suspendedInstance
	}
	cur, exists, err := readConfirmed(func() (current, bool, error) {
		service, err := findSubscribedService(r.osaasContext, state.ServiceID.ValueString())
		if err != nil {
			return current{}, false, fmt.Errorf("could not read the OSC catalog: %w", err)
		}
		if service == nil {
			// OSC drops the subscription while every instance of the service is
			// suspended, so no subscription means no running instance, not none at all.
			suspended, err := findSuspended(r.osaasContext, state.ServiceID.ValueString(), state.Name.ValueString())
			if err != nil {
				return current{}, false, fmt.Errorf("could not read the suspended instances: %w", err)
			}
			return current{suspended: suspended}, suspended != nil, nil
		}
		token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
		if err != nil {
			return current{}, false, fmt.Errorf("could not get a service access token: %w", err)
		}
		instance, err := getInstance(service, state.Name.ValueString(), token)
		if err != nil || instance != nil {
			return current{service, token, instance, nil}, err == nil, err
		}
		// A suspended instance is gone from the service API but not deleted.
		suspended, err := findSuspended(r.osaasContext, service.ServiceId, state.Name.ValueString())
		if err != nil {
			return current{}, false, fmt.Errorf("could not read the suspended instances: %w", err)
		}
		return current{service, token, nil, suspended}, suspended != nil, nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	if cur.suspended != nil {
		// Keep the attributes from when it last ran, so dependents do not change.
		state.Suspended = types.BoolValue(true)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, state)...)
		return
	}
	state.Suspended = types.BoolValue(false)
	service, token, instance := cur.service, cur.token, cur.instance

	// A parameter no longer referring to its secret is dropped from secret_names, so the
	// plan shows an update that points it at the secret again.
	if drifted := secretDrift(mapToStrings(state.SecretNames), instance); len(drifted) > 0 {
		names := mapToStrings(state.SecretNames)
		for _, k := range drifted {
			delete(names, k)
		}
		state.SecretNames = stringsToMap(names)
	}

	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, state)...)
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

	var resumedAt time.Time
	if state.Suspended.ValueBool() {
		if plan.Suspended.ValueBool() {
			// Nothing OSC is sent changes, so the instance stays suspended with the
			// attributes it had when it last ran.
			newState := plan
			newState.ID, newState.URL, newState.Instance = state.ID, state.URL, state.Instance
			newState.ExternalIP, newState.ExternalPort = state.ExternalIP, state.ExternalPort
			resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
			resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, newState)...)
			return
		}
		if err := resumeSuspended(r.osaasContext, service.ServiceId, plan.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to resume instance",
				fmt.Sprintf("Could not resume suspended instance %q of service %q: %s", plan.Name.ValueString(), service.ServiceId, err.Error()))
			return
		}
		resumedAt = time.Now()
		if _, err := waitForResumed(service, plan.Name.ValueString(), token, readyTimeout); err != nil {
			resp.Diagnostics.AddError("Resumed instance did not come back",
				fmt.Sprintf("Instance %q of service %q was resumed but its API did not return it within %s: %v",
					plan.Name.ValueString(), service.ServiceId, readyTimeout, err))
			return
		}
		resp.Diagnostics.Append(r.waitReady(service, token, plan)...)
	}
	resumed := !resumedAt.IsZero()

	oldSensitive, oldNames := mapToStrings(state.SensitiveParameters), mapToStrings(state.SecretNames)
	newSensitive, newNames := mapToStrings(plan.SensitiveParameters), mapToStrings(plan.SecretNames)

	// Secrets are written before the instance refers to them.
	changedSecrets := map[string]string{}
	for k, name := range newNames {
		if oldNames[k] != name || oldSensitive[k] != newSensitive[k] {
			changedSecrets[name] = newSensitive[k]
		}
	}
	if err := putSecrets(r.osaasContext, service.ServiceId, changedSecrets); err != nil {
		resp.Diagnostics.AddError("Failed to store sensitive parameters as secrets",
			fmt.Sprintf("Could not write the secrets of instance %q to service %q: %s", plan.Name.ValueString(), service.ServiceId, err.Error()))
		return
	}

	// Compare what OSC is sent rather than the attributes: a new secret value leaves the
	// reference unchanged, and moving a plain text value into a secret changes it.
	oldBody := buildInstanceBody(service, plan.Name.ValueString(), mapToStrings(state.Parameters), withSecretRefs(oldSensitive, oldNames))
	body := buildInstanceBody(service, plan.Name.ValueString(), mapToStrings(plan.Parameters), withSecretRefs(newSensitive, newNames))

	var instance map[string]interface{}
	switch {
	case !reflect.DeepEqual(oldBody, body):
		delete(body, "name")
		if resumed {
			instance, err = patchAfterResume(service, plan.Name.ValueString(), token, body, resumedAt)
		} else {
			instance, err = patchInstance(service, plan.Name.ValueString(), token, body)
		}
		if err != nil {
			var ae *apiError
			hint := ""
			if isNotFound(err) || (asAPIError(err, &ae) && (ae.StatusCode == http.StatusMethodNotAllowed || ae.StatusCode == http.StatusNotImplemented)) {
				hint = fmt.Sprintf("\n\nService %q does not support in-place updates. Recreate the instance with:\n"+
					"  terraform apply -replace=\"<address of this osc_instance resource>\"", service.ServiceId)
				if movesToSecrets(oldNames, newNames) {
					hint = fmt.Sprintf("\n\nService %q does not support in-place updates, which moving the sensitive parameters "+
						"of an existing instance into secrets needs. Replacing the instance deletes it and its data; to keep it, "+
						"set sensitive_parameters_as_secrets = false.", service.ServiceId)
				}
			}
			resp.Diagnostics.AddError("Failed to update instance",
				fmt.Sprintf("Service %q rejected the update of instance %q: %s%s\n\n%s",
					service.ServiceId, plan.Name.ValueString(), err.Error(), hint, describeOptions(service)))
			return
		}
		resp.Diagnostics.Append(r.waitReady(service, token, plan)...)
	case len(changedSecrets) > 0 && !resumed:
		// A resumed instance has just started and read its secrets.
		resp.Diagnostics.Append(r.restartForSecrets(service, token, plan)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Secrets no parameter refers to any more.
	resp.Diagnostics.Append(r.deleteSecrets(service.ServiceId, oldNames, newNames)...)

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
	newState.Suspended = types.BoolValue(false)
	resp.Diagnostics.Append(r.refreshComputed(service, token, instance, &newState)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, newState)...)
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
		// Without a subscription nothing runs, but a suspended record may be left: OSC
		// drops the subscription while every instance of the service is suspended.
		if err := discardSuspended(r.osaasContext, state.ServiceID.ValueString(), state.Name.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to discard suspended instance", err.Error())
			return
		}
		resp.Diagnostics.Append(r.deleteSecrets(state.ServiceID.ValueString(), mapToStrings(state.SecretNames), nil)...)
		return
	}

	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	if err := removeInstance(service, state.Name.ValueString(), token); err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
	// Also when state says it runs: it may have been suspended since the last refresh, and
	// a suspended record left behind would block creating an instance with the name.
	if err := discardSuspended(r.osaasContext, service.ServiceId, state.Name.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to discard suspended instance", err.Error())
		return
	}
	resp.Diagnostics.Append(r.deleteSecrets(service.ServiceId, mapToStrings(state.SecretNames), nil)...)
}

// restartForSecrets restarts the instance after only secret values changed, since the
// instance configuration it is sent is the same and an update would not reach it.
func (r *InstanceResource) restartForSecrets(service *catalogService, token string, plan InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	instance, err := getInstance(service, plan.Name.ValueString(), token)
	if err != nil || instance == nil {
		diags.AddError("Failed to restart instance",
			fmt.Sprintf("The secrets of instance %q were updated, but the instance could not be read to restart it: %v", plan.Name.ValueString(), err))
		return diags
	}
	restarted, err := restartInstance(service, token, instance)
	switch {
	case err != nil:
		diags.AddError("Failed to restart instance",
			fmt.Sprintf("The secrets of instance %q were updated, but restarting it failed: %s. It uses the new values "+
				"after its next restart.", plan.Name.ValueString(), err.Error()))
	case !restarted:
		diags.AddWarning("Instance not restarted",
			fmt.Sprintf("The secrets of instance %q were updated, but service %q offers no restart. The instance uses "+
				"the new values after its next restart.", plan.Name.ValueString(), service.ServiceId))
	default:
		diags.Append(r.waitReady(service, token, plan)...)
	}
	return diags
}

// deleteSecrets removes the secrets in names that are not also in keep. A secret that
// cannot be deleted is left behind with a warning rather than failing the apply.
func (r *InstanceResource) deleteSecrets(serviceID string, names, keep map[string]string) diag.Diagnostics {
	var diags diag.Diagnostics
	kept := map[string]bool{}
	for _, name := range keep {
		kept[name] = true
	}
	for _, name := range names {
		if kept[name] {
			continue
		}
		if err := deleteSecret(r.osaasContext, serviceID, name); err != nil {
			diags.AddWarning("Could not delete secret",
				fmt.Sprintf("Secret %q of service %q is no longer used but could not be deleted: %s. Remove it with "+
					"`osc secrets rm %s %s`.", name, serviceID, err.Error(), serviceID, name))
		}
	}
	return diags
}

// ImportState accepts the id "service_id/name", or the identity {service_id, name} from
// an import block. The instance is read from OSC so that its parameters land in state and
// `terraform plan -generate-config-out` writes a complete resource block.
func (r *InstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var serviceID, name string
	switch {
	case req.ID != "":
		parts := strings.SplitN(req.ID, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import id",
				fmt.Sprintf("Expected \"service_id/name\", e.g. \"valkey-io-valkey/mycache\", got %q.", req.ID))
			return
		}
		serviceID, name = parts[0], parts[1]
	case req.Identity != nil:
		var identity InstanceIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		serviceID, name = identity.ServiceID.ValueString(), identity.Name.ValueString()
	}
	if serviceID == "" || name == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Both service_id and name are required.")
		return
	}

	service, err := findSubscribedService(r.osaasContext, serviceID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read OSC catalog", err.Error())
		return
	}
	if service == nil {
		// OSC drops the subscription while every instance of the service is suspended.
		suspended, err := findSuspended(r.osaasContext, serviceID, name)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read the suspended instances", err.Error())
			return
		}
		if suspended == nil {
			resp.Diagnostics.AddError("Instance not found",
				fmt.Sprintf("The workspace is not subscribed to service %q, so it has no instances of it. Check the service id; "+
					"ids look like {contributor}-{name} and cannot be guessed.", serviceID))
			return
		}
		// The service's options, to tell sensitive parameters from plain ones, come from
		// the catalog mirror while there is no subscription to read them from.
		mirrored, _, err := mirrorService(ctx, serviceID)
		if err != nil || mirrored == nil {
			detail := "the catalog mirror does not list it"
			if err != nil {
				detail = err.Error()
			}
			resp.Diagnostics.AddError("Failed to read OSC catalog",
				fmt.Sprintf("Instance %q of service %q is suspended and the workspace has no subscription to the service, "+
					"so its options must come from the catalog mirror: %s.", name, serviceID, detail))
			return
		}
		model := modelFromSuspended(mirrored, name, suspended)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
		resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, model)...)
		return
	}
	token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}
	instance, err := getInstance(service, name, token)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read instance", err.Error())
		return
	}
	if instance == nil {
		suspended, err := findSuspended(r.osaasContext, service.ServiceId, name)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read the suspended instances", err.Error())
			return
		}
		if suspended == nil {
			resp.Diagnostics.AddError("Instance not found",
				fmt.Sprintf("Service %q has no instance named %q in this workspace.", service.ServiceId, name))
			return
		}
		model := modelFromSuspended(service, name, suspended)
		resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
		resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, model)...)
		return
	}
	if _, ok := instance["name"]; !ok {
		instance["name"] = name
	}

	model, diags := r.modelFromInstance(service, token, instance)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	resp.Diagnostics.Append(setIdentity(ctx, resp.Identity, model)...)
}
