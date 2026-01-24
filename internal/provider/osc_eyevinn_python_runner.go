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
	_ resource.Resource              = &eyevinnpythonrunner{}
	_ resource.ResourceWithConfigure = &eyevinnpythonrunner{}
)

func Neweyevinnpythonrunner() resource.Resource {
	return &eyevinnpythonrunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnpythonrunner)
}

func (r *eyevinnpythonrunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnpythonrunner is the resource implementation.
type eyevinnpythonrunner struct {
	osaasContext *osaasclient.Context
}

type eyevinnpythonrunnerModel struct {
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

func (r *eyevinnpythonrunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_python_runner"
}

// Schema defines the schema for the resource.
func (r *eyevinnpythonrunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly deploy your Python web apps with our Docker-based Python Runner! Clone from GitHub or S3, install dependencies, and auto-detect frameworks for seamless app execution. Ideal for FastAPI, Flask, and more!`,
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
				Description: "Name of python-runner",
			},
			"source_url": schema.StringAttribute{
				Required: true,
				Description: "URL to the source code location. Can be either a GitHub repository URL (e.g., https://github.com/org/repo/) or an S3 URL to a zipped application (e.g., s3://bucket/app.zip)",
			},
			"git_hub_token": schema.StringAttribute{
				Optional: true,
				Description: "GitHub personal access token for accessing private repositories",
			},
			"aws_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "AWS access key ID for authenticating with S3 or S3-compatible storage services",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "AWS secret access key for authenticating with S3 or S3-compatible storage services",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "AWS region where your S3 bucket is located",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional: true,
				Description: "Custom S3 endpoint URL for MinIO or other S3-compatible storage services",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "Access token for Eyevinn Open Source Cloud configuration service",
			},
			"config_service": schema.StringAttribute{
				Optional: true,
				Description: "URL endpoint for external configuration service",
			},
		},
	}
}

func (r *eyevinnpythonrunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnpythonrunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-python-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-python-runner", serviceAccessToken, map[string]interface{}{
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

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-python-runner", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnpythonrunnerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-python-runner"),
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
func (r *eyevinnpythonrunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnpythonrunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnpythonrunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnpythonrunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-python-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-python-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
