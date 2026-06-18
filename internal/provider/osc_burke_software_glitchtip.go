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
	_ resource.Resource              = &burkesoftwareglitchtip{}
	_ resource.ResourceWithConfigure = &burkesoftwareglitchtip{}
)

func Newburkesoftwareglitchtip() resource.Resource {
	return &burkesoftwareglitchtip{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newburkesoftwareglitchtip)
}

func (r *burkesoftwareglitchtip) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// burkesoftwareglitchtip is the resource implementation.
type burkesoftwareglitchtip struct {
	osaasContext *osaasclient.Context
}

type burkesoftwareglitchtipModel struct {
	InstanceUrl  types.String `tfsdk:"instance_url"`
	ServiceId    types.String `tfsdk:"service_id"`
	ExternalIp   types.String `tfsdk:"external_ip"`
	ExternalPort types.Int32  `tfsdk:"external_port"`
	Name         types.String `tfsdk:"name"`
	Secretkey    types.String `tfsdk:"secret_key"`
	Databaseurl  types.String `tfsdk:"database_url"`
}

func (r *burkesoftwareglitchtip) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_burke_software_glitchtip"
}

// Schema defines the schema for the resource.
func (r *burkesoftwareglitchtip) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Seamlessly monitor and track app issues with GlitchTip! Experience smooth deployment on DigitalOcean or Heroku, complete with robust backend and frontend integration, plus Postgres and Redis flexibility.`,
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
				Description: "Name of glitchtip",
			},
			"secret_key": schema.StringAttribute{
				Required:    true,
				Description: "",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "",
			},
		},
	}
}

func (r *burkesoftwareglitchtip) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan burkesoftwareglitchtipModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("burke-software-glitchtip")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "burke-software-glitchtip", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"secretKey":   plan.Secretkey.ValueString(),
		"databaseUrl": plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "burke-software-glitchtip", instance["name"].(string), serviceAccessToken)
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
	state := burkesoftwareglitchtipModel{
		InstanceUrl:  types.StringValue(instance["url"].(string)),
		ServiceId:    types.StringValue("burke-software-glitchtip"),
		ExternalIp:   types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name:         plan.Name,
		Secretkey:    plan.Secretkey,
		Databaseurl:  plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *burkesoftwareglitchtip) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *burkesoftwareglitchtip) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *burkesoftwareglitchtip) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state burkesoftwareglitchtipModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("burke-software-glitchtip")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "burke-software-glitchtip", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
