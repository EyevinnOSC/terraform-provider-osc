package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &EncoreTransferInstanceResource{}
	_ resource.ResourceWithConfigure = &EncoreTransferInstanceResource{}
)

// NewEncoreTransferInstanceResource is a helper function to simplify the provider implementation.
func NewEncoreTransferInstanceResource() resource.Resource {
	return &EncoreTransferInstanceResource{}
}

func (r *EncoreTransferInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// EncoreTransferInstanceResource is the resource implementation.
type EncoreTransferInstanceResource struct {
	osaasContext *osaasclient.Context
}

type EncoreTransferInstanceResourceModel struct {
	Name		   types.String	`tfsdk:"name"`
	RedisUrl	   types.String	`tfsdk:"redis_url"`
	RedisQueue	   types.String	`tfsdk:"redis_queue"`
	Output         types.String `tfsdk:"output"` 
	AwsKeyId       types.String `tfsdk:"aws_keyid"` 
	AwsSecret      types.String `tfsdk:"aws_secret"` 
	OscToken       types.String `tfsdk:"osc_token"`
}

// Metadata returns the resource type name.
func (r *EncoreTransferInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encore_transfer_instance"
}

// Schema defines the schema for the resource.
func (r *EncoreTransferInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"redis_url": schema.StringAttribute{
				Required: true,
			},
			"redis_queue": schema.StringAttribute{
				Optional: true,
			},
			"output": schema.StringAttribute{
				Required: true,
			},
			"aws_keyid": schema.StringAttribute{
				Required: true,
			},
			"aws_secret": schema.StringAttribute{
				Required: true,
			},
			"osc_token": schema.StringAttribute{
				Required: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *EncoreTransferInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EncoreTransferInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-transfer")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	_, err = osaasclient.CreateInstance(r.osaasContext, "eyevinn-encore-transfer", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name.ValueString(),
		"RedisUrl":    plan.RedisUrl.ValueString(),
		"RedisQueue":  plan.RedisQueue.ValueString(),
		"Output":      plan.Output.ValueString(),
		"OscAccessToken": plan.OscToken.ValueString(),
		"AwsAccessKeyIdSecret": plan.AwsKeyId.ValueString(),
		"AwsSecretAccessKeySecret": plan.AwsSecret.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create valkey instance", err.Error())
		return
	}

	// Update the state with the actual data returned from the API
	state := EncoreTransferInstanceResourceModel{
		Name:        plan.Name,
		RedisUrl:	 plan.RedisUrl,
		RedisQueue:  plan.RedisQueue,
		Output:      plan.Output,
		OscToken:    plan.OscToken,
		AwsKeyId:    plan.AwsKeyId,
		AwsSecret:   plan.AwsSecret,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *EncoreTransferInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *EncoreTransferInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *EncoreTransferInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EncoreTransferInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-transfer")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}
	fmt.Println("state: ", state)

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-encore-transfer", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete valkey instance", err.Error())
		return
	}
}
