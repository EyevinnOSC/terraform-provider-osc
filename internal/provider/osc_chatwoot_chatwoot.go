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
	_ resource.Resource              = &chatwootchatwoot{}
	_ resource.ResourceWithConfigure = &chatwootchatwoot{}
)

func Newchatwootchatwoot() resource.Resource {
	return &chatwootchatwoot{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newchatwootchatwoot)
}

func (r *chatwootchatwoot) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// chatwootchatwoot is the resource implementation.
type chatwootchatwoot struct {
	osaasContext *osaasclient.Context
}

type chatwootchatwootModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Databaseurl         types.String       `tfsdk:"database_url"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Secretkeybase         types.String       `tfsdk:"secret_key_base"`
	Smtpaddress         types.String       `tfsdk:"smtp_address"`
	Smtpport         types.String       `tfsdk:"smtp_port"`
	Smtpusername         types.String       `tfsdk:"smtp_username"`
	Smtppassword         types.String       `tfsdk:"smtp_password"`
	Mailersenderemail         types.String       `tfsdk:"mailer_sender_email"`
}

func (r *chatwootchatwoot) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_chatwoot_chatwoot"
}

// Schema defines the schema for the resource.
func (r *chatwootchatwoot) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your customer service with Chatwoot, the open-source platform that centralizes conversations across channels. Empower your team with AI-driven support, omnichannel integration, and insightful analytics.`,
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
				Description: "Name of chatwoot",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "Database connection URL for PostgreSQL database that stores all Chatwoot data including conversations, contacts, agents, and configuration",
			},
			"redis_url": schema.StringAttribute{
				Required: true,
				Description: "Redis connection URL used for caching, session storage, background job processing, and real-time features like live chat",
			},
			"secret_key_base": schema.StringAttribute{
				Required: true,
				Description: "Rails application secret key used for encrypting sessions, cookies, and other sensitive data within the application",
			},
			"smtp_address": schema.StringAttribute{
				Optional: true,
				Description: "SMTP server hostname or IP address for sending outbound emails including notifications, password resets, and conversation replies",
			},
			"smtp_port": schema.StringAttribute{
				Optional: true,
				Description: "SMTP server port number for email delivery, typically 587 for TLS or 465 for SSL connections",
			},
			"smtp_username": schema.StringAttribute{
				Optional: true,
				Description: "Username for authenticating with the SMTP server when sending emails from Chatwoot",
			},
			"smtp_password": schema.StringAttribute{
				Optional: true,
				Description: "Password or app-specific password for SMTP server authentication when sending emails",
			},
			"mailer_sender_email": schema.StringAttribute{
				Optional: true,
				Description: "Email address that appears as the sender for all outbound emails from Chatwoot including notifications and system messages",
			},
		},
	}
}

func (r *chatwootchatwoot) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan chatwootchatwootModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("chatwoot-chatwoot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "chatwoot-chatwoot", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"SecretKeyBase": plan.Secretkeybase.ValueString(),
		"SmtpAddress": plan.Smtpaddress.ValueString(),
		"SmtpPort": plan.Smtpport.ValueString(),
		"SmtpUsername": plan.Smtpusername.ValueString(),
		"SmtpPassword": plan.Smtppassword.ValueString(),
		"MailerSenderEmail": plan.Mailersenderemail.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "chatwoot-chatwoot", instance["name"].(string), serviceAccessToken)
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
	state := chatwootchatwootModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("chatwoot-chatwoot"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Databaseurl: plan.Databaseurl,
		Redisurl: plan.Redisurl,
		Secretkeybase: plan.Secretkeybase,
		Smtpaddress: plan.Smtpaddress,
		Smtpport: plan.Smtpport,
		Smtpusername: plan.Smtpusername,
		Smtppassword: plan.Smtppassword,
		Mailersenderemail: plan.Mailersenderemail,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *chatwootchatwoot) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *chatwootchatwoot) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *chatwootchatwoot) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state chatwootchatwootModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("chatwoot-chatwoot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "chatwoot-chatwoot", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
