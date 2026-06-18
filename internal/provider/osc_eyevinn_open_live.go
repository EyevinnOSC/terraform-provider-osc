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
	_ resource.Resource              = &eyevinnopenlive{}
	_ resource.ResourceWithConfigure = &eyevinnopenlive{}
)

func Neweyevinnopenlive() resource.Resource {
	return &eyevinnopenlive{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnopenlive)
}

func (r *eyevinnopenlive) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnopenlive is the resource implementation.
type eyevinnopenlive struct {
	osaasContext *osaasclient.Context
}

type eyevinnopenliveModel struct {
	InstanceUrl      types.String `tfsdk:"instance_url"`
	ServiceId        types.String `tfsdk:"service_id"`
	ExternalIp       types.String `tfsdk:"external_ip"`
	ExternalPort     types.Int32  `tfsdk:"external_port"`
	Name             types.String `tfsdk:"name"`
	Databaseurl      types.String `tfsdk:"database_url"`
	Stromurl         types.String `tfsdk:"strom_url"`
	Stromauthmode    types.String `tfsdk:"strom_auth_mode"`
	Stromaccesstoken types.String `tfsdk:"strom_access_token"`
	Corsorigin       types.String `tfsdk:"cors_origin"`
}

func (r *eyevinnopenlive) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_open_live"
}

// Schema defines the schema for the resource.
func (r *eyevinnopenlive) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Supercharge your broadcast productions with Open Live&#39;s central API server. Built with cutting-edge tech, streamline workflows, activate productions swiftly, and manage sources seamlessly. Elevate now!`,
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
				Description: "Name of open-live",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "Full CouchDB connection URL including credentials for data persistence",
			},
			"strom_url": schema.StringAttribute{
				Required:    true,
				Description: "Base URL of the Strom pipeline engine used for video flow processing",
			},
			"strom_auth_mode": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication mode for connecting to the Strom pipeline engine",
			},
			"strom_access_token": schema.StringAttribute{
				Optional:    true,
				Description: "OSC Personal Access Token for authenticating against OSC-hosted Strom instances",
			},
			"cors_origin": schema.StringAttribute{
				Optional:    true,
				Description: "Allowed CORS origin URL for the studio frontend to enable cross-origin requests to the API server.",
			},
		},
	}
}

func (r *eyevinnopenlive) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnopenliveModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-live")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-open-live", serviceAccessToken, map[string]interface{}{
		"name":             plan.Name.ValueString(),
		"DatabaseUrl":      plan.Databaseurl.ValueString(),
		"StromUrl":         plan.Stromurl.ValueString(),
		"StromAuthMode":    plan.Stromauthmode.ValueString(),
		"StromAccessToken": plan.Stromaccesstoken.ValueString(),
		"CorsOrigin":       plan.Corsorigin.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-open-live", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnopenliveModel{
		InstanceUrl:      types.StringValue(instance["url"].(string)),
		ServiceId:        types.StringValue("eyevinn-open-live"),
		ExternalIp:       types.StringValue(externalIp),
		ExternalPort:     types.Int32Value(int32(externalPort)),
		Name:             plan.Name,
		Databaseurl:      plan.Databaseurl,
		Stromurl:         plan.Stromurl,
		Stromauthmode:    plan.Stromauthmode,
		Stromaccesstoken: plan.Stromaccesstoken,
		Corsorigin:       plan.Corsorigin,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnopenlive) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnopenlive) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnopenlive) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnopenliveModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-live")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-open-live", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
