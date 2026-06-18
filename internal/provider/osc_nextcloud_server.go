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
	_ resource.Resource              = &nextcloudserver{}
	_ resource.ResourceWithConfigure = &nextcloudserver{}
)

func Newnextcloudserver() resource.Resource {
	return &nextcloudserver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newnextcloudserver)
}

func (r *nextcloudserver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// nextcloudserver is the resource implementation.
type nextcloudserver struct {
	osaasContext *osaasclient.Context
}

type nextcloudserverModel struct {
	InstanceUrl   types.String `tfsdk:"instance_url"`
	ServiceId     types.String `tfsdk:"service_id"`
	ExternalIp    types.String `tfsdk:"external_ip"`
	ExternalPort  types.Int32  `tfsdk:"external_port"`
	Name          types.String `tfsdk:"name"`
	Adminuser     types.String `tfsdk:"admin_user"`
	Adminpassword types.String `tfsdk:"admin_password"`
	Databaseurl   types.String `tfsdk:"database_url"`
}

func (r *nextcloudserver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_nextcloud_server"
}

// Schema defines the schema for the resource.
func (r *nextcloudserver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Empower your data with Nextcloud! Securely store, sync, and share your files, contacts, and calendars across devices. With robust security, expandability, and ease of use, your data thrives effortlessly.`,
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
				Description: "Name of server",
			},
			"admin_user": schema.StringAttribute{
				Required:    true,
				Description: "Choose an admin username",
			},
			"admin_password": schema.StringAttribute{
				Required:    true,
				Description: "Choose an admin password",
			},
			"database_url": schema.StringAttribute{
				Optional:    true,
				Description: "Database connection configuration",
			},
		},
	}
}

func (r *nextcloudserver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan nextcloudserverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("nextcloud-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "nextcloud-server", serviceAccessToken, map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"AdminUser":     plan.Adminuser.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
		"DatabaseUrl":   plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "nextcloud-server", instance["name"].(string), serviceAccessToken)
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
	state := nextcloudserverModel{
		InstanceUrl:   types.StringValue(instance["url"].(string)),
		ServiceId:     types.StringValue("nextcloud-server"),
		ExternalIp:    types.StringValue(externalIp),
		ExternalPort:  types.Int32Value(int32(externalPort)),
		Name:          plan.Name,
		Adminuser:     plan.Adminuser,
		Adminpassword: plan.Adminpassword,
		Databaseurl:   plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *nextcloudserver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *nextcloudserver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *nextcloudserver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state nextcloudserverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("nextcloud-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "nextcloud-server", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
