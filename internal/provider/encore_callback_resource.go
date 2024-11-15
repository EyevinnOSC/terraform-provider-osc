package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/eyevinn/osaas-client-go"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource              = &EncoreCallbackListenerInstanceResource{}
	_ resource.ResourceWithConfigure = &EncoreCallbackListenerInstanceResource{}
)

// NewEncoreCallbackListenerInstanceResource is a helper function to simplify the provider implementation.
func NewEncoreCallbackListenerInstanceResource() resource.Resource {
	return &EncoreCallbackListenerInstanceResource{}
}

func (r *EncoreCallbackListenerInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// EncoreCallbackListenerInstanceResource is the resource implementation.
type EncoreCallbackListenerInstanceResource struct {
	osaasContext *osaasclient.Context
}

type EncoreCallbackListenerInstanceResourceModel struct {
	Name		string	        `tfsdk:"name"`
	Url		    types.String	`tfsdk:"url"`
	RedisUrl	string       	`tfsdk:"redis_url"`
	EncoreUrl	string       	`tfsdk:"encore_url"`
	RedisQueue	string       	`tfsdk:"redis_queue"`
}

// Metadata returns the resource type name.
func (r *EncoreCallbackListenerInstanceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_encore_callback_instance"
}

// Schema defines the schema for the resource.
func (r *EncoreCallbackListenerInstanceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"redis_url": schema.StringAttribute{
				Required: true,
			},
			"encore_url": schema.StringAttribute{
				Required: true,
			},
			"redis_queue": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *EncoreCallbackListenerInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan EncoreCallbackListenerInstanceResourceModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-callback-listener")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-encore-callback-listener", serviceAccessToken, map[string]interface{}{
		"name":        plan.Name,
		"RedisUrl":    plan.RedisUrl,
		"EncoreUrl":   plan.EncoreUrl,
		"RedisQueue":  plan.RedisQueue,

	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create valkey instance", err.Error())
		return
	}

	// Update the state with the actual data returned from the API
	state := EncoreCallbackListenerInstanceResourceModel{
		Name:        plan.Name,
		Url:        types.StringValue(instance["url"].(string)),
		RedisUrl:	plan.RedisUrl,
		EncoreUrl: plan.EncoreUrl,
		RedisQueue: plan.RedisQueue,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *EncoreCallbackListenerInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *EncoreCallbackListenerInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *EncoreCallbackListenerInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state EncoreCallbackListenerInstanceResourceModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-callback-listener")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}
	fmt.Println("state: ", state)

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-encore-callback-listener", state.Name, serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete valkey instance", err.Error())
		return
	}
}
