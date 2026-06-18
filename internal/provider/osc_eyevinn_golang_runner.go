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
	_ resource.Resource              = &eyevinngolangrunner{}
	_ resource.ResourceWithConfigure = &eyevinngolangrunner{}
)

func Neweyevinngolangrunner() resource.Resource {
	return &eyevinngolangrunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinngolangrunner)
}

func (r *eyevinngolangrunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinngolangrunner is the resource implementation.
type eyevinngolangrunner struct {
	osaasContext *osaasclient.Context
}

type eyevinngolangrunnerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Sourceurl         types.String       `tfsdk:"source_url"`
	Githubtoken         types.String       `tfsdk:"git_hub_token"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Configservice         types.String       `tfsdk:"config_service"`
	Configapikey         types.String       `tfsdk:"config_api_key"`
	Subpath         types.String       `tfsdk:"sub_path"`
	Oscbuildcmd         types.String       `tfsdk:"osc_build_cmd"`
	Oscentry         types.String       `tfsdk:"osc_entry"`
	Cgoenabled         types.String       `tfsdk:"c_go_enabled"`
}

func (r *eyevinngolangrunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_golang_runner"
}

// Schema defines the schema for the resource.
func (r *eyevinngolangrunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your Go projects effortlessly with Golang-Runner. Deploy apps as &#34;My Apps&#34; on the Eyevinn Open Source Cloud, simplifying builds and integrations. Secure and customizable for all your cloud needs!`,
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
				Description: "Name of golang-runner",
			},
			"source_url": schema.StringAttribute{
				Required: true,
				Description: "HTTPS URL of the Git repository to clone and build. This is the primary source location for your Go application code.",
			},
			"git_hub_token": schema.StringAttribute{
				Optional: true,
				Description: "Personal access token for authenticating with private Git repositories. This is a fallback option that gets used if GIT_TOKEN is not provided.",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "OSC (Open Source Cloud) runner token used for authenticating with the OSC config service to load environment variables at startup.",
			},
			"config_service": schema.StringAttribute{
				Optional: true,
				Description: "OSC config service endpoint URL for loading environment variables at startup. Works in conjunction with OSC_ACCESS_TOKEN.",
			},
			"config_api_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"sub_path": schema.StringAttribute{
				Optional: true,
				Description: "Subdirectory within the cloned repository to use as the build root. This enables support for monorepo structures where your Go application is located in a specific folder.",
			},
			"osc_build_cmd": schema.StringAttribute{
				Optional: true,
				Description: "Override the auto-detected build command with a custom Go build command. When not set, the runner automatically detects your project structure and chooses an appropriate build command.",
			},
			"osc_entry": schema.StringAttribute{
				Optional: true,
				Description: "Override the binary executable path that will be run after the build completes. Allows you to specify a different binary to execute instead of the default.",
			},
			"c_go_enabled": schema.StringAttribute{
				Optional: true,
				Description: "Enable or disable CGO during the Go build process. Set to &#39;1&#39; to enable CGO, which allows calling C code from Go but requires gcc and increases image size.",
			},
		},
	}
}

func (r *eyevinngolangrunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinngolangrunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-golang-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-golang-runner", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SourceUrl": plan.Sourceurl.ValueString(),
		"GitHubToken": plan.Githubtoken.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"ConfigService": plan.Configservice.ValueString(),
		"ConfigApiKey": plan.Configapikey.ValueString(),
		"SubPath": plan.Subpath.ValueString(),
		"OscBuildCmd": plan.Oscbuildcmd.ValueString(),
		"OscEntry": plan.Oscentry.ValueString(),
		"CGoEnabled": plan.Cgoenabled.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-golang-runner", instance["name"].(string), serviceAccessToken)
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
	state := eyevinngolangrunnerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-golang-runner"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Sourceurl: plan.Sourceurl,
		Githubtoken: plan.Githubtoken,
		Oscaccesstoken: plan.Oscaccesstoken,
		Configservice: plan.Configservice,
		Configapikey: plan.Configapikey,
		Subpath: plan.Subpath,
		Oscbuildcmd: plan.Oscbuildcmd,
		Oscentry: plan.Oscentry,
		Cgoenabled: plan.Cgoenabled,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinngolangrunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinngolangrunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinngolangrunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinngolangrunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-golang-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-golang-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
