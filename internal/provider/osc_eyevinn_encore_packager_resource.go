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
	_ resource.Resource              = &eyevinnencorepackager{}
	_ resource.ResourceWithConfigure = &eyevinnencorepackager{}
)

func Neweyevinnencorepackager() resource.Resource {
	return &eyevinnencorepackager{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnencorepackager)
}

func (r *eyevinnencorepackager) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnencorepackager is the resource implementation.
type eyevinnencorepackager struct {
	osaasContext *osaasclient.Context
}

type eyevinnencorepackagerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Redisqueue         types.String       `tfsdk:"redis_queue"`
	Outputfolder         types.String       `tfsdk:"output_folder"`
	Concurrency         types.String       `tfsdk:"concurrency"`
	Personalaccesstoken         types.String       `tfsdk:"personal_access_token"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	Awssessiontoken         types.String       `tfsdk:"aws_session_token"`
}

func (r *eyevinnencorepackager) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_encore_packager_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinnencorepackager) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Enhance your transcoding workflow with Encore packager! Run as a service, listen for messages on redis queue, and customize packaging events. Boost productivity with this versatile tool.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of encore-packager",
			},
			"redis_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"redis_queue": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"output_folder": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"concurrency": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"personal_access_token": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_access_key_id": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_session_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnencorepackager) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnencorepackagerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-packager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-encore-packager", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"RedisQueue": plan.Redisqueue.ValueString(),
		"OutputFolder": plan.Outputfolder.ValueString(),
		"Concurrency": plan.Concurrency.ValueString(),
		"PersonalAccessToken": plan.Personalaccesstoken.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"AwsRegion": plan.Awsregion.ValueString(),
		"AwsSessionToken": plan.Awssessiontoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-encore-packager", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinnencorepackagerModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Redisurl: plan.Redisurl,
		Redisqueue: plan.Redisqueue,
		Outputfolder: plan.Outputfolder,
		Concurrency: plan.Concurrency,
		Personalaccesstoken: plan.Personalaccesstoken,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Awsregion: plan.Awsregion,
		Awssessiontoken: plan.Awssessiontoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnencorepackager) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnencorepackager) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnencorepackager) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnencorepackagerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-encore-packager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-encore-packager", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
