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
	_ resource.Resource              = &eyevinndashmonitor{}
	_ resource.ResourceWithConfigure = &eyevinndashmonitor{}
)

func Neweyevinndashmonitor() resource.Resource {
	return &eyevinndashmonitor{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinndashmonitor)
}

func (r *eyevinndashmonitor) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinndashmonitor is the resource implementation.
type eyevinndashmonitor struct {
	osaasContext *osaasclient.Context
}

type eyevinndashmonitorModel struct {
	InstanceUrl  types.String `tfsdk:"instance_url"`
	ServiceId    types.String `tfsdk:"service_id"`
	ExternalIp   types.String `tfsdk:"external_ip"`
	ExternalPort types.Int32  `tfsdk:"external_port"`
	Name         types.String `tfsdk:"name"`
	Nodeenv      types.String `tfsdk:"node_env"`
}

func (r *eyevinndashmonitor) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_dash_monitor"
}

// Schema defines the schema for the resource.
func (r *eyevinndashmonitor) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Ensure smooth streaming experiences with DASH Stream Monitor, a powerful tool for detecting errors in DASH/MPEG-DASH live streams. Its REST API, Prometheus metrics, and Docker readiness make integration seamless.`,
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
				Description: "Name of dash-monitor",
			},
			"node_env": schema.StringAttribute{
				Optional:    true,
				Description: "",
			},
		},
	}
}

func (r *eyevinndashmonitor) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinndashmonitorModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-dash-monitor")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-dash-monitor", serviceAccessToken, map[string]interface{}{
		"name":    plan.Name.ValueString(),
		"nodeEnv": plan.Nodeenv.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-dash-monitor", instance["name"].(string), serviceAccessToken)
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
	state := eyevinndashmonitorModel{
		InstanceUrl:  types.StringValue(instance["url"].(string)),
		ServiceId:    types.StringValue("eyevinn-dash-monitor"),
		ExternalIp:   types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name:         plan.Name,
		Nodeenv:      plan.Nodeenv,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinndashmonitor) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinndashmonitor) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinndashmonitor) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinndashmonitorModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-dash-monitor")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-dash-monitor", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
