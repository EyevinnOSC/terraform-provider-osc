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
	_ resource.Resource              = &eyevinnopenlivestudio{}
	_ resource.ResourceWithConfigure = &eyevinnopenlivestudio{}
)

func Neweyevinnopenlivestudio() resource.Resource {
	return &eyevinnopenlivestudio{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnopenlivestudio)
}

func (r *eyevinnopenlivestudio) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnopenlivestudio is the resource implementation.
type eyevinnopenlivestudio struct {
	osaasContext *osaasclient.Context
}

type eyevinnopenlivestudioModel struct {
	InstanceUrl    types.String `tfsdk:"instance_url"`
	ServiceId      types.String `tfsdk:"service_id"`
	ExternalIp     types.String `tfsdk:"external_ip"`
	ExternalPort   types.Int32  `tfsdk:"external_port"`
	Name           types.String `tfsdk:"name"`
	Openliveurl    types.String `tfsdk:"open_live_url"`
	Oscaccesstoken types.String `tfsdk:"osc_access_token"`
}

func (r *eyevinnopenlivestudio) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_open_live_studio"
}

// Schema defines the schema for the resource.
func (r *eyevinnopenlivestudio) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Revolutionize your broadcasting with Open Live Studio, the ultimate browser-based production controller. Seamlessly integrate and manage broadcasts using cutting-edge tech for a flawless live experience.`,
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
				Description: "Name of open-live-studio",
			},
			"open_live_url": schema.StringAttribute{
				Required:    true,
				Description: "The URL of the open-live backend API that this production controller will connect to for managing sources and productions",
			},
			"osc_access_token": schema.StringAttribute{
				Optional:    true,
				Description: "Personal Access Token for Open Source Cloud (OSC) authentication and deployment operations",
			},
		},
	}
}

func (r *eyevinnopenlivestudio) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnopenlivestudioModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-live-studio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-open-live-studio", serviceAccessToken, map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"OpenLiveUrl":    plan.Openliveurl.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-open-live-studio", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnopenlivestudioModel{
		InstanceUrl:    types.StringValue(instance["url"].(string)),
		ServiceId:      types.StringValue("eyevinn-open-live-studio"),
		ExternalIp:     types.StringValue(externalIp),
		ExternalPort:   types.Int32Value(int32(externalPort)),
		Name:           plan.Name,
		Openliveurl:    plan.Openliveurl,
		Oscaccesstoken: plan.Oscaccesstoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnopenlivestudio) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnopenlivestudio) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnopenlivestudio) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnopenlivestudioModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-open-live-studio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-open-live-studio", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
