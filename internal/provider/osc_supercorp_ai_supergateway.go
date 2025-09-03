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
	_ resource.Resource              = &supercorpaisupergateway{}
	_ resource.ResourceWithConfigure = &supercorpaisupergateway{}
)

func Newsupercorpaisupergateway() resource.Resource {
	return &supercorpaisupergateway{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newsupercorpaisupergateway)
}

func (r *supercorpaisupergateway) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// supercorpaisupergateway is the resource implementation.
type supercorpaisupergateway struct {
	osaasContext *osaasclient.Context
}

type supercorpaisupergatewayModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Mcpserver         types.String       `tfsdk:"mcp_server"`
	Envvars         types.String       `tfsdk:"env_vars"`
}

func (r *supercorpaisupergateway) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_supercorp_ai_supergateway"
}

// Schema defines the schema for the resource.
func (r *supercorpaisupergateway) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock seamless stdio MCP server connectivity with Supergateway! Run servers over SSE effortlessly, ideal for remote access and debugging. Start with one command to deliver powerful, real-time interactions!`,
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
				Description: "Name of supergateway",
			},
			"mcp_server": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"env_vars": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *supercorpaisupergateway) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan supercorpaisupergatewayModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("supercorp-ai-supergateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "supercorp-ai-supergateway", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"McpServer": plan.Mcpserver.ValueString(),
		"EnvVars": plan.Envvars.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "supercorp-ai-supergateway", instance["name"].(string), serviceAccessToken)
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
	state := supercorpaisupergatewayModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("supercorp-ai-supergateway"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Mcpserver: plan.Mcpserver,
		Envvars: plan.Envvars,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *supercorpaisupergateway) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *supercorpaisupergateway) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *supercorpaisupergateway) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state supercorpaisupergatewayModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("supercorp-ai-supergateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "supercorp-ai-supergateway", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
