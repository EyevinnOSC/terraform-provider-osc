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
	_ resource.Resource              = &formbricksformbricks{}
	_ resource.ResourceWithConfigure = &formbricksformbricks{}
)

func Newformbricksformbricks() resource.Resource {
	return &formbricksformbricks{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newformbricksformbricks)
}

func (r *formbricksformbricks) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// formbricksformbricks is the resource implementation.
type formbricksformbricks struct {
	osaasContext *osaasclient.Context
}

type formbricksformbricksModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Databaseurl         types.String       `tfsdk:"database_url"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	S3bucketname         types.String       `tfsdk:"s3_bucket_name"`
	S3endpointurl         types.String       `tfsdk:"s3_endpoint_url"`
}

func (r *formbricksformbricks) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_formbricks_formbricks"
}

// Schema defines the schema for the resource.
func (r *formbricksformbricks) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Revolutionize user engagement with Formbricks, the open-source Qualtrics alternative! Create conversion-optimized, privacy-first surveys effortlessly. Empower your team with transformational insights today!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed: true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed: true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed: true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of formbricks",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_bucket_name": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *formbricksformbricks) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan formbricksformbricksModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("formbricks-formbricks")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "formbricks-formbricks", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"AwsRegion": plan.Awsregion.ValueString(),
		"S3BucketName": plan.S3bucketname.ValueString(),
		"S3EndpointUrl": plan.S3endpointurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "formbricks-formbricks", instance["name"].(string), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
		return
	}

	var externalPort = 0
	var externalIp = ""
	if len(ports) > 0 {
		port := ports[0]
		externalPort = port.ExternalPort
		externalIp = port.ExternalIP
	}


	// Update the state with the actual data returned from the API
	state := formbricksformbricksModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("formbricks-formbricks"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Databaseurl: plan.Databaseurl,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Awsregion: plan.Awsregion,
		S3bucketname: plan.S3bucketname,
		S3endpointurl: plan.S3endpointurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *formbricksformbricks) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *formbricksformbricks) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *formbricksformbricks) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state formbricksformbricksModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("formbricks-formbricks")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "formbricks-formbricks", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
