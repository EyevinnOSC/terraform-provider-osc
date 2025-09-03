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
	_ resource.Resource              = &emedvedevslackinextended{}
	_ resource.ResourceWithConfigure = &emedvedevslackinextended{}
)

func Newemedvedevslackinextended() resource.Resource {
	return &emedvedevslackinextended{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newemedvedevslackinextended)
}

func (r *emedvedevslackinextended) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// emedvedevslackinextended is the resource implementation.
type emedvedevslackinextended struct {
	osaasContext *osaasclient.Context
}

type emedvedevslackinextendedModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Slackworkspaceid         types.String       `tfsdk:"slack_workspace_id"`
	Slackapitoken         types.String       `tfsdk:"slack_api_token"`
	Slackinviteurl         types.String       `tfsdk:"slack_invite_url"`
	Recaptchasecret         types.String       `tfsdk:"recaptcha_secret"`
	Recaptchasitekey         types.String       `tfsdk:"recaptcha_sitekey"`
	Theme         types.String       `tfsdk:"theme"`
	Cocurl         types.String       `tfsdk:"co_c_url"`
}

func (r *emedvedevslackinextended) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_emedvedev_slackin_extended"
}

// Schema defines the schema for the resource.
func (r *emedvedevslackinextended) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost your Slack community engagement with Slackin-Extended! Our customizable platform offers real-time user tracking, effortless invites, and abuse prevention. Enhance user experience with personalized themes and simple integration options. Perfect for building and maintaining a vibrant online community!`,
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
				Description: "Name of slackin-extended",
			},
			"slack_workspace_id": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"slack_api_token": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"slack_invite_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"recaptcha_secret": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"recaptcha_sitekey": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"theme": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"co_c_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *emedvedevslackinextended) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan emedvedevslackinextendedModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("emedvedev-slackin-extended")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "emedvedev-slackin-extended", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SlackWorkspaceId": plan.Slackworkspaceid.ValueString(),
		"SlackApiToken": plan.Slackapitoken.ValueString(),
		"SlackInviteUrl": plan.Slackinviteurl.ValueString(),
		"RecaptchaSecret": plan.Recaptchasecret.ValueString(),
		"RecaptchaSitekey": plan.Recaptchasitekey.ValueString(),
		"Theme": plan.Theme.ValueString(),
		"CoCUrl": plan.Cocurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "emedvedev-slackin-extended", instance["name"].(string), serviceAccessToken)
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
	state := emedvedevslackinextendedModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("emedvedev-slackin-extended"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Slackworkspaceid: plan.Slackworkspaceid,
		Slackapitoken: plan.Slackapitoken,
		Slackinviteurl: plan.Slackinviteurl,
		Recaptchasecret: plan.Recaptchasecret,
		Recaptchasitekey: plan.Recaptchasitekey,
		Theme: plan.Theme,
		Cocurl: plan.Cocurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *emedvedevslackinextended) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *emedvedevslackinextended) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *emedvedevslackinextended) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state emedvedevslackinextendedModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("emedvedev-slackin-extended")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "emedvedev-slackin-extended", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
