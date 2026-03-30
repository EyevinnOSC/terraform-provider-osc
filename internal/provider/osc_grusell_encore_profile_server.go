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
	_ resource.Resource              = &grusellencoreprofileserver{}
	_ resource.ResourceWithConfigure = &grusellencoreprofileserver{}
)

func Newgrusellencoreprofileserver() resource.Resource {
	return &grusellencoreprofileserver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newgrusellencoreprofileserver)
}

func (r *grusellencoreprofileserver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// grusellencoreprofileserver is the resource implementation.
type grusellencoreprofileserver struct {
	osaasContext *osaasclient.Context
}

type grusellencoreprofileserverModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	S3region         types.String       `tfsdk:"s3_region"`
	S3accesskey         types.String       `tfsdk:"s3_access_key"`
	S3secretkey         types.String       `tfsdk:"s3_secret_key"`
	S3bucket         types.String       `tfsdk:"s3_bucket"`
	S3prefix         types.String       `tfsdk:"s3_prefix"`
	Anthropicapikey         types.String       `tfsdk:"anthropic_api_key"`
	Anthropicmodel         types.String       `tfsdk:"anthropic_model"`
}

func (r *grusellencoreprofileserver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_grusell_encore_profile_server"
}

// Schema defines the schema for the resource.
func (r *grusellencoreprofileserver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your video processing with the Encore Profile Server. Serve dynamic transcoding profiles directly from S3-compatible storage, seamlessly integrating AI capabilities for on-demand profile creation.`,
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
				Description: "Name of encore-profile-server",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "The endpoint URL for the S3-compatible storage service where Encore transcoding profiles are stored",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "The AWS region or region identifier for the S3-compatible storage service",
			},
			"s3_access_key": schema.StringAttribute{
				Optional: true,
				Description: "The access key ID for authenticating with the S3-compatible storage service",
			},
			"s3_secret_key": schema.StringAttribute{
				Optional: true,
				Description: "The secret access key for authenticating with the S3-compatible storage service",
			},
			"s3_bucket": schema.StringAttribute{
				Optional: true,
				Description: "The name of the S3 bucket containing the Encore transcoding profile files (YAML/JSON)",
			},
			"s3_prefix": schema.StringAttribute{
				Optional: true,
				Description: "Optional prefix path within the S3 bucket to limit profile file discovery to a specific directory/folder",
			},
			"anthropic_api_key": schema.StringAttribute{
				Optional: true,
				Description: "API key for accessing Anthropic&#39;s Claude AI service to enable AI-powered profile generation via the /feelinglucky endpoint",
			},
			"anthropic_model": schema.StringAttribute{
				Optional: true,
				Description: "Specifies which Claude AI model to use for generating Encore transcoding profiles",
			},
		},
	}
}

func (r *grusellencoreprofileserver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan grusellencoreprofileserverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("grusell-encore-profile-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "grusell-encore-profile-server", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
		"s3Region": plan.S3region.ValueString(),
		"s3AccessKey": plan.S3accesskey.ValueString(),
		"s3SecretKey": plan.S3secretkey.ValueString(),
		"s3Bucket": plan.S3bucket.ValueString(),
		"s3Prefix": plan.S3prefix.ValueString(),
		"anthropicApiKey": plan.Anthropicapikey.ValueString(),
		"anthropicModel": plan.Anthropicmodel.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "grusell-encore-profile-server", instance["name"].(string), serviceAccessToken)
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
	state := grusellencoreprofileserverModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("grusell-encore-profile-server"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		S3endpoint: plan.S3endpoint,
		S3region: plan.S3region,
		S3accesskey: plan.S3accesskey,
		S3secretkey: plan.S3secretkey,
		S3bucket: plan.S3bucket,
		S3prefix: plan.S3prefix,
		Anthropicapikey: plan.Anthropicapikey,
		Anthropicmodel: plan.Anthropicmodel,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *grusellencoreprofileserver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *grusellencoreprofileserver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *grusellencoreprofileserver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state grusellencoreprofileserverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("grusell-encore-profile-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "grusell-encore-profile-server", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
