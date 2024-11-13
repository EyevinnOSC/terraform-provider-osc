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
	_ resource.Resource              = &drakkansftpgo{}
	_ resource.ResourceWithConfigure = &drakkansftpgo{}
)

func Newdrakkansftpgo() resource.Resource {
	return &drakkansftpgo{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newdrakkansftpgo)
}

func (r *drakkansftpgo) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// drakkansftpgo is the resource implementation.
type drakkansftpgo struct {
	osaasContext *osaasclient.Context
}

type drakkansftpgoModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
}

func (r *drakkansftpgo) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_drakkan_sftpgo_resource"
}

// Schema defines the schema for the resource.
func (r *drakkansftpgo) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *drakkansftpgo) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan drakkansftpgoModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("drakkan-sftpgo")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "drakkan-sftpgo", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "drakkan-sftpgo", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := drakkansftpgoModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *drakkansftpgo) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *drakkansftpgo) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *drakkansftpgo) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state drakkansftpgoModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("drakkan-sftpgo")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "drakkan-sftpgo", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
