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
	_ resource.Resource              = &birmecontactformsvc{}
	_ resource.ResourceWithConfigure = &birmecontactformsvc{}
)

func Newbirmecontactformsvc() resource.Resource {
	return &birmecontactformsvc{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmecontactformsvc)
}

func (r *birmecontactformsvc) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmecontactformsvc is the resource implementation.
type birmecontactformsvc struct {
	osaasContext *osaasclient.Context
}

type birmecontactformsvcModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Transport         types.Int32       `tfsdk:"transport"`
	Slackbottoken         types.String       `tfsdk:"slack_bot_token"`
	Slackchannelid         types.String       `tfsdk:"slack_channel_id"`
}

func (r *birmecontactformsvc) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_contact_form_svc_resource"
}

// Schema defines the schema for the resource.
func (r *birmecontactformsvc) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your communication with our Contact Form Service! Seamlessly send messages from your website directly to Slack. Easy-to-install, Docker-ready backend ensures you never miss a lead. Try it now!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of service",
			},
			"transport": schema.Int32Attribute{
				Required: true,
				Description: "",
			},
			"slack_bot_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"slack_channel_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *birmecontactformsvc) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmecontactformsvcModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-contact-form-svc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-contact-form-svc", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"Transport": plan.Transport,
		"SlackBotToken": plan.Slackbottoken.ValueString(),
		"SlackChannelId": plan.Slackchannelid.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-contact-form-svc", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := birmecontactformsvcModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Transport: plan.Transport,
		Slackbottoken: plan.Slackbottoken,
		Slackchannelid: plan.Slackchannelid,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmecontactformsvc) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmecontactformsvc) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmecontactformsvc) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmecontactformsvcModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-contact-form-svc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-contact-form-svc", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}