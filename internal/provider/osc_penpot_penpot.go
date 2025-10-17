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
	_ resource.Resource              = &penpotpenpot{}
	_ resource.ResourceWithConfigure = &penpotpenpot{}
)

func Newpenpotpenpot() resource.Resource {
	return &penpotpenpot{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newpenpotpenpot)
}

func (r *penpotpenpot) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// penpotpenpot is the resource implementation.
type penpotpenpot struct {
	osaasContext *osaasclient.Context
}

type penpotpenpotModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dburl         types.String       `tfsdk:"db_url"`
	Dbusername         types.String       `tfsdk:"db_username"`
	Dbpassword         types.String       `tfsdk:"db_password"`
	Redisurl         types.String       `tfsdk:"redis_url"`
}

func (r *penpotpenpot) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_penpot_penpot"
}

// Schema defines the schema for the resource.
func (r *penpotpenpot) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Revolutionize your design workflow with Penpot, the open-source tool where design meets code. Create stunning designs, prototypes, and integrate seamlessly with developers. Collaborate effortlessly!`,
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
				Description: "Name of penpot",
			},
			"db_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_username": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_password": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"redis_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *penpotpenpot) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan penpotpenpotModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("penpot-penpot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "penpot-penpot", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbUrl": plan.Dburl.ValueString(),
		"DbUsername": plan.Dbusername.ValueString(),
		"DbPassword": plan.Dbpassword.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "penpot-penpot", instance["name"].(string), serviceAccessToken)
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
	state := penpotpenpotModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("penpot-penpot"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dburl: plan.Dburl,
		Dbusername: plan.Dbusername,
		Dbpassword: plan.Dbpassword,
		Redisurl: plan.Redisurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *penpotpenpot) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *penpotpenpot) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *penpotpenpot) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state penpotpenpotModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("penpot-penpot")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "penpot-penpot", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
