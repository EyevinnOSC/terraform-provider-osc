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
	_ resource.Resource              = &roundcuberoundcubemail{}
	_ resource.ResourceWithConfigure = &roundcuberoundcubemail{}
)

func Newroundcuberoundcubemail() resource.Resource {
	return &roundcuberoundcubemail{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newroundcuberoundcubemail)
}

func (r *roundcuberoundcubemail) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// roundcuberoundcubemail is the resource implementation.
type roundcuberoundcubemail struct {
	osaasContext *osaasclient.Context
}

type roundcuberoundcubemailModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Imapaddress         types.String       `tfsdk:"imap_address"`
	Imapport         types.String       `tfsdk:"imap_port"`
	Smtpaddress         types.String       `tfsdk:"smtp_address"`
	Smtpport         types.String       `tfsdk:"smtp_port"`
}

func (r *roundcuberoundcubemail) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_roundcube_roundcubemail"
}

// Schema defines the schema for the resource.
func (r *roundcuberoundcubemail) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your email experience with Roundcube Webmail! Enjoy a browser-based multilang IMAP client with intuitive design, customizable skins, and extensive plugin support. Efficiency meets versatility!`,
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
				Description: "Name of roundcubemail",
			},
			"imap_address": schema.StringAttribute{
				Required: true,
				Description: "Imap URL (e.g. ssl://mail.osaas.io)",
			},
			"imap_port": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"smtp_address": schema.StringAttribute{
				Required: true,
				Description: "Smtp URL (e.g. tls://mail.osaas.io)",
			},
			"smtp_port": schema.StringAttribute{
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *roundcuberoundcubemail) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roundcuberoundcubemailModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("roundcube-roundcubemail")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "roundcube-roundcubemail", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"ImapAddress": plan.Imapaddress.ValueString(),
		"ImapPort": plan.Imapport.ValueString(),
		"SmtpAddress": plan.Smtpaddress.ValueString(),
		"SmtpPort": plan.Smtpport.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "roundcube-roundcubemail", instance["name"].(string), serviceAccessToken)
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
	state := roundcuberoundcubemailModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("roundcube-roundcubemail"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Imapaddress: plan.Imapaddress,
		Imapport: plan.Imapport,
		Smtpaddress: plan.Smtpaddress,
		Smtpport: plan.Smtpport,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *roundcuberoundcubemail) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *roundcuberoundcubemail) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *roundcuberoundcubemail) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roundcuberoundcubemailModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("roundcube-roundcubemail")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "roundcube-roundcubemail", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
