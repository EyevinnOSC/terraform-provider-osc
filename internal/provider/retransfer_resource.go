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
	_ resource.Resource              = &RetransferResource{}
	_ resource.ResourceWithConfigure = &RetransferResource{}
)

// NewRetransferResource is a helper function to simplify the provider implementation.
func NewRetransferResource() resource.Resource {
	return &RetransferResource{}
}

func (r *RetransferResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// RetransferResource is the resource implementation.
type RetransferResource struct {
	osaasContext *osaasclient.Context
}

type RetransferResourceModel struct {
	AwsKeyIdName	types.String `tfsdk:"aws_keyid_name"` 
	AwsSecretName	types.String `tfsdk:"aws_secret_name"` 
	AwsKeyIdValue	types.String `tfsdk:"aws_keyid_value"` 
	AwsSecretValue	types.String `tfsdk:"aws_secret_value"` 
}

// Metadata returns the resource type name.
func (r *RetransferResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_retransfer"
}

// Schema defines the schema for the resource.
func (r *RetransferResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"aws_keyid_name": schema.StringAttribute{
				Optional: true,
			},
			"aws_secret_name": schema.StringAttribute{
				Optional: true,
			},
			"aws_keyid_value": schema.StringAttribute{
				Optional: true,
			},
			"aws_secret_value": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}


// Create creates the resource and sets the initial Terraform state.
func (r *RetransferResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan RetransferResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}


	osaasclient.AddServiceSecret(r.osaasContext, "eyevinn-docker-retransfer", plan.AwsKeyIdName.ValueString(), plan.AwsKeyIdValue.ValueString())
	osaasclient.AddServiceSecret(r.osaasContext, "eyevinn-docker-retransfer", plan.AwsSecretName.ValueString(), plan.AwsSecretValue.ValueString())

	state := RetransferResourceModel{
		AwsKeyIdName:	plan.AwsKeyIdName,
		AwsSecretName:	plan.AwsSecretName,
		AwsKeyIdValue:	plan.AwsKeyIdValue,
		AwsSecretValue:	plan.AwsSecretValue,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *RetransferResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *RetransferResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *RetransferResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}
