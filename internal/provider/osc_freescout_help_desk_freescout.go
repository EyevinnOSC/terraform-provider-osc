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
	_ resource.Resource              = &freescouthelpdeskfreescout{}
	_ resource.ResourceWithConfigure = &freescouthelpdeskfreescout{}
)

func Newfreescouthelpdeskfreescout() resource.Resource {
	return &freescouthelpdeskfreescout{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newfreescouthelpdeskfreescout)
}

func (r *freescouthelpdeskfreescout) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// freescouthelpdeskfreescout is the resource implementation.
type freescouthelpdeskfreescout struct {
	osaasContext *osaasclient.Context
}

type freescouthelpdeskfreescoutModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dburl         types.String       `tfsdk:"db_url"`
	Adminemail         types.String       `tfsdk:"admin_email"`
	Adminpassword         types.String       `tfsdk:"admin_password"`
}

func (r *freescouthelpdeskfreescout) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_freescout_help_desk_freescout"
}

// Schema defines the schema for the resource.
func (r *freescouthelpdeskfreescout) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Discover FreeScout, the ultimate self-hosted help desk solution. Enjoy robust features akin to Zendesk &amp; Help Scout without conceding privacy or control. Fully customizable, mobile-friendly, and free!`,
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
				Description: "Name of freescout",
			},
			"db_url": schema.StringAttribute{
				Required: true,
				Description: "Mysql Database url in the format mysql:&#x2F;&#x2F;&lt;user&gt;:&lt;password&gt;@&lt;host&gt;:&lt;port&gt;&#x2F;&lt;database&gt;",
			},
			"admin_email": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"admin_password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *freescouthelpdeskfreescout) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan freescouthelpdeskfreescoutModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("freescout-help-desk-freescout")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "freescout-help-desk-freescout", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbUrl": plan.Dburl.ValueString(),
		"AdminEmail": plan.Adminemail.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "freescout-help-desk-freescout", instance["name"].(string), serviceAccessToken)
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
	state := freescouthelpdeskfreescoutModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("freescout-help-desk-freescout"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dburl: plan.Dburl,
		Adminemail: plan.Adminemail,
		Adminpassword: plan.Adminpassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *freescouthelpdeskfreescout) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *freescouthelpdeskfreescout) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *freescouthelpdeskfreescout) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state freescouthelpdeskfreescoutModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("freescout-help-desk-freescout")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "freescout-help-desk-freescout", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
