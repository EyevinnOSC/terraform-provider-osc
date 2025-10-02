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
	_ resource.Resource              = &birmevideouploader{}
	_ resource.ResourceWithConfigure = &birmevideouploader{}
)

func Newbirmevideouploader() resource.Resource {
	return &birmevideouploader{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmevideouploader)
}

func (r *birmevideouploader) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmevideouploader is the resource implementation.
type birmevideouploader struct {
	osaasContext *osaasclient.Context
}

type birmevideouploaderModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	S3accesskey         types.String       `tfsdk:"s3_access_key"`
	S3secretkey         types.String       `tfsdk:"s3_secret_key"`
	S3awsregion         types.String       `tfsdk:"s3_aws_region"`
}

func (r *birmevideouploader) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_video_uploader"
}

// Schema defines the schema for the resource.
func (r *birmevideouploader) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly upload and manage your videos with our intuitive Video Uploader. Enjoy seamless drag-and-drop functionality, real-time upload tracking, and support for large files, all on your preferred S3-compatible storage.`,
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
				Description: "Name of video-uploader",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"s3_secret_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"s3_aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *birmevideouploader) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmevideouploaderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-video-uploader")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-video-uploader", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
		"s3AccessKey": plan.S3accesskey.ValueString(),
		"s3SecretKey": plan.S3secretkey.ValueString(),
		"s3AwsRegion": plan.S3awsregion.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-video-uploader", instance["name"].(string), serviceAccessToken)
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
	state := birmevideouploaderModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("birme-video-uploader"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		S3endpoint: plan.S3endpoint,
		S3accesskey: plan.S3accesskey,
		S3secretkey: plan.S3secretkey,
		S3awsregion: plan.S3awsregion,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmevideouploader) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmevideouploader) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmevideouploader) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmevideouploaderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-video-uploader")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-video-uploader", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
