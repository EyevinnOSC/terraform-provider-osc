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
	_ resource.Resource              = &eyevinnsgaiadproxy{}
	_ resource.ResourceWithConfigure = &eyevinnsgaiadproxy{}
)

func Neweyevinnsgaiadproxy() resource.Resource {
	return &eyevinnsgaiadproxy{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnsgaiadproxy)
}

func (r *eyevinnsgaiadproxy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnsgaiadproxy is the resource implementation.
type eyevinnsgaiadproxy struct {
	osaasContext *osaasclient.Context
}

type eyevinnsgaiadproxyModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Vastendpoint         types.String       `tfsdk:"vast_endpoint"`
	Originurl         types.String       `tfsdk:"origin_url"`
	Couchdbendpoint         types.String       `tfsdk:"couch_db_endpoint"`
	Couchdbtable         types.String       `tfsdk:"couch_db_table"`
	Couchdbuser         types.String       `tfsdk:"couch_db_user"`
	Couchdbpassword         types.String       `tfsdk:"couch_db_password"`
}

func (r *eyevinnsgaiadproxy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_sgai_ad_proxy_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinnsgaiadproxy) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"vast_endpoint": schema.StringAttribute{
				Required: true,
			},
			"origin_url": schema.StringAttribute{
				Required: true,
			},
			"couch_db_endpoint": schema.StringAttribute{
				Optional: true,
			},
			"couch_db_table": schema.StringAttribute{
				Optional: true,
			},
			"couch_db_user": schema.StringAttribute{
				Optional: true,
			},
			"couch_db_password": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *eyevinnsgaiadproxy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnsgaiadproxyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-sgai-ad-proxy")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"VastEndpoint": plan.Vastendpoint.ValueString(),
		"OriginUrl": plan.Originurl.ValueString(),
		"CouchDbEndpoint": plan.Couchdbendpoint.ValueString(),
		"CouchDbTable": plan.Couchdbtable.ValueString(),
		"CouchDbUser": plan.Couchdbuser.ValueString(),
		"CouchDbPassword": plan.Couchdbpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinnsgaiadproxyModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Vastendpoint: plan.Vastendpoint,
		Originurl: plan.Originurl,
		Couchdbendpoint: plan.Couchdbendpoint,
		Couchdbtable: plan.Couchdbtable,
		Couchdbuser: plan.Couchdbuser,
		Couchdbpassword: plan.Couchdbpassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnsgaiadproxy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnsgaiadproxy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnsgaiadproxy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnsgaiadproxyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-sgai-ad-proxy")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
