
package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/eyevinn/osaas-client-go"
)


var (
	_ resource.Resource              = &TestsrcHlsLiveInstanceResource{}
	_ resource.ResourceWithConfigure = &TestsrcHlsLiveInstanceResource{}
)

func NewTestsrcHlsLiveInstanceResource() resource.Resource {
	return &TestsrcHlsLiveInstanceResource{}
}

func (r *TestsrcHlsLiveInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type TestsrcHlsLiveInstanceResource struct {
	osaasContext *osaasclient.Context
}

type TestsrcHlsLiveInstanceResourceModel struct {
	Name		   types.String	`tfsdk:"name"`
	Url		       types.String	`tfsdk:"url"`
}

func (r *TestsrcHlsLiveInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_testsrc_hls_live_instance"
}

func (r *TestsrcHlsLiveInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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

func (r *TestsrcHlsLiveInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan TestsrcHlsLiveInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
    
	serviceId := "eyevinn-docker-testsrc-hls-live"
	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken(serviceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, serviceId, serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// Update the state with the actual data returned from the API
	state := TestsrcHlsLiveInstanceResourceModel{
		Name:        types.StringValue(instance["name"].(string)),
		Url:         types.StringValue(instance["url"].(string)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *TestsrcHlsLiveInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *TestsrcHlsLiveInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *TestsrcHlsLiveInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state TestsrcHlsLiveInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := "eyevinn-docker-testsrc-hls-live"
	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken(serviceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, serviceId, state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
