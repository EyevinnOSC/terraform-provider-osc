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
	_ resource.Resource              = &danigarciavaultwarden{}
	_ resource.ResourceWithConfigure = &danigarciavaultwarden{}
)

func Newdanigarciavaultwarden() resource.Resource {
	return &danigarciavaultwarden{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newdanigarciavaultwarden)
}

func (r *danigarciavaultwarden) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// danigarciavaultwarden is the resource implementation.
type danigarciavaultwarden struct {
	osaasContext *osaasclient.Context
}

type danigarciavaultwardenModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Admintoken         types.String       `tfsdk:"admin_token"`
	Webvaultenabled         bool       `tfsdk:"web_vault_enabled"`
	Smtphost         types.String       `tfsdk:"smtp_host"`
	Smtpport         types.String       `tfsdk:"smtp_port"`
	Smtpfrom         types.String       `tfsdk:"smtp_from"`
	Smtpusername         types.String       `tfsdk:"smtp_username"`
	Smtppassword         types.String       `tfsdk:"smtp_password"`
	Signupsallowed         bool       `tfsdk:"signups_allowed"`
	Invitationsallowed         bool       `tfsdk:"invitations_allowed"`
	Showpasswordhint         bool       `tfsdk:"show_password_hint"`
	Databaseurl         types.String       `tfsdk:"database_url"`
}

func (r *danigarciavaultwarden) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_dani_garcia_vaultwarden"
}

// Schema defines the schema for the resource.
func (r *danigarciavaultwarden) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Experience seamless, lightweight password management with Vaultwarden! Our Rust-based server implementation is fully compatible with Bitwarden clients, offering top-notch security for self-hosted setups without resource-heavy overhead.`,
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
				Description: "Name of vaultwarden",
			},
			"admin_token": schema.StringAttribute{
				Optional: true,
				Description: "Authentication token for accessing the Vaultwarden admin backend interface",
			},
			"web_vault_enabled": schema.BoolAttribute{
				Optional: true,
				Description: "Controls whether the web vault interface is enabled and accessible",
			},
			"smtp_host": schema.StringAttribute{
				Optional: true,
				Description: "SMTP server hostname or IP address for sending emails",
			},
			"smtp_port": schema.StringAttribute{
				Optional: true,
				Description: "Port number for the SMTP server connection",
			},
			"smtp_from": schema.StringAttribute{
				Optional: true,
				Description: "Email address that appears as the sender for all outgoing emails from Vaultwarden",
			},
			"smtp_username": schema.StringAttribute{
				Optional: true,
				Description: "Username for authenticating with the SMTP server",
			},
			"smtp_password": schema.StringAttribute{
				Optional: true,
				Description: "Password for authenticating with the SMTP server",
			},
			"signups_allowed": schema.BoolAttribute{
				Optional: true,
				Description: "Controls whether new users can create accounts directly on the Vaultwarden instance",
			},
			"invitations_allowed": schema.BoolAttribute{
				Optional: true,
				Description: "Controls whether existing users can invite new users to join the Vaultwarden instance",
			},
			"show_password_hint": schema.BoolAttribute{
				Optional: true,
				Description: "Controls whether password hints are displayed to users who request them",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "Connection string for the database where Vaultwarden stores all data including user accounts, passwords, and organizational information",
			},
		},
	}
}

func (r *danigarciavaultwarden) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan danigarciavaultwardenModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("dani-garcia-vaultwarden")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "dani-garcia-vaultwarden", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"adminToken": plan.Admintoken.ValueString(),
		"webVaultEnabled": plan.Webvaultenabled,
		"smtpHost": plan.Smtphost.ValueString(),
		"smtpPort": plan.Smtpport.ValueString(),
		"smtpFrom": plan.Smtpfrom.ValueString(),
		"smtpUsername": plan.Smtpusername.ValueString(),
		"smtpPassword": plan.Smtppassword.ValueString(),
		"signupsAllowed": plan.Signupsallowed,
		"invitationsAllowed": plan.Invitationsallowed,
		"showPasswordHint": plan.Showpasswordhint,
		"databaseUrl": plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "dani-garcia-vaultwarden", instance["name"].(string), serviceAccessToken)
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
	state := danigarciavaultwardenModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("dani-garcia-vaultwarden"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Admintoken: plan.Admintoken,
		Webvaultenabled: plan.Webvaultenabled,
		Smtphost: plan.Smtphost,
		Smtpport: plan.Smtpport,
		Smtpfrom: plan.Smtpfrom,
		Smtpusername: plan.Smtpusername,
		Smtppassword: plan.Smtppassword,
		Signupsallowed: plan.Signupsallowed,
		Invitationsallowed: plan.Invitationsallowed,
		Showpasswordhint: plan.Showpasswordhint,
		Databaseurl: plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *danigarciavaultwarden) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *danigarciavaultwarden) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *danigarciavaultwarden) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state danigarciavaultwardenModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("dani-garcia-vaultwarden")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "dani-garcia-vaultwarden", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
