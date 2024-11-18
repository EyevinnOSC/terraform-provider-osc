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
	_ resource.Resource              = &usefathomfathom{}
	_ resource.ResourceWithConfigure = &usefathomfathom{}
)

func Newusefathomfathom() resource.Resource {
	return &usefathomfathom{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newusefathomfathom)
}

func (r *usefathomfathom) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// usefathomfathom is the resource implementation.
type usefathomfathom struct {
	osaasContext *osaasclient.Context
}

type usefathomfathomModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Adminemail         types.String       `tfsdk:"admin_email"`
	Adminpassword         types.String       `tfsdk:"admin_password"`
}

func (r *usefathomfathom) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_usefathom_fathom_resource"
}

// Schema defines the schema for the resource.
func (r *usefathomfathom) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"admin_email": schema.StringAttribute{
				Required: true,
			},
			"admin_password": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (r *usefathomfathom) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan usefathomfathomModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("usefathom-fathom")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "usefathom-fathom", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AdminEmail": plan.Adminemail.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "usefathom-fathom", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := usefathomfathomModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
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
func (r *usefathomfathom) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *usefathomfathom) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *usefathomfathom) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state usefathomfathomModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("usefathom-fathom")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "usefathom-fathom", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}