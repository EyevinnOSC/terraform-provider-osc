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
	_ resource.Resource              = &birmeoscpostgresql{}
	_ resource.ResourceWithConfigure = &birmeoscpostgresql{}
)

func Newbirmeoscpostgresql() resource.Resource {
	return &birmeoscpostgresql{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmeoscpostgresql)
}

func (r *birmeoscpostgresql) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmeoscpostgresql is the resource implementation.
type birmeoscpostgresql struct {
	osaasContext *osaasclient.Context
}

type birmeoscpostgresqlModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Postgrespassword         types.String       `tfsdk:"postgres_password"`
	Postgresuser         types.String       `tfsdk:"postgres_user"`
	Postgresdb         types.String       `tfsdk:"postgres_db"`
	Postgresinitdbargs         types.String       `tfsdk:"postgres_init_db_args"`
}

func (r *birmeoscpostgresql) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_osc_postgresql"
}

// Schema defines the schema for the resource.
func (r *birmeoscpostgresql) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock the full potential of your data with the PostgreSQL OSC image, seamlessly integrated for use in Eyevinn Open Source Cloud. Experience robust scalability, high security, and unmatched extensibility.`,
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
				Description: "Name of osc-postgresql",
			},
			"postgres_password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"postgres_user": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"postgres_db": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"postgres_init_db_args": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *birmeoscpostgresql) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmeoscpostgresqlModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-osc-postgresql")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-osc-postgresql", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PostgresPassword": plan.Postgrespassword.ValueString(),
		"PostgresUser": plan.Postgresuser.ValueString(),
		"PostgresDb": plan.Postgresdb.ValueString(),
		"PostgresInitDbArgs": plan.Postgresinitdbargs.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-osc-postgresql", instance["name"].(string), serviceAccessToken)
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
	state := birmeoscpostgresqlModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("birme-osc-postgresql"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Postgrespassword: plan.Postgrespassword,
		Postgresuser: plan.Postgresuser,
		Postgresdb: plan.Postgresdb,
		Postgresinitdbargs: plan.Postgresinitdbargs,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmeoscpostgresql) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmeoscpostgresql) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmeoscpostgresql) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmeoscpostgresqlModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-osc-postgresql")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-osc-postgresql", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
