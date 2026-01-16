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
	_ resource.Resource              = &n8nion8n{}
	_ resource.ResourceWithConfigure = &n8nion8n{}
)

func Newn8nion8n() resource.Resource {
	return &n8nion8n{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newn8nion8n)
}

func (r *n8nion8n) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// n8nion8n is the resource implementation.
type n8nion8n struct {
	osaasContext *osaasclient.Context
}

type n8nion8nModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Databaseurl         types.String       `tfsdk:"database_url"`
	Runnersauthtoken         types.String       `tfsdk:"runners_auth_token"`
}

func (r *n8nion8n) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_n8n_io_n8n"
}

// Schema defines the schema for the resource.
func (r *n8nion8n) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Supercharge your team&#39;s productivity with n8n, the ultimate workflow automation platform. Enjoy seamless integration with 400&#43; apps, built-in AI, and full control over your data. Flexibility meets efficiency.`,
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
				Description: "Name of n8n",
			},
			"database_url": schema.StringAttribute{
				Optional: true,
				Description: "Connection string URL for the database that n8n uses to store workflow data, execution history, credentials, and other persistent information. This is essential for production deployments where data needs to be preserved across restarts.",
			},
			"runners_auth_token": schema.StringAttribute{
				Required: true,
				Description: "Authentication token used to secure communication between n8n main process and task runners. Required for isolating and executing code in separate processes for enhanced security.",
			},
		},
	}
}

func (r *n8nion8n) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan n8nion8nModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("n8n-io-n8n")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "n8n-io-n8n", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"RunnersAuthToken": plan.Runnersauthtoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "n8n-io-n8n", instance["name"].(string), serviceAccessToken)
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
	state := n8nion8nModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("n8n-io-n8n"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Databaseurl: plan.Databaseurl,
		Runnersauthtoken: plan.Runnersauthtoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *n8nion8n) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *n8nion8n) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *n8nion8n) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state n8nion8nModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("n8n-io-n8n")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "n8n-io-n8n", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
