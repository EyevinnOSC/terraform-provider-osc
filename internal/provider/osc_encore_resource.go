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
	_ resource.Resource              = &encore{}
	_ resource.ResourceWithConfigure = &encore{}
)

func Newencore() resource.Resource {
	return &encore{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newencore)
}

func (r *encore) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// encore is the resource implementation.
type encore struct {
	osaasContext *osaasclient.Context
}

type encoreModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Profilesurl         types.String       `tfsdk:"profiles_url"`
	S3accesskeyid         types.String       `tfsdk:"s3_access_key_id"`
	S3secretaccesskey         types.String       `tfsdk:"s3_secret_access_key"`
	S3sessiontoken         types.String       `tfsdk:"s3_session_token"`
	S3region         types.String       `tfsdk:"s3_region"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
}

func (r *encore) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_encore_resource"
}

// Schema defines the schema for the resource.
func (r *encore) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `SVT Encore is an open-source video transcoding system for efficient cloud-based video processing. It offers scalable, automated transcoding to optimize video workflows for various platforms, supporting multiple formats and codecs. With a focus on cost-effectiveness and flexibility, Encore is ideal for broadcasters and content creators needing dynamic scaling and reliable performance in their video production and distribution processes.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of the Encore instance",
			},
			"profiles_url": schema.StringAttribute{
				Optional: true,
				Description: "URL pointing to list of transcoding profiles",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_session_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *encore) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan encoreModel
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
		"name": plan.Name.ValueString(),
		"profilesUrl": plan.Profilesurl.ValueString(),
		"s3AccessKeyId": plan.S3accesskeyid.ValueString(),
		"s3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"s3SessionToken": plan.S3sessiontoken.ValueString(),
		"s3Region": plan.S3region.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "encore", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := encoreModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Profilesurl: plan.Profilesurl,
		S3accesskeyid: plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		S3sessiontoken: plan.S3sessiontoken,
		S3region: plan.S3region,
		S3endpoint: plan.S3endpoint,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *encore) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *encore) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *encore) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state encoreModel
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

	err = osaasclient.RemoveInstance(r.osaasContext, "encore", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
