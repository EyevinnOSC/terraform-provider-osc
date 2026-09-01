package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource              = &eyevinnopenvideocore{}
	_ resource.ResourceWithConfigure = &eyevinnopenvideocore{}
)

func Neweyevinnopenvideocore() resource.Resource {
	return &eyevinnopenvideocore{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnopenvideocore)
}

func (r *eyevinnopenvideocore) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	osaasContext, ok := req.ProviderData.(*osaasclient.Context)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *OscClient, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.osaasContext = osaasContext
}

// eyevinnopenvideocore is the resource implementation.
type eyevinnopenvideocore struct {
	osaasContext *osaasclient.Context
}

type eyevinnopenvideocoreModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Parameterstoreapikey         types.String       `tfsdk:"parameter_store_api_key"`
	Parameterstore         types.String       `tfsdk:"parameter_store"`
	Miniorootpassword         types.String       `tfsdk:"minio_root_password"`
	Couchdbadminpassword         types.String       `tfsdk:"couchdb_admin_password"`
	Encoremaxinstances         types.String       `tfsdk:"encore_max_instances"`
	Encoremininstances         types.String       `tfsdk:"encore_min_instances"`
	Encoreidletimeoutms         types.String       `tfsdk:"encore_idle_timeout_ms"`
}

func (r *eyevinnopenvideocore) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_open_videocore"
}

// Schema defines the schema for the resource.
func (r *eyevinnopenvideocore) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock the potential of your media assets with Open Videocore, the headless MAM solution that seamlessly scales in the cloud. Effortlessly manage, transcode, package, and deliver content with a single API call!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed: true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed: true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed: true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of open-videocore",
			},
			"osc_access_token": schema.StringAttribute{
				Required: true,
				Description: "Configuration option for oscaccesstoken",
			},
			"parameter_store_api_key": schema.StringAttribute{
				Required: true,
				Description: "API key for authentication",
			},
			"parameter_store": schema.StringAttribute{
				Required: true,
				Description: "Configuration option for parameterstore",
			},
			"minio_root_password": schema.StringAttribute{
				Required: true,
				Description: "Configuration option for miniorootpassword",
			},
			"couchdb_admin_password": schema.StringAttribute{
				Required: true,
				Description: "Database connection configuration",
			},
			"encore_max_instances": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for encoremaxinstances",
			},
			"encore_min_instances": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for encoremininstances",
			},
			"encore_idle_timeout_ms": schema.StringAttribute{
				Optional: true,
				Description: "Timeout value in milliseconds or seconds",
			},
		},
	}
}

func (r *eyevinnopenvideocore) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnopenvideocoreModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-videocore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-open-videocore", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"ParameterStoreApiKey": plan.Parameterstoreapikey.ValueString(),
		"ParameterStore": plan.Parameterstore.ValueString(),
		"MinioRootPassword": plan.Miniorootpassword.ValueString(),
		"CouchdbAdminPassword": plan.Couchdbadminpassword.ValueString(),
		"EncoreMaxInstances": plan.Encoremaxinstances.ValueString(),
		"EncoreMinInstances": plan.Encoremininstances.ValueString(),
		"EncoreIdleTimeoutMs": plan.Encoreidletimeoutms.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-open-videocore", instance["name"].(string), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
		return
	}

	var externalPort = 0
	var externalIp = ""
	if len(ports) > 0 {
		port := ports[0]
		externalPort = port.ExternalPort
		externalIp = port.ExternalIP
	}


	// Update the state with the actual data returned from the API
	state := eyevinnopenvideocoreModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-open-videocore"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Oscaccesstoken: plan.Oscaccesstoken,
		Parameterstoreapikey: plan.Parameterstoreapikey,
		Parameterstore: plan.Parameterstore,
		Miniorootpassword: plan.Miniorootpassword,
		Couchdbadminpassword: plan.Couchdbadminpassword,
		Encoremaxinstances: plan.Encoremaxinstances,
		Encoremininstances: plan.Encoremininstances,
		Encoreidletimeoutms: plan.Encoreidletimeoutms,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnopenvideocore) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnopenvideocore) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnopenvideocore) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnopenvideocoreModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-videocore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-open-videocore", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
