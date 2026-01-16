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
	_ resource.Resource              = &eyevinnaudioqc{}
	_ resource.ResourceWithConfigure = &eyevinnaudioqc{}
)

func Neweyevinnaudioqc() resource.Resource {
	return &eyevinnaudioqc{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnaudioqc)
}

func (r *eyevinnaudioqc) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnaudioqc is the resource implementation.
type eyevinnaudioqc struct {
	osaasContext *osaasclient.Context
}

type eyevinnaudioqcModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Cmdlineargs         types.String       `tfsdk:"cmd_line_args"`
	S3accesskeyid         types.String       `tfsdk:"s3_access_key_id"`
	S3secretaccesskey         types.String       `tfsdk:"s3_secret_access_key"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	S3endpointurl         types.String       `tfsdk:"s3_endpoint_url"`
	Awssessiontoken         types.String       `tfsdk:"aws_session_token"`
}

func (r *eyevinnaudioqc) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_audio_qc"
}

// Schema defines the schema for the resource.
func (r *eyevinnaudioqc) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your audio analysis with Audio QC – a powerful tool ensuring EBU R128 compliance. Analyze and report seamlessly with S3 support, video container integration, and efficient HTTP streaming. Perfect for achieving broadcast and music standards effortlessly!`,
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
				Description: "Name of audio-qc",
			},
			"cmd_line_args": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint_url": schema.StringAttribute{
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

func (r *eyevinnaudioqc) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnaudioqcModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-audio-qc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-audio-qc", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"cmdLineArgs": plan.Cmdlineargs.ValueString(),
		"s3AccessKeyId": plan.S3accesskeyid.ValueString(),
		"s3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"awsRegion": plan.Awsregion.ValueString(),
		"s3EndpointUrl": plan.S3endpointurl.ValueString(),
		"awsSessionToken": plan.Awssessiontoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-audio-qc", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnaudioqcModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-audio-qc"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Cmdlineargs: plan.Cmdlineargs,
		S3accesskeyid: plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		Awsregion: plan.Awsregion,
		S3endpointurl: plan.S3endpointurl,
		Awssessiontoken: plan.Awssessiontoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnaudioqc) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnaudioqc) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnaudioqc) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnaudioqcModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-audio-qc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-audio-qc", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
