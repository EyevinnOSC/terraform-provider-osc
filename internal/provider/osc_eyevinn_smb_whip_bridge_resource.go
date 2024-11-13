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
	_ resource.Resource              = &eyevinnsmbwhipbridge{}
	_ resource.ResourceWithConfigure = &eyevinnsmbwhipbridge{}
)

func Neweyevinnsmbwhipbridge() resource.Resource {
	return &eyevinnsmbwhipbridge{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnsmbwhipbridge)
}

func (r *eyevinnsmbwhipbridge) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnsmbwhipbridge is the resource implementation.
type eyevinnsmbwhipbridge struct {
	osaasContext *osaasclient.Context
}

type eyevinnsmbwhipbridgeModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Smburl         types.String       `tfsdk:"smb_url"`
	Smbapikey         types.String       `tfsdk:"smb_api_key"`
	Whependpointurl         types.String       `tfsdk:"whep_endpoint_url"`
	Whipapikey         types.String       `tfsdk:"whip_api_key"`
}

func (r *eyevinnsmbwhipbridge) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_smb_whip_bridge_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinnsmbwhipbridge) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"smb_url": schema.StringAttribute{
				Required: true,
			},
			"smb_api_key": schema.StringAttribute{
				Optional: true,
			},
			"whep_endpoint_url": schema.StringAttribute{
				Optional: true,
			},
			"whip_api_key": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *eyevinnsmbwhipbridge) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnsmbwhipbridgeModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-smb-whip-bridge")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-smb-whip-bridge", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SmbUrl": plan.Smburl.ValueString(),
		"SmbApiKey": plan.Smbapikey.ValueString(),
		"WhepEndpointUrl": plan.Whependpointurl.ValueString(),
		"WhipApiKey": plan.Whipapikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-smb-whip-bridge", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinnsmbwhipbridgeModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Smburl: plan.Smburl,
		Smbapikey: plan.Smbapikey,
		Whependpointurl: plan.Whependpointurl,
		Whipapikey: plan.Whipapikey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnsmbwhipbridge) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnsmbwhipbridge) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnsmbwhipbridge) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnsmbwhipbridgeModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-smb-whip-bridge")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-smb-whip-bridge", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
