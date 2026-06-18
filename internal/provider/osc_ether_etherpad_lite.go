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
	_ resource.Resource              = &etheretherpadlite{}
	_ resource.ResourceWithConfigure = &etheretherpadlite{}
)

func Newetheretherpadlite() resource.Resource {
	return &etheretherpadlite{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newetheretherpadlite)
}

func (r *etheretherpadlite) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// etheretherpadlite is the resource implementation.
type etheretherpadlite struct {
	osaasContext *osaasclient.Context
}

type etheretherpadliteModel struct {
	InstanceUrl  types.String `tfsdk:"instance_url"`
	ServiceId    types.String `tfsdk:"service_id"`
	ExternalIp   types.String `tfsdk:"external_ip"`
	ExternalPort types.Int32  `tfsdk:"external_port"`
	Name         types.String `tfsdk:"name"`
	Databaseurl  types.String `tfsdk:"database_url"`
}

func (r *etheretherpadlite) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_ether_etherpad_lite"
}

// Schema defines the schema for the resource.
func (r *etheretherpadlite) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unleash seamless collaboration with Etherpad, the ultimate real-time web editor! Host unlimited users on your servers, secure data control, and customize with essential plugins. Elevate teamwork today!`,
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
				Description: "Name of etherpad-lite",
			},
			"database_url": schema.StringAttribute{
				Optional:    true,
				Description: "Specifies the database connection URL for Etherpad. This allows you to connect to an external database instead of using the default dirtyDB driver.",
			},
		},
	}
}

func (r *etheretherpadlite) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan etheretherpadliteModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ether-etherpad-lite")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "ether-etherpad-lite", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "ether-etherpad-lite", instance["name"].(string), serviceAccessToken)
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
	state := etheretherpadliteModel{
		InstanceUrl:  types.StringValue(instance["url"].(string)),
		ServiceId:    types.StringValue("ether-etherpad-lite"),
		ExternalIp:   types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name:         plan.Name,
		Databaseurl:  plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *etheretherpadlite) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *etheretherpadlite) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *etheretherpadlite) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state etheretherpadliteModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ether-etherpad-lite")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "ether-etherpad-lite", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
