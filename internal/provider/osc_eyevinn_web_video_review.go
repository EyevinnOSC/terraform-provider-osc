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
	_ resource.Resource              = &eyevinnwebvideoreview{}
	_ resource.ResourceWithConfigure = &eyevinnwebvideoreview{}
)

func Neweyevinnwebvideoreview() resource.Resource {
	return &eyevinnwebvideoreview{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnwebvideoreview)
}

func (r *eyevinnwebvideoreview) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnwebvideoreview is the resource implementation.
type eyevinnwebvideoreview struct {
	osaasContext *osaasclient.Context
}

type eyevinnwebvideoreviewModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Accesskeyid         types.String       `tfsdk:"access_key_id"`
	Secretaccesskey         types.String       `tfsdk:"secret_access_key"`
	Bucket         types.String       `tfsdk:"bucket"`
	S3region         types.String       `tfsdk:"s3_region"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	Awssessiontoken         types.String       `tfsdk:"aws_session_token"`
}

func (r *eyevinnwebvideoreview) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_web_video_review"
}

// Schema defines the schema for the resource.
func (r *eyevinnwebvideoreview) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock seamless video review with Web Video Review! Stream, analyze, and navigate broadcast videos straight from S3 storage. Experience real-time analysis, dynamic timeline navigation, and powerful transcoding with unparalleled ease. Deploy effortlessly using Docker and transform your video reviewing process today!`,
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
				Description: "Name of web-video-review",
			},
			"access_key_id": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"secret_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"bucket": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_session_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnwebvideoreview) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnwebvideoreviewModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-web-video-review")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-web-video-review", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AccessKeyId": plan.Accesskeyid.ValueString(),
		"SecretAccessKey": plan.Secretaccesskey.ValueString(),
		"Bucket": plan.Bucket.ValueString(),
		"s3Region": plan.S3region.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
		"awsSessionToken": plan.Awssessiontoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-web-video-review", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnwebvideoreviewModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-web-video-review"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Accesskeyid: plan.Accesskeyid,
		Secretaccesskey: plan.Secretaccesskey,
		Bucket: plan.Bucket,
		S3region: plan.S3region,
		S3endpoint: plan.S3endpoint,
		Awssessiontoken: plan.Awssessiontoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnwebvideoreview) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnwebvideoreview) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnwebvideoreview) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnwebvideoreviewModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-web-video-review")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-web-video-review", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
