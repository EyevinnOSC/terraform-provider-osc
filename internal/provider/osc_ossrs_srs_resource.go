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
	_ resource.Resource              = &ossrssrs{}
	_ resource.ResourceWithConfigure = &ossrssrs{}
)

func Newossrssrs() resource.Resource {
	return &ossrssrs{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newossrssrs)
}

func (r *ossrssrs) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ossrssrs is the resource implementation.
type ossrssrs struct {
	osaasContext *osaasclient.Context
}

type ossrssrsModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
}

func (r *ossrssrs) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_ossrs_srs_resource"
}

// Schema defines the schema for the resource.
func (r *ossrssrs) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Experience high-efficiency video streaming with SRS&#x2F;6.0. Stream seamlessly with essential features included. 
Transform your streaming experience now! Explore RTMP, HLS, HTTP-FLV, SRT, MPEG-DASH protocols, and more.
Get started easily!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of srs",
			},
		},
	}
}

func (r *ossrssrs) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ossrssrsModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ossrs-srs")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "ossrs-srs", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "ossrs-srs", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := ossrssrsModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *ossrssrs) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ossrssrs) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ossrssrs) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ossrssrsModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ossrs-srs")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "ossrs-srs", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}