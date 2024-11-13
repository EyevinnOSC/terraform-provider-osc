package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/eyevinn/osaas-client-go"
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
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Postgrespassword         types.String       `tfsdk:"postgres_password"`
	Postgresuser         types.String       `tfsdk:"postgres_user"`
	Postgresdb         types.String       `tfsdk:"postgres_db"`
	Postgresinitdbargs         types.String       `tfsdk:"postgres_init_db_args"`
}

func (r *birmeoscpostgresql) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_osc_postgresql_resource"
}

// Schema defines the schema for the resource.
func (r *birmeoscpostgresql) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"postgres_password": schema.StringAttribute{
				Required: true,
			},
			"postgres_user": schema.StringAttribute{
				Optional: true,
			},
			"postgres_db": schema.StringAttribute{
				Optional: true,
			},
			"postgres_init_db_args": schema.StringAttribute{
				Optional: true,
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

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-osc-postgresql", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := birmeoscpostgresqlModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
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
