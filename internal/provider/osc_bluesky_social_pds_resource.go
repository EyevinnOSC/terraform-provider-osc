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
	_ resource.Resource              = &blueskysocialpds{}
	_ resource.ResourceWithConfigure = &blueskysocialpds{}
)

func Newblueskysocialpds() resource.Resource {
	return &blueskysocialpds{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newblueskysocialpds)
}

func (r *blueskysocialpds) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// blueskysocialpds is the resource implementation.
type blueskysocialpds struct {
	osaasContext *osaasclient.Context
}

type blueskysocialpdsModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Adminpassword         types.String       `tfsdk:"admin_password"`
	Dnsname         types.String       `tfsdk:"dns_name"`
}

func (r *blueskysocialpds) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_bluesky_social_pds_resource"
}

// Schema defines the schema for the resource.
func (r *blueskysocialpds) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Empower your network with self-hosted Bluesky PDS! Harness the power of AT Protocol to easily manage your data server. Seamless installation, full control, and enhanced security for your social media presence.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of pds",
			},
			"admin_password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"dns_name": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *blueskysocialpds) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan blueskysocialpdsModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("bluesky-social-pds")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "bluesky-social-pds", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
		"DnsName": plan.Dnsname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "bluesky-social-pds", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := blueskysocialpdsModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Adminpassword: plan.Adminpassword,
		Dnsname: plan.Dnsname,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *blueskysocialpds) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *blueskysocialpds) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *blueskysocialpds) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state blueskysocialpdsModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("bluesky-social-pds")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "bluesky-social-pds", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
