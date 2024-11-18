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
	_ resource.Resource              = &eyevinnencorecallbacklistener{}
	_ resource.ResourceWithConfigure = &eyevinnencorecallbacklistener{}
)

func Neweyevinnencorecallbacklistener() resource.Resource {
	return &eyevinnencorecallbacklistener{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnencorecallbacklistener)
}

func (r *eyevinnencorecallbacklistener) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnencorecallbacklistener is the resource implementation.
type eyevinnencorecallbacklistener struct {
	osaasContext *osaasclient.Context
}

type eyevinnencorecallbacklistenerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Encoreurl         types.String       `tfsdk:"encore_url"`
	Redisqueue         types.String       `tfsdk:"redis_queue"`
}

func (r *eyevinnencorecallbacklistener) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_encore_callback_listener_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinnencorecallbacklistener) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Encore callback listener is a powerful HTTP server that listens for successful job callbacks, posting jobId and Url on a redis queue. Fully customizable with environment variables. Enhance your project efficiency now! Contact sales@eyevinn.se for further details.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of encore-callback-listener",
			},
			"redis_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"encore_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"redis_queue": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnencorecallbacklistener) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnencorecallbacklistenerModel
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
		"name": plan.Name.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"EncoreUrl": plan.Encoreurl.ValueString(),
		"RedisQueue": plan.Redisqueue.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-encore-callback-listener", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinnencorecallbacklistenerModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Redisurl: plan.Redisurl,
		Encoreurl: plan.Encoreurl,
		Redisqueue: plan.Redisqueue,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnencorecallbacklistener) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnencorecallbacklistener) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnencorecallbacklistener) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnencorecallbacklistenerModel
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

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-encore-callback-listener", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
