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
	_ resource.Resource              = &eyevinncontinuewatchingapi{}
	_ resource.ResourceWithConfigure = &eyevinncontinuewatchingapi{}
)

func Neweyevinncontinuewatchingapi() resource.Resource {
	return &eyevinncontinuewatchingapi{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinncontinuewatchingapi)
}

func (r *eyevinncontinuewatchingapi) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinncontinuewatchingapi is the resource implementation.
type eyevinncontinuewatchingapi struct {
	osaasContext *osaasclient.Context
}

type eyevinncontinuewatchingapiModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Redishost         types.String       `tfsdk:"redis_host"`
	Redisport         types.String       `tfsdk:"redis_port"`
	Redisusername         types.String       `tfsdk:"redis_username"`
	Redispassword         types.String       `tfsdk:"redis_password"`
}

func (r *eyevinncontinuewatchingapi) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_continue_watching_api_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinncontinuewatchingapi) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `A user of a streaming service expects that they can pick up where they left on any of their devices. To handle that you would need to develop a service with endpoints for the application to write and read from. This open source cloud component take care of that and all you need is to have a Redis database running on Redis Cloud for example.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of continue-watching-api",
			},
			"redis_host": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"redis_port": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"redis_username": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"redis_password": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinncontinuewatchingapi) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinncontinuewatchingapiModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-continue-watching-api")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-continue-watching-api", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"RedisHost": plan.Redishost.ValueString(),
		"RedisPort": plan.Redisport.ValueString(),
		"RedisUsername": plan.Redisusername.ValueString(),
		"RedisPassword": plan.Redispassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-continue-watching-api", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinncontinuewatchingapiModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Redishost: plan.Redishost,
		Redisport: plan.Redisport,
		Redisusername: plan.Redisusername,
		Redispassword: plan.Redispassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinncontinuewatchingapi) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinncontinuewatchingapi) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinncontinuewatchingapi) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinncontinuewatchingapiModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-continue-watching-api")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-continue-watching-api", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
