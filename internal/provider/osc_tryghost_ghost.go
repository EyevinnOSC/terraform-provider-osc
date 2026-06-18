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
	_ resource.Resource              = &tryghostghost{}
	_ resource.ResourceWithConfigure = &tryghostghost{}
)

func Newtryghostghost() resource.Resource {
	return &tryghostghost{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newtryghostghost)
}

func (r *tryghostghost) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// tryghostghost is the resource implementation.
type tryghostghost struct {
	osaasContext *osaasclient.Context
}

type tryghostghostModel struct {
	InstanceUrl  types.String `tfsdk:"instance_url"`
	ServiceId    types.String `tfsdk:"service_id"`
	ExternalIp   types.String `tfsdk:"external_ip"`
	ExternalPort types.Int32  `tfsdk:"external_port"`
	Name         types.String `tfsdk:"name"`
	Databaseurl  types.String `tfsdk:"database_url"`
	Smtphost     types.String `tfsdk:"smtp_host"`
	Smtpport     types.String `tfsdk:"smtp_port"`
	Smtpuser     types.String `tfsdk:"smtp_user"`
	Smtppass     types.String `tfsdk:"smtp_pass"`
	Mailfrom     types.String `tfsdk:"mail_from"`
}

func (r *tryghostghost) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_tryghost_ghost"
}

// Schema defines the schema for the resource.
func (r *tryghostghost) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Experience the power of Ghost, the leading open-source Node.js CMS! With Ghost(Pro), launch a secure, high-performance site in 2 minutes, featuring worldwide CDN, backups, and maintenance.`,
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
				Description: "Name of ghost",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "Database connection string for Ghost&#39;s primary data storage. Ghost requires a database to store all content, users, settings, and metadata.",
			},
			"smtp_host": schema.StringAttribute{
				Optional:    true,
				Description: "SMTP server hostname for sending emails. Ghost uses this to send member notifications, password resets, and newsletter emails.",
			},
			"smtp_port": schema.StringAttribute{
				Optional:    true,
				Description: "SMTP server port number for email delivery. Common ports are 587 (TLS) or 465 (SSL) for secure email transmission.",
			},
			"smtp_user": schema.StringAttribute{
				Optional:    true,
				Description: "Username for SMTP server authentication. Required when the email provider needs authentication credentials for sending emails.",
			},
			"smtp_pass": schema.StringAttribute{
				Optional:    true,
				Description: "Password for SMTP server authentication. Used alongside SMTP_USER to authenticate with the email provider for sending emails.",
			},
			"mail_from": schema.StringAttribute{
				Optional:    true,
				Description: "Default &#39;from&#39; email address for all emails sent by Ghost. This appears as the sender address for newsletters, notifications, and system emails.",
			},
		},
	}
}

func (r *tryghostghost) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tryghostghostModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("tryghost-ghost")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "tryghost-ghost", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"SmtpHost":    plan.Smtphost.ValueString(),
		"SmtpPort":    plan.Smtpport.ValueString(),
		"SmtpUser":    plan.Smtpuser.ValueString(),
		"SmtpPass":    plan.Smtppass.ValueString(),
		"MailFrom":    plan.Mailfrom.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "tryghost-ghost", instance["name"].(string), serviceAccessToken)
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
	state := tryghostghostModel{
		InstanceUrl:  types.StringValue(instance["url"].(string)),
		ServiceId:    types.StringValue("tryghost-ghost"),
		ExternalIp:   types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name:         plan.Name,
		Databaseurl:  plan.Databaseurl,
		Smtphost:     plan.Smtphost,
		Smtpport:     plan.Smtpport,
		Smtpuser:     plan.Smtpuser,
		Smtppass:     plan.Smtppass,
		Mailfrom:     plan.Mailfrom,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *tryghostghost) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tryghostghost) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *tryghostghost) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tryghostghostModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("tryghost-ghost")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "tryghost-ghost", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
