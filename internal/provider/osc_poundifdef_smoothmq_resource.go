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
	_ resource.Resource              = &poundifdefsmoothmq{}
	_ resource.ResourceWithConfigure = &poundifdefsmoothmq{}
)

func Newpoundifdefsmoothmq() resource.Resource {
	return &poundifdefsmoothmq{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newpoundifdefsmoothmq)
}

func (r *poundifdefsmoothmq) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// poundifdefsmoothmq is the resource implementation.
type poundifdefsmoothmq struct {
	osaasContext *osaasclient.Context
}

type poundifdefsmoothmqModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Accesskey         types.String       `tfsdk:"access_key"`
	Secretkey         types.String       `tfsdk:"secret_key"`
}

func (r *poundifdefsmoothmq) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_poundifdef_smoothmq_resource"
}

// Schema defines the schema for the resource.
func (r *poundifdefsmoothmq) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"access_key": schema.StringAttribute{
				Optional: true,
			},
			"secret_key": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *poundifdefsmoothmq) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan poundifdefsmoothmqModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("poundifdef-smoothmq")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "poundifdef-smoothmq", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AccessKey": plan.Accesskey.ValueString(),
		"SecretKey": plan.Secretkey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "poundifdef-smoothmq", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := poundifdefsmoothmqModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Accesskey: plan.Accesskey,
		Secretkey: plan.Secretkey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *poundifdefsmoothmq) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *poundifdefsmoothmq) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *poundifdefsmoothmq) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state poundifdefsmoothmqModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("poundifdef-smoothmq")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "poundifdef-smoothmq", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
