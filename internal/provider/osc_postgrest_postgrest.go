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
	_ resource.Resource              = &postgrestpostgrest{}
	_ resource.ResourceWithConfigure = &postgrestpostgrest{}
)

func Newpostgrestpostgrest() resource.Resource {
	return &postgrestpostgrest{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newpostgrestpostgrest)
}

func (r *postgrestpostgrest) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// postgrestpostgrest is the resource implementation.
type postgrestpostgrest struct {
	osaasContext *osaasclient.Context
}

type postgrestpostgrestModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dburi         types.String       `tfsdk:"db_uri"`
	Dbanonrole         types.String       `tfsdk:"db_anon_role"`
	Dbschemas         types.String       `tfsdk:"db_schemas"`
}

func (r *postgrestpostgrest) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_postgrest_postgrest"
}

// Schema defines the schema for the resource.
func (r *postgrestpostgrest) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your PostgreSQL database into a high-performance RESTful API with PostgREST. Enjoy rapid response times, enhanced security, and seamless scaling for robust, efficient app development.`,
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
				Description: "Name of postgrest",
			},
			"db_uri": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_anon_role": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"db_schemas": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *postgrestpostgrest) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postgrestpostgrestModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("postgrest-postgrest")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "postgrest-postgrest", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbUri": plan.Dburi.ValueString(),
		"DbAnonRole": plan.Dbanonrole.ValueString(),
		"DbSchemas": plan.Dbschemas.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "postgrest-postgrest", instance["name"].(string), serviceAccessToken)
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
	state := postgrestpostgrestModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("postgrest-postgrest"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dburi: plan.Dburi,
		Dbanonrole: plan.Dbanonrole,
		Dbschemas: plan.Dbschemas,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *postgrestpostgrest) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *postgrestpostgrest) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *postgrestpostgrest) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state postgrestpostgrestModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("postgrest-postgrest")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "postgrest-postgrest", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
