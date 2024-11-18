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
	_ resource.Resource              = &SecretResource{}
	_ resource.ResourceWithConfigure = &SecretResource{}
)

// NewSecretResource is a helper function to simplify the provider implementation.
func NewSecretResource() resource.Resource {
	return &SecretResource{}
}

func (r *SecretResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// SecretResource is the resource implementation.
type SecretResource struct {
	osaasContext *osaasclient.Context
}

type SecretResourceModel struct {
	ServiceIds      []types.String  `tfsdk:"service_ids"` 
	SecretName		types.String    `tfsdk:"secret_name"` 
	SecretValue		types.String    `tfsdk:"secret_value"` 
	Ref				types.String	`tfsdk:"ref"`
}

// Metadata returns the resource type name.
func (r *SecretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_secret"
}

// Schema defines the schema for the resource.
func (r *SecretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"service_ids": schema.ListAttribute{
				ElementType: types.StringType,
				Required:    true,
			},
			"secret_name": schema.StringAttribute{
				Required: true,
			},
			"secret_value": schema.StringAttribute{
				Required: true,
			},
			"ref": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}


// Create creates the resource and sets the initial Terraform state.
func (r *SecretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan SecretResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}


	for _, serviceId := range plan.ServiceIds {
		osaasclient.AddServiceSecret(r.osaasContext, serviceId.ValueString(), plan.SecretName.ValueString(), plan.SecretValue.ValueString())
	}

	ref := fmt.Sprintf("{{secrets.%s}}", plan.SecretName)
	state := SecretResourceModel{
		ServiceIds:		plan.ServiceIds,
		SecretName:		plan.SecretName,
		SecretValue:	plan.SecretValue,
		Ref:			types.StringValue(ref),
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *SecretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *SecretResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *SecretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
