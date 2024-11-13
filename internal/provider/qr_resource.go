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
	_ resource.Resource              = &QrGeneratorResource{}
	_ resource.ResourceWithConfigure = &QrGeneratorResource{}
)

func NewQrGeneratorResource() resource.Resource {
	return &QrGeneratorResource{}
}

func init() {
	RegisteredResources = append(RegisteredResources, NewQrGeneratorResource)
}

func (r *QrGeneratorResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// QrGeneratorResource is the resource implementation.
type QrGeneratorResource struct {
	osaasContext *osaasclient.Context
}

type QrGeneratorResourceModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	GotoUrl         types.String       `tfsdk:"goto_url"`
}

func (r *QrGeneratorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_qr_generator_resource"
}

// Schema defines the schema for the resource.
func (r *QrGeneratorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"goto_url": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

func (r *QrGeneratorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan QrGeneratorResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-qr-generator")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-qr-generator", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"GotoUrl": plan.GotoUrl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-qr-generator", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := QrGeneratorResourceModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		GotoUrl: plan.GotoUrl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *QrGeneratorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *QrGeneratorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *QrGeneratorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state QrGeneratorResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-qr-generator")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-qr-generator", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
