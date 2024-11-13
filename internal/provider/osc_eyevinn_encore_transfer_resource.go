package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/eyevinn/osaas-client-go"
)

var (
	_ resource.Resource              = &eyevinnencoretransfer{}
	_ resource.ResourceWithConfigure = &eyevinnencoretransfer{}
)

func Neweyevinnencoretransfer() resource.Resource {
	return &eyevinnencoretransfer{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnencoretransfer)
}

func (r *eyevinnencoretransfer) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnencoretransfer is the resource implementation.
type eyevinnencoretransfer struct {
	osaasContext *osaasclient.Context
}

type eyevinnencoretransferModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Redisqueue         types.String       `tfsdk:"redis_queue"`
	Output         types.String       `tfsdk:"output"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
	Awsaccesskeyidsecret         types.String       `tfsdk:"aws_access_key_id_secret"`
	Awssecretaccesskeysecret         types.String       `tfsdk:"aws_secret_access_key_secret"`
}

func (r *eyevinnencoretransfer) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_encore_transfer_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinnencoretransfer) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			"redis_queue": schema.StringAttribute{
				Optional: true,
			},
			"output": schema.StringAttribute{
				Required: true,
			},
			"osc_access_token": schema.StringAttribute{
				Required: true,
			},
			"aws_access_key_id_secret": schema.StringAttribute{
				Optional: true,
			},
			"aws_secret_access_key_secret": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *eyevinnencoretransfer) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnencoretransferModel
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

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-encore-transfer", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"RedisQueue": plan.Redisqueue.ValueString(),
		"Output": plan.Output.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
		"AwsAccessKeyIdSecret": plan.Awsaccesskeyidsecret.ValueString(),
		"AwsSecretAccessKeySecret": plan.Awssecretaccesskeysecret.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-encore-transfer", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinnencoretransferModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Redisurl: plan.Redisurl,
		Redisqueue: plan.Redisqueue,
		Output: plan.Output,
		Oscaccesstoken: plan.Oscaccesstoken,
		Awsaccesskeyidsecret: plan.Awsaccesskeyidsecret,
		Awssecretaccesskeysecret: plan.Awssecretaccesskeysecret,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnencoretransfer) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnencoretransfer) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnencoretransfer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnencoretransferModel
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

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-encore-transfer", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
