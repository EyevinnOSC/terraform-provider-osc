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
	_ resource.Resource              = &SgaiAdProxyInstanceResource{}
	_ resource.ResourceWithConfigure = &SgaiAdProxyInstanceResource{}
)

func NewSgaiAdProxyInstanceResource() resource.Resource {
	return &SgaiAdProxyInstanceResource{}
}

func (r *SgaiAdProxyInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

type SgaiAdProxyInstanceResource struct {
	osaasContext *osaasclient.Context
}

type SgaiAdProxyInstanceResourceModel struct {
	Name		   types.String	`tfsdk:"name"`
	Url		       types.String	`tfsdk:"url"`
	VastEndpoint   types.String `tfsdk:"vast_endpoint"`
	OriginUrl      types.String `tfsdk:"origin_url"`
	InsertionMode  types.String `tfsdk:"insertion_mode"`
}

func (r *SgaiAdProxyInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sgai_ad_proxy_instance"
}

func (r *SgaiAdProxyInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"insertion_mode": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

type InsertionMode int
const (
	Static InsertionMode = iota
	Dynamic
)
func (r *SgaiAdProxyInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SgaiAdProxyInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := "eyevinn-sgai-ad-proxy"
	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken(serviceId)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}


	insertionMode := 0 
	if plan.InsertionMode.ValueString() == "dynamic" {
		insertionMode = 1
	}
	instance, err := osaasclient.CreateInstance(r.osaasContext, serviceId, serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"VastEndpoint": plan.VastEndpoint.ValueString(),
		"OriginUrl": plan.OriginUrl.ValueString(),
		"InsertionMode": insertionMode,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// Update the state with the actual data returned from the API
	state := SgaiAdProxyInstanceResourceModel{
		Name:          types.StringValue(instance["name"].(string)),
		Url:           types.StringValue(instance["url"].(string)),
		VastEndpoint:  plan.VastEndpoint, 
		OriginUrl:     plan.OriginUrl,
		InsertionMode: plan.InsertionMode,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func (r *SgaiAdProxyInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

func (r *SgaiAdProxyInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

func (r *SgaiAdProxyInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state SgaiAdProxyInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceId := "eyevinn-sgai-ad-proxy"
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
