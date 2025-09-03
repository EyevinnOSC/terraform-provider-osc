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
	_ resource.Resource              = &eyevinnautosubtitles{}
	_ resource.ResourceWithConfigure = &eyevinnautosubtitles{}
)

func Neweyevinnautosubtitles() resource.Resource {
	return &eyevinnautosubtitles{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnautosubtitles)
}

func (r *eyevinnautosubtitles) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnautosubtitles is the resource implementation.
type eyevinnautosubtitles struct {
	osaasContext *osaasclient.Context
}

type eyevinnautosubtitlesModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Openaikey         types.String       `tfsdk:"openaikey"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
}

func (r *eyevinnautosubtitles) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_auto_subtitles"
}

// Schema defines the schema for the resource.
func (r *eyevinnautosubtitles) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly transform audio and video files into accurate subtitles with Automatic Subtitle Generator. Utilizing Open AI Whisper, enjoy seamless integration to transcribe and format content efficiently.`,
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
				Description: "Name of auto-subtitles",
			},
			"openaikey": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnautosubtitles) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnautosubtitlesModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-auto-subtitles")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-auto-subtitles", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"openaikey": plan.Openaikey.ValueString(),
		"awsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"awsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"awsRegion": plan.Awsregion.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-auto-subtitles", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnautosubtitlesModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-auto-subtitles"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Openaikey: plan.Openaikey,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Awsregion: plan.Awsregion,
		S3endpoint: plan.S3endpoint,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnautosubtitles) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnautosubtitles) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnautosubtitles) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnautosubtitlesModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-auto-subtitles")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-auto-subtitles", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
