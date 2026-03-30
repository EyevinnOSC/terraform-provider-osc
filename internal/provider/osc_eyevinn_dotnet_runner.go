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
	_ resource.Resource              = &eyevinndotnetrunner{}
	_ resource.ResourceWithConfigure = &eyevinndotnetrunner{}
)

func Neweyevinndotnetrunner() resource.Resource {
	return &eyevinndotnetrunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinndotnetrunner)
}

func (r *eyevinndotnetrunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinndotnetrunner is the resource implementation.
type eyevinndotnetrunner struct {
	osaasContext *osaasclient.Context
}

type eyevinndotnetrunnerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Sourceurl         types.String       `tfsdk:"source_url"`
	Githubtoken         types.String       `tfsdk:"git_hub_token"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Configservice         types.String       `tfsdk:"config_service"`
	Subpath         types.String       `tfsdk:"sub_path"`
	Oscbuildcmd         types.String       `tfsdk:"osc_build_cmd"`
	Oscentry         types.String       `tfsdk:"osc_entry"`
}

func (r *eyevinndotnetrunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_dotnet_runner"
}

// Schema defines the schema for the resource.
func (r *eyevinndotnetrunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly run your .NET apps on Open Source Cloud with dotnet-runner! Seamlessly build, deploy, and manage applications right from your repository, ensuring smooth operation on port 8080.`,
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
				Description: "Name of dotnet-runner",
			},
			"source_url": schema.StringAttribute{
				Required: true,
				Description: "HTTPS URL to the Git repository containing your .NET application. You can append &#39;#branch&#39; to checkout a specific branch.",
			},
			"git_hub_token": schema.StringAttribute{
				Optional: true,
				Description: "Personal access token for accessing private repositories. Not required for public repositories.",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "OSC personal access token required for authentication when using the CONFIG_SVC option to load environment variables from an OSC app-config-svc instance.",
			},
			"config_service": schema.StringAttribute{
				Optional: true,
				Description: "Name of an OSC app-config-svc instance to load additional environment variables from for your application.",
			},
			"sub_path": schema.StringAttribute{
				Optional: true,
				Description: "Sub-directory within the repository to build, useful when your .NET project is not located in the repository root.",
			},
			"osc_build_cmd": schema.StringAttribute{
				Optional: true,
				Description: "Override the default build command used to compile your .NET application. This replaces the auto-detected &#39;dotnet publish&#39; invocation.",
			},
			"osc_entry": schema.StringAttribute{
				Optional: true,
				Description: "Override the entry DLL filename inside the published output directory. Specify the exact DLL name to run your application.",
			},
		},
	}
}

func (r *eyevinndotnetrunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinndotnetrunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-dotnet-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-dotnet-runner", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SourceUrl": plan.Sourceurl.ValueString(),
		"GitHubToken": plan.Githubtoken.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"ConfigService": plan.Configservice.ValueString(),
		"SubPath": plan.Subpath.ValueString(),
		"OscBuildCmd": plan.Oscbuildcmd.ValueString(),
		"OscEntry": plan.Oscentry.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-dotnet-runner", instance["name"].(string), serviceAccessToken)
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
	state := eyevinndotnetrunnerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-dotnet-runner"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Sourceurl: plan.Sourceurl,
		Githubtoken: plan.Githubtoken,
		Oscaccesstoken: plan.Oscaccesstoken,
		Configservice: plan.Configservice,
		Subpath: plan.Subpath,
		Oscbuildcmd: plan.Oscbuildcmd,
		Oscentry: plan.Oscentry,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinndotnetrunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinndotnetrunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinndotnetrunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinndotnetrunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-dotnet-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-dotnet-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
