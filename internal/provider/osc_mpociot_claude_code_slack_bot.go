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
	_ resource.Resource              = &mpociotclaudecodeslackbot{}
	_ resource.ResourceWithConfigure = &mpociotclaudecodeslackbot{}
)

func Newmpociotclaudecodeslackbot() resource.Resource {
	return &mpociotclaudecodeslackbot{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newmpociotclaudecodeslackbot)
}

func (r *mpociotclaudecodeslackbot) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// mpociotclaudecodeslackbot is the resource implementation.
type mpociotclaudecodeslackbot struct {
	osaasContext *osaasclient.Context
}

type mpociotclaudecodeslackbotModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Slackbottoken         types.String       `tfsdk:"slack_bot_token"`
	Slackapptoken         types.String       `tfsdk:"slack_app_token"`
	Slacksigningsecret         types.String       `tfsdk:"slack_signing_secret"`
	Anthropicapikey         types.String       `tfsdk:"anthropic_api_key"`
	Githubappid         types.String       `tfsdk:"github_app_id"`
	Githubprivatekey         types.String       `tfsdk:"github_private_key"`
	Githubinstallationid         types.String       `tfsdk:"github_installation_id"`
	Githubtoken         types.String       `tfsdk:"github_token"`
}

func (r *mpociotclaudecodeslackbot) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_mpociot_claude_code_slack_bot"
}

// Schema defines the schema for the resource.
func (r *mpociotclaudecodeslackbot) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost productivity with Claude Code Slack Bot! Get AI-driven coding assistance right in Slack. Chat in threads, stream responses, and keep context seamlessly across messages. Transform your coding workflow today!`,
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
				Description: "Name of claude-code-slack-bot",
			},
			"slack_bot_token": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"slack_app_token": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"slack_signing_secret": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"anthropic_api_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"github_app_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"github_private_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"github_installation_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"github_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *mpociotclaudecodeslackbot) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mpociotclaudecodeslackbotModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("mpociot-claude-code-slack-bot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "mpociot-claude-code-slack-bot", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SlackBotToken": plan.Slackbottoken.ValueString(),
		"SlackAppToken": plan.Slackapptoken.ValueString(),
		"SlackSigningSecret": plan.Slacksigningsecret.ValueString(),
		"AnthropicApiKey": plan.Anthropicapikey.ValueString(),
		"GithubAppId": plan.Githubappid.ValueString(),
		"GithubPrivateKey": plan.Githubprivatekey.ValueString(),
		"GithubInstallationId": plan.Githubinstallationid.ValueString(),
		"GithubToken": plan.Githubtoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "mpociot-claude-code-slack-bot", instance["name"].(string), serviceAccessToken)
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
	state := mpociotclaudecodeslackbotModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("mpociot-claude-code-slack-bot"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Slackbottoken: plan.Slackbottoken,
		Slackapptoken: plan.Slackapptoken,
		Slacksigningsecret: plan.Slacksigningsecret,
		Anthropicapikey: plan.Anthropicapikey,
		Githubappid: plan.Githubappid,
		Githubprivatekey: plan.Githubprivatekey,
		Githubinstallationid: plan.Githubinstallationid,
		Githubtoken: plan.Githubtoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *mpociotclaudecodeslackbot) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mpociotclaudecodeslackbot) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mpociotclaudecodeslackbot) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mpociotclaudecodeslackbotModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("mpociot-claude-code-slack-bot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "mpociot-claude-code-slack-bot", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
