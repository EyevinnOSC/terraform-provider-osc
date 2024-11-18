package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &EncoreInstanceResource{}
	_ resource.ResourceWithConfigure = &EncoreInstanceResource{}
)

// NewEncoreInstanceResource is a helper function to simplify the provider implementation.
func NewEncoreInstanceResource() resource.Resource {
	return &EncoreInstanceResource{}
}

func (r *EncoreInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// EncoreInstanceResource is the resource implementation.
type EncoreInstanceResource struct {
	osaasContext *osaasclient.Context
}

type EncoreInstanceResourceModel struct {
	Name        string       `tfsdk:"name"`
	ProfilesUrl string       `tfsdk:"profiles_url"`
	Url         types.String `tfsdk:"url"`
}

// Metadata returns the resource type name.
func (r *EncoreInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encore_instance"
}

// Schema defines the schema for the resource.
func (r *EncoreInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"profiles_url": schema.StringAttribute{
				Optional: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *EncoreInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EncoreInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("encore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "encore", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name,
		"profilesUrl": plan.ProfilesUrl,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create encore instance", err.Error())
		return
	}
	// Update the state with the actual data returned from the API

	state := EncoreInstanceResourceModel{
		Name:        instance["name"].(string),
		ProfilesUrl: instance["profilesUrl"].(string),
		Url:         types.StringValue(instance["url"].(string)),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	//r.client.CreateEncoreInstance(plan.Name, plan.ProfilesUrl)

}

// Read refreshes the Terraform state with the latest data.
func (r *EncoreInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *EncoreInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *EncoreInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EncoreInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("encore")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}
	fmt.Println("state: ", state)

	err = osaasclient.RemoveInstance(r.osaasContext, "encore", state.Name, serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete encore instance", err.Error())
		return
	}
}
