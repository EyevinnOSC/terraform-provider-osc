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
	_ resource.Resource              = &miniominio{}
	_ resource.ResourceWithConfigure = &miniominio{}
)

func Newminiominio() resource.Resource {
	return &miniominio{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newminiominio)
}

func (r *miniominio) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// miniominio is the resource implementation.
type miniominio struct {
	osaasContext *osaasclient.Context
}

type miniominioModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Rootuser         types.String       `tfsdk:"root_user"`
	Rootpassword         types.String       `tfsdk:"root_password"`
}

func (r *miniominio) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_minio_minio_resource"
}

// Schema defines the schema for the resource.
func (r *miniominio) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `MinIO is the High Performance Object Storage solution you&#39;ve been searching for! API compatible with Amazon S3, it&#39;s perfect for machine learning, analytics, and app data workloads. Easy container installation with stable podman run commands. Mac, Linux, Windows support available for simple standalone server setup. Explore further with MinIO SDKs and contribute to the MinIO Project. Get your MinIO now and revolutionize your storage game!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of minio",
			},
			"root_user": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"root_password": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *miniominio) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan miniominioModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("minio-minio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "minio-minio", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"RootUser": plan.Rootuser.ValueString(),
		"RootPassword": plan.Rootpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "minio-minio", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := miniominioModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Rootuser: plan.Rootuser,
		Rootpassword: plan.Rootpassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *miniominio) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *miniominio) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *miniominio) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state miniominioModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("minio-minio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "minio-minio", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
