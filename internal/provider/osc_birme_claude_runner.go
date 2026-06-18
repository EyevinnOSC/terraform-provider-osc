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
	_ resource.Resource              = &birmeclauderunner{}
	_ resource.ResourceWithConfigure = &birmeclauderunner{}
)

func Newbirmeclauderunner() resource.Resource {
	return &birmeclauderunner{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmeclauderunner)
}

func (r *birmeclauderunner) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmeclauderunner is the resource implementation.
type birmeclauderunner struct {
	osaasContext *osaasclient.Context
}

type birmeclauderunnerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Prompt         types.String       `tfsdk:"prompt"`
	Anthropicapikey         types.String       `tfsdk:"anthropic_api_key"`
	Claudecodeoauthtoken         types.String       `tfsdk:"claude_code_oauth_token"`
	Sourceurl         types.String       `tfsdk:"source_url"`
	Gittoken         types.String       `tfsdk:"git_token"`
	Model         types.String       `tfsdk:"model"`
	Maxturns         types.String       `tfsdk:"max_turns"`
	Allowedtools         types.String       `tfsdk:"allowed_tools"`
	Disallowedtools         types.String       `tfsdk:"disallowed_tools"`
	Subpath         types.String       `tfsdk:"sub_path"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Configsvc         types.String       `tfsdk:"config_svc"`
	Configapikey         types.String       `tfsdk:"config_api_key"`
	Oscmcpurl         types.String       `tfsdk:"osc_mcp_url"`
}

func (r *birmeclauderunner) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_claude_runner"
}

// Schema defines the schema for the resource.
func (r *birmeclauderunner) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your AI-driven operations with Claude Runner. Effortlessly execute AI tasks in a container, pulling directly from your Git repository. Simplify agent workflows, automate code analysis, and boost productivity seamlessly. Perfect for dynamic environments requiring flexibility and precision.
`,
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
				Description: "Name of claude-runner",
			},
			"prompt": schema.StringAttribute{
				Required: true,
				Description: "The task or prompt for Claude to execute within the cloned repository",
			},
			"anthropic_api_key": schema.StringAttribute{
				Optional: true,
				Description: "Anthropic API key for Claude authentication",
			},
			"claude_code_oauth_token": schema.StringAttribute{
				Optional: true,
				Description: "Claude OAuth token as an alternative authentication method to the Anthropic API key",
			},
			"source_url": schema.StringAttribute{
				Required: true,
				Description: "Git repository URL to clone containing the Claude Code configuration and source code",
			},
			"git_token": schema.StringAttribute{
				Optional: true,
				Description: "Token for cloning private repositories, supporting GitHub Personal Access Tokens and Gitea-style tokens",
			},
			"model": schema.StringAttribute{
				Optional: true,
				Description: "Specifies which Claude model to use for the execution",
			},
			"max_turns": schema.StringAttribute{
				Optional: true,
				Description: "Maximum number of agentic turns Claude can perform during task execution",
			},
			"allowed_tools": schema.StringAttribute{
				Optional: true,
				Description: "Comma-separated list of tools that Claude is allowed to use during execution",
			},
			"disallowed_tools": schema.StringAttribute{
				Optional: true,
				Description: "Comma-separated list of tools that Claude is not allowed to use during execution",
			},
			"sub_path": schema.StringAttribute{
				Optional: true,
				Description: "Subdirectory within the cloned repository to use as the working directory",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "Open Source Cloud access token that configures an MCP server for OSC integration",
			},
			"config_svc": schema.StringAttribute{
				Optional: true,
				Description: "Name of an OSC Application Config Service instance for loading environment variables",
			},
			"config_api_key": schema.StringAttribute{
				Optional: true,
				Description: "API key for encrypted parameter store to decrypt secret parameters",
			},
			"osc_mcp_url": schema.StringAttribute{
				Optional: true,
				Description: "Override URL for the OSC MCP server",
			},
		},
	}
}

func (r *birmeclauderunner) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmeclauderunnerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-claude-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-claude-runner", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"Prompt": plan.Prompt.ValueString(),
		"AnthropicApiKey": plan.Anthropicapikey.ValueString(),
		"ClaudeCodeOauthToken": plan.Claudecodeoauthtoken.ValueString(),
		"SourceUrl": plan.Sourceurl.ValueString(),
		"GitToken": plan.Gittoken.ValueString(),
		"Model": plan.Model.ValueString(),
		"MaxTurns": plan.Maxturns.ValueString(),
		"AllowedTools": plan.Allowedtools.ValueString(),
		"DisallowedTools": plan.Disallowedtools.ValueString(),
		"SubPath": plan.Subpath.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"ConfigSvc": plan.Configsvc.ValueString(),
		"ConfigApiKey": plan.Configapikey.ValueString(),
		"OscMcpUrl": plan.Oscmcpurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-claude-runner", instance["name"].(string), serviceAccessToken)
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
	state := birmeclauderunnerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("birme-claude-runner"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Prompt: plan.Prompt,
		Anthropicapikey: plan.Anthropicapikey,
		Claudecodeoauthtoken: plan.Claudecodeoauthtoken,
		Sourceurl: plan.Sourceurl,
		Gittoken: plan.Gittoken,
		Model: plan.Model,
		Maxturns: plan.Maxturns,
		Allowedtools: plan.Allowedtools,
		Disallowedtools: plan.Disallowedtools,
		Subpath: plan.Subpath,
		Oscaccesstoken: plan.Oscaccesstoken,
		Configsvc: plan.Configsvc,
		Configapikey: plan.Configapikey,
		Oscmcpurl: plan.Oscmcpurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmeclauderunner) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmeclauderunner) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmeclauderunner) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmeclauderunnerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-claude-runner")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-claude-runner", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
