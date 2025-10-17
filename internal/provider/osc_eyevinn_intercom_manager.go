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
	_ resource.Resource              = &eyevinnintercommanager{}
	_ resource.ResourceWithConfigure = &eyevinnintercommanager{}
)

func Neweyevinnintercommanager() resource.Resource {
	return &eyevinnintercommanager{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnintercommanager)
}

func (r *eyevinnintercommanager) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnintercommanager is the resource implementation.
type eyevinnintercommanager struct {
	osaasContext *osaasclient.Context
}

type eyevinnintercommanagerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Smburl         types.String       `tfsdk:"smb_url"`
	Smbapikey         types.String       `tfsdk:"smb_api_key"`
	Dburl         types.String       `tfsdk:"db_url"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Whipauthkey         types.String       `tfsdk:"whip_auth_key"`
	Iceservers         types.String       `tfsdk:"ice_servers"`
}

func (r *eyevinnintercommanager) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_intercom_manager"
}

// Schema defines the schema for the resource.
func (r *eyevinnintercommanager) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Open Source Intercom Solution providing production-grade audio quality and real-time latency. Powered by Symphony Media Bridge open source media server.

Join our Slack community for support and customization. Contact sales@eyevinn.se for further development and support. Visit Eyevinn Technology for innovative video solutions.`,
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
				Description: "Name of intercom-manager",
			},
			"smb_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"smb_api_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"db_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"whip_auth_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"ice_servers": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnintercommanager) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnintercommanagerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-intercom-manager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-intercom-manager", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"smbUrl": plan.Smburl.ValueString(),
		"smbApiKey": plan.Smbapikey.ValueString(),
		"dbUrl": plan.Dburl.ValueString(),
		"oscAccessToken": plan.Oscaccesstoken.ValueString(),
		"whipAuthKey": plan.Whipauthkey.ValueString(),
		"iceServers": plan.Iceservers.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-intercom-manager", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnintercommanagerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-intercom-manager"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Smburl: plan.Smburl,
		Smbapikey: plan.Smbapikey,
		Dburl: plan.Dburl,
		Oscaccesstoken: plan.Oscaccesstoken,
		Whipauthkey: plan.Whipauthkey,
		Iceservers: plan.Iceservers,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnintercommanager) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnintercommanager) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnintercommanager) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnintercommanagerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-intercom-manager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-intercom-manager", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
