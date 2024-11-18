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
	_ resource.Resource              = &alexbj75movierecommendator{}
	_ resource.ResourceWithConfigure = &alexbj75movierecommendator{}
)

func Newalexbj75movierecommendator() resource.Resource {
	return &alexbj75movierecommendator{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newalexbj75movierecommendator)
}

func (r *alexbj75movierecommendator) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// alexbj75movierecommendator is the resource implementation.
type alexbj75movierecommendator struct {
	osaasContext *osaasclient.Context
}

type alexbj75movierecommendatorModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Openaikey         types.String       `tfsdk:"open_ai_key"`
}

func (r *alexbj75movierecommendator) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_alexbj75_movierecommendator_resource"
}

// Schema defines the schema for the resource.
func (r *alexbj75movierecommendator) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Discover new films effortlessly! Enter a movie name and get two personalized recommendations powered by OpenAI. Transform your movie nights with Movie Recommender’s smart suggestions. Try it now!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of movierecommendator",
			},
			"open_ai_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *alexbj75movierecommendator) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alexbj75movierecommendatorModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-movierecommendator")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "alexbj75-movierecommendator", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"OpenAiKey": plan.Openaikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "alexbj75-movierecommendator", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := alexbj75movierecommendatorModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Openaikey: plan.Openaikey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *alexbj75movierecommendator) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *alexbj75movierecommendator) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *alexbj75movierecommendator) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alexbj75movierecommendatorModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-movierecommendator")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "alexbj75-movierecommendator", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
