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
	_ resource.Resource              = &pgvectorpgvector{}
	_ resource.ResourceWithConfigure = &pgvectorpgvector{}
)

func Newpgvectorpgvector() resource.Resource {
	return &pgvectorpgvector{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newpgvectorpgvector)
}

func (r *pgvectorpgvector) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// pgvectorpgvector is the resource implementation.
type pgvectorpgvector struct {
	osaasContext *osaasclient.Context
}

type pgvectorpgvectorModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Postgrespassword         types.String       `tfsdk:"postgres_password"`
	Postgresuser         types.String       `tfsdk:"postgres_user"`
	Postgresdb         types.String       `tfsdk:"postgres_db"`
	Postgresinitdbargs         types.String       `tfsdk:"postgres_init_db_args"`
	Postgresinitdbsql         types.String       `tfsdk:"postgres_init_db_sql"`
}

func (r *pgvectorpgvector) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_pgvector_pgvector"
}

// Schema defines the schema for the resource.
func (r *pgvectorpgvector) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Enhance your database with pgvector&#39;s robust vector similarity search integrated into Postgres. Effortlessly manage vectors alongside traditional data and execute advanced nearest neighbor searches with ease.`,
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
				Description: "Name of pgvector",
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
			"postgres_init_db_sql": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *pgvectorpgvector) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pgvectorpgvectorModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("pgvector-pgvector")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "pgvector-pgvector", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PostgresPassword": plan.Postgrespassword.ValueString(),
		"PostgresUser": plan.Postgresuser.ValueString(),
		"PostgresDb": plan.Postgresdb.ValueString(),
		"PostgresInitDbArgs": plan.Postgresinitdbargs.ValueString(),
		"PostgresInitDbSql": plan.Postgresinitdbsql.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "pgvector-pgvector", instance["name"].(string), serviceAccessToken)
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
	state := pgvectorpgvectorModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("pgvector-pgvector"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Postgrespassword: plan.Postgrespassword,
		Postgresuser: plan.Postgresuser,
		Postgresdb: plan.Postgresdb,
		Postgresinitdbargs: plan.Postgresinitdbargs,
		Postgresinitdbsql: plan.Postgresinitdbsql,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *pgvectorpgvector) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *pgvectorpgvector) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *pgvectorpgvector) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pgvectorpgvectorModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("pgvector-pgvector")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "pgvector-pgvector", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
