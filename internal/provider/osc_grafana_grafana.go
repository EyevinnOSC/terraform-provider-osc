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
	_ resource.Resource              = &grafanagrafana{}
	_ resource.ResourceWithConfigure = &grafanagrafana{}
)

func Newgrafanagrafana() resource.Resource {
	return &grafanagrafana{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newgrafanagrafana)
}

func (r *grafanagrafana) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// grafanagrafana is the resource implementation.
type grafanagrafana struct {
	osaasContext *osaasclient.Context
}

type grafanagrafanaModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Pluginspreinstall         types.String       `tfsdk:"plugins_preinstall"`
	Allowembedorigins         types.String       `tfsdk:"allow_embed_origins"`
	Anonymousenabled         bool       `tfsdk:"anonymous_enabled"`
}

func (r *grafanagrafana) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_grafana_grafana"
}

// Schema defines the schema for the resource.
func (r *grafanagrafana) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your organization&#39;s data viewing experience with Grafana&#39;s cutting-edge visualizations and dynamic dashboards. Effortlessly explore metrics, logs, and receive alerts tailored precisely for powerful insights.`,
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
				Description: "Name of grafana",
			},
			"plugins_preinstall": schema.StringAttribute{
				Optional: true,
				Description: "Provide a list of plugins to pre install",
			},
			"allow_embed_origins": schema.StringAttribute{
				Optional: true,
				Description: "Web origin allowed to embed in an iframe",
			},
			"anonymous_enabled": schema.BoolAttribute{
				Optional: true,
				Description: "Enable anonymous access",
			},
		},
	}
}

func (r *grafanagrafana) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan grafanagrafanaModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("grafana-grafana")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "grafana-grafana", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PluginsPreinstall": plan.Pluginspreinstall.ValueString(),
		"AllowEmbedOrigins": plan.Allowembedorigins.ValueString(),
		"AnonymousEnabled": plan.Anonymousenabled,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "grafana-grafana", instance["name"].(string), serviceAccessToken)
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
	state := grafanagrafanaModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("grafana-grafana"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Pluginspreinstall: plan.Pluginspreinstall,
		Allowembedorigins: plan.Allowembedorigins,
		Anonymousenabled: plan.Anonymousenabled,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *grafanagrafana) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *grafanagrafana) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *grafanagrafana) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state grafanagrafanaModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("grafana-grafana")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "grafana-grafana", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
