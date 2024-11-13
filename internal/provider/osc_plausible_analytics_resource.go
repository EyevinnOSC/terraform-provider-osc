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
	_ resource.Resource              = &plausibleanalytics{}
	_ resource.ResourceWithConfigure = &plausibleanalytics{}
)

func Newplausibleanalytics() resource.Resource {
	return &plausibleanalytics{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newplausibleanalytics)
}

func (r *plausibleanalytics) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// plausibleanalytics is the resource implementation.
type plausibleanalytics struct {
	osaasContext *osaasclient.Context
}

type plausibleanalyticsModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Postgresqlurl         types.String       `tfsdk:"postgre_sql_url"`
	Clickhousedburl         types.String       `tfsdk:"click_house_db_url"`
}

func (r *plausibleanalytics) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_plausible_analytics_resource"
}

// Schema defines the schema for the resource.
func (r *plausibleanalytics) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"postgre_sql_url": schema.StringAttribute{
				Required: true,
			},
			"click_house_db_url": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (r *plausibleanalytics) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan plausibleanalyticsModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("plausible-analytics")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "plausible-analytics", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PostgreSQLUrl": plan.Postgresqlurl.ValueString(),
		"ClickHouseDbUrl": plan.Clickhousedburl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "plausible-analytics", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := plausibleanalyticsModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Postgresqlurl: plan.Postgresqlurl,
		Clickhousedburl: plan.Clickhousedburl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *plausibleanalytics) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *plausibleanalytics) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *plausibleanalytics) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state plausibleanalyticsModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("plausible-analytics")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "plausible-analytics", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
