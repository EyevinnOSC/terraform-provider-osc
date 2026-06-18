package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource              = &birmecodexrunner{}
	_ resource.ResourceWithConfigure = &birmecodexrunner{}
)

func Newbirmecodexrunner() resource.Resource {
	return &birmecodexrunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmecodexrunner)
}

func (r *birmecodexrunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmecodexrunner is the resource implementation.
type birmecodexrunner struct {
	osaasContext *osaasclient.Context
}

type birmecodexrunnerModel struct {
	InstanceUrl     types.String `tfsdk:"instance_url"`
	ServiceId       types.String `tfsdk:"service_id"`
	ExternalIp      types.String `tfsdk:"external_ip"`
	ExternalPort    types.Int32  `tfsdk:"external_port"`
	Name            types.String `tfsdk:"name"`
	Prompt          types.String `tfsdk:"prompt"`
	Codexapikey     types.String `tfsdk:"codex_api_key"`
	Openaiapikey    types.String `tfsdk:"openai_api_key"`
	Sourceurl       types.String `tfsdk:"source_url"`
	Gittoken        types.String `tfsdk:"git_token"`
	Model           types.String `tfsdk:"model"`
	Maxturns        types.String `tfsdk:"max_turns"`
	Allowedtools    types.String `tfsdk:"allowed_tools"`
	Disallowedtools types.String `tfsdk:"disallowed_tools"`
	Subpath         types.String `tfsdk:"sub_path"`
	Oscaccesstoken  types.String `tfsdk:"osc_access_token"`
	Configsvc       types.String `tfsdk:"config_svc"`
	Configapikey    types.String `tfsdk:"config_api_key"`
}

func (r *birmecodexrunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_codex_runner"
}

// Schema defines the schema for the resource.
func (r *birmecodexrunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly automate your software with Codex Runner! Seamlessly integrate OpenAI Codex in a container to execute tasks on your Git repositories. Simplify your workflows and boost productivity today!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed:    true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed:    true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of codex-runner",
			},
			"prompt": schema.StringAttribute{
				Required:    true,
				Description: "The task or prompt for Codex to execute on the cloned repository",
			},
			"codex_api_key": schema.StringAttribute{
				Optional:    true,
				Description: "OpenAI API key for authenticating with Codex services",
			},
			"openai_api_key": schema.StringAttribute{
				Optional:    true,
				Description: "OpenAI API key (alias for CODEX_API_KEY, gets normalized internally)",
			},
			"source_url": schema.StringAttribute{
				Required:    true,
				Description: "Git repository URL to clone and work with",
			},
			"git_token": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication token for cloning private repositories",
			},
			"model": schema.StringAttribute{
				Optional:    true,
				Description: "AI model to use for the Codex session",
			},
			"max_turns": schema.StringAttribute{
				Optional:    true,
				Description: "Maximum number of conversation turns or iterations for the Codex session",
			},
			"allowed_tools": schema.StringAttribute{
				Optional:    true,
				Description: "Comma-separated list of tools that Codex is permitted to use during execution",
			},
			"disallowed_tools": schema.StringAttribute{
				Optional:    true,
				Description: "Comma-separated list of tools that Codex is prohibited from using during execution",
			},
			"sub_path": schema.StringAttribute{
				Optional:    true,
				Description: "Subdirectory within the cloned repository to use as the working directory",
			},
			"osc_access_token": schema.StringAttribute{
				Optional:    true,
				Description: "Open Source Cloud access token for enabling OSC MCP server and config service integration",
			},
			"config_svc": schema.StringAttribute{
				Optional:    true,
				Description: "Name of an OSC Application Config Service instance for loading additional environment variables",
			},
			"config_api_key": schema.StringAttribute{
				Optional:    true,
				Description: "API key for accessing encrypted parameters in the parameter store",
			},
		},
	}
}

func (r *birmecodexrunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmecodexrunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-codex-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-codex-runner", serviceAccessToken, map[string]interface{}{
		"name":            plan.Name.ValueString(),
		"Prompt":          plan.Prompt.ValueString(),
		"CodexApiKey":     plan.Codexapikey.ValueString(),
		"OpenaiApiKey":    plan.Openaiapikey.ValueString(),
		"SourceUrl":       plan.Sourceurl.ValueString(),
		"GitToken":        plan.Gittoken.ValueString(),
		"Model":           plan.Model.ValueString(),
		"MaxTurns":        plan.Maxturns.ValueString(),
		"AllowedTools":    plan.Allowedtools.ValueString(),
		"DisallowedTools": plan.Disallowedtools.ValueString(),
		"SubPath":         plan.Subpath.ValueString(),
		"OscAccessToken":  plan.Oscaccesstoken.ValueString(),
		"ConfigSvc":       plan.Configsvc.ValueString(),
		"ConfigApiKey":    plan.Configapikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-codex-runner", instance["name"].(string), serviceAccessToken)
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
	state := birmecodexrunnerModel{
		InstanceUrl:     types.StringValue(instance["url"].(string)),
		ServiceId:       types.StringValue("birme-codex-runner"),
		ExternalIp:      types.StringValue(externalIp),
		ExternalPort:    types.Int32Value(int32(externalPort)),
		Name:            plan.Name,
		Prompt:          plan.Prompt,
		Codexapikey:     plan.Codexapikey,
		Openaiapikey:    plan.Openaiapikey,
		Sourceurl:       plan.Sourceurl,
		Gittoken:        plan.Gittoken,
		Model:           plan.Model,
		Maxturns:        plan.Maxturns,
		Allowedtools:    plan.Allowedtools,
		Disallowedtools: plan.Disallowedtools,
		Subpath:         plan.Subpath,
		Oscaccesstoken:  plan.Oscaccesstoken,
		Configsvc:       plan.Configsvc,
		Configapikey:    plan.Configapikey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmecodexrunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmecodexrunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmecodexrunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmecodexrunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-codex-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-codex-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
