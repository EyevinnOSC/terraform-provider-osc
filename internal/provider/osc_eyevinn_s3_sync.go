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
	_ resource.Resource              = &eyevinns3sync{}
	_ resource.ResourceWithConfigure = &eyevinns3sync{}
)

func Neweyevinns3sync() resource.Resource {
	return &eyevinns3sync{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinns3sync)
}

func (r *eyevinns3sync) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinns3sync is the resource implementation.
type eyevinns3sync struct {
	osaasContext *osaasclient.Context
}

type eyevinns3syncModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Cmdlineargs         types.String       `tfsdk:"cmd_line_args"`
	Sourceaccesskey         types.String       `tfsdk:"source_access_key"`
	Sourcesecretkey         types.String       `tfsdk:"source_secret_key"`
	Sourceregion         types.String       `tfsdk:"source_region"`
	Sourceendpoint         types.String       `tfsdk:"source_endpoint"`
	Sourcesessiontoken         types.String       `tfsdk:"source_session_token"`
	Destaccesskey         types.String       `tfsdk:"dest_access_key"`
	Destsecretkey         types.String       `tfsdk:"dest_secret_key"`
	Destregion         types.String       `tfsdk:"dest_region"`
	Destendpoint         types.String       `tfsdk:"dest_endpoint"`
	Destsessiontoken         types.String       `tfsdk:"dest_session_token"`
}

func (r *eyevinns3sync) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_s3_sync"
}

// Schema defines the schema for the resource.
func (r *eyevinns3sync) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly synchronize files between AWS S3 buckets with S3 Sync by Eyevinn. Simple installation with powerful command-line or environment configurations, this script ensures seamless data management!`,
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
				Description: "Name of s3-sync",
			},
			"cmd_line_args": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"source_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"source_secret_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"source_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"source_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"source_session_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"dest_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"dest_secret_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"dest_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"dest_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"dest_session_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinns3sync) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinns3syncModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-s3-sync")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-s3-sync", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"cmdLineArgs": plan.Cmdlineargs.ValueString(),
		"SourceAccessKey": plan.Sourceaccesskey.ValueString(),
		"SourceSecretKey": plan.Sourcesecretkey.ValueString(),
		"SourceRegion": plan.Sourceregion.ValueString(),
		"SourceEndpoint": plan.Sourceendpoint.ValueString(),
		"SourceSessionToken": plan.Sourcesessiontoken.ValueString(),
		"DestAccessKey": plan.Destaccesskey.ValueString(),
		"DestSecretKey": plan.Destsecretkey.ValueString(),
		"DestRegion": plan.Destregion.ValueString(),
		"DestEndpoint": plan.Destendpoint.ValueString(),
		"DestSessionToken": plan.Destsessiontoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-s3-sync", instance["name"].(string), serviceAccessToken)
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
	state := eyevinns3syncModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-s3-sync"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Cmdlineargs: plan.Cmdlineargs,
		Sourceaccesskey: plan.Sourceaccesskey,
		Sourcesecretkey: plan.Sourcesecretkey,
		Sourceregion: plan.Sourceregion,
		Sourceendpoint: plan.Sourceendpoint,
		Sourcesessiontoken: plan.Sourcesessiontoken,
		Destaccesskey: plan.Destaccesskey,
		Destsecretkey: plan.Destsecretkey,
		Destregion: plan.Destregion,
		Destendpoint: plan.Destendpoint,
		Destsessiontoken: plan.Destsessiontoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinns3sync) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinns3sync) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinns3sync) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinns3syncModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-s3-sync")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-s3-sync", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
