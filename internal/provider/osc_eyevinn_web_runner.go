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
	_ resource.Resource              = &eyevinnwebrunner{}
	_ resource.ResourceWithConfigure = &eyevinnwebrunner{}
)

func Neweyevinnwebrunner() resource.Resource {
	return &eyevinnwebrunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnwebrunner)
}

func (r *eyevinnwebrunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnwebrunner is the resource implementation.
type eyevinnwebrunner struct {
	osaasContext *osaasclient.Context
}

type eyevinnwebrunnerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Sourceurl         types.String       `tfsdk:"source_url"`
	Githubtoken         types.String       `tfsdk:"git_hub_token"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	S3endpointurl         types.String       `tfsdk:"s3_endpoint_url"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Configservice         types.String       `tfsdk:"config_service"`
}

func (r *eyevinnwebrunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_web_runner"
}

// Schema defines the schema for the resource.
func (r *eyevinnwebrunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly deploy NodeJS web apps with Web-Runner! This Docker container seamlessly handles cloning, building, and running your GitHub repositories. Simplify your deployment process today!`,
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
				Description: "Name of web-runner",
			},
			"source_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"git_hub_token": schema.StringAttribute{
				Optional: true,
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
			"s3_endpoint_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"config_service": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnwebrunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnwebrunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-web-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-web-runner", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SourceUrl": plan.Sourceurl.ValueString(),
		"GitHubToken": plan.Githubtoken.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"AwsRegion": plan.Awsregion.ValueString(),
		"S3EndpointUrl": plan.S3endpointurl.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"ConfigService": plan.Configservice.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-web-runner", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnwebrunnerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-web-runner"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Sourceurl: plan.Sourceurl,
		Githubtoken: plan.Githubtoken,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Awsregion: plan.Awsregion,
		S3endpointurl: plan.S3endpointurl,
		Oscaccesstoken: plan.Oscaccesstoken,
		Configservice: plan.Configservice,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnwebrunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnwebrunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnwebrunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnwebrunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-web-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-web-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
