package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &ValkeyInstanceResource{}
	_ resource.ResourceWithConfigure = &ValkeyInstanceResource{}
)

// NewValkeyInstanceResource is a helper function to simplify the provider implementation.
func NewValkeyInstanceResource() resource.Resource {
	return &ValkeyInstanceResource{}
}

func (r *ValkeyInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ValkeyInstanceResource is the resource implementation.
type ValkeyInstanceResource struct {
	osaasContext *osaasclient.Context
}

type ValkeyInstanceResourceModel struct {
	Name         string       `tfsdk:"name"`
	ExternalIP   types.String `tfsdk:"external_ip"`
	ExternalPort types.Int32  `tfsdk:"external_port"`
}

// Metadata returns the resource type name.
func (r *ValkeyInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_valkey_instance"
}

// Schema defines the schema for the resource.
func (r *ValkeyInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"external_ip": schema.StringAttribute{
				Computed: true,
			},
			"external_port": schema.Int32Attribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *ValkeyInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ValkeyInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("valkey-io-valkey")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	_, err = osaasclient.CreateInstance(r.osaasContext, "valkey-io-valkey", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create valkey instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "valkey-io-valkey", plan.Name, serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
		return
	}

	if len(ports) == 0 {
		resp.Diagnostics.AddError("Failed to get ports for service", "Ports list empty")
		return
	}

	port := ports[0]

	// Update the state with the actual data returned from the API
	state := ValkeyInstanceResourceModel{
		Name:        plan.Name,
		ExternalIP: types.StringValue(port.ExternalIP),
		ExternalPort: types.Int32Value(int32(port.ExternalPort)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//r.client.CreateValkeyInstance(plan.Name, plan.ProfilesUrl)

}

// Read refreshes the Terraform state with the latest data.
func (r *ValkeyInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ValkeyInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ValkeyInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ValkeyInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("valkey-io-valkey")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}
	fmt.Println("state: ", state)

	err = osaasclient.RemoveInstance(r.osaasContext, "valkey-io-valkey", state.Name, serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete valkey instance", err.Error())
		return
	}
}
