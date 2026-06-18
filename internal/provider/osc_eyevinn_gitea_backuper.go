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
	_ resource.Resource              = &eyevinngiteabackuper{}
	_ resource.ResourceWithConfigure = &eyevinngiteabackuper{}
)

func Neweyevinngiteabackuper() resource.Resource {
	return &eyevinngiteabackuper{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinngiteabackuper)
}

func (r *eyevinngiteabackuper) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinngiteabackuper is the resource implementation.
type eyevinngiteabackuper struct {
	osaasContext *osaasclient.Context
}

type eyevinngiteabackuperModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Operation         types.String       `tfsdk:"operation"`
	Giteaurl         types.String       `tfsdk:"gitea_url"`
	Giteatoken         types.String       `tfsdk:"gitea_token"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	S3bucket         types.String       `tfsdk:"s3_bucket"`
	S3objectkey         types.String       `tfsdk:"s3_object_key"`
	S3accesskey         types.String       `tfsdk:"s3_access_key"`
	S3secretkey         types.String       `tfsdk:"s3_secret_key"`
	S3region         types.String       `tfsdk:"s3_region"`
	Encryptionkey         types.String       `tfsdk:"encryption_key"`
}

func (r *eyevinngiteabackuper) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_gitea_backuper"
}

// Schema defines the schema for the resource.
func (r *eyevinngiteabackuper) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Secure your Gitea instances effortlessly with gitea-backuper! Perform full Git mirror backups and restorations with encryption support on MinIO/S3-compatible storage. Efficient, reliable, and simple backup management!`,
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
				Description: "Name of gitea-backuper",
			},
			"operation": schema.StringAttribute{
				Required: true,
				Description: "Specifies the operation to perform on the Gitea instance",
			},
			"gitea_url": schema.StringAttribute{
				Required: true,
				Description: "The base URL of the Gitea instance to backup or restore",
			},
			"gitea_token": schema.StringAttribute{
				Required: true,
				Description: "Admin API token for authenticating with the Gitea instance",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "The endpoint URL for the MinIO or S3-compatible storage service",
			},
			"s3_bucket": schema.StringAttribute{
				Optional: true,
				Description: "The name of the S3/MinIO bucket where backups will be stored or retrieved from",
			},
			"s3_object_key": schema.StringAttribute{
				Optional: true,
				Description: "The specific object key (file path) within the S3 bucket for the backup archive",
			},
			"s3_access_key": schema.StringAttribute{
				Optional: true,
				Description: "The access key for authenticating with the S3/MinIO storage service",
			},
			"s3_secret_key": schema.StringAttribute{
				Optional: true,
				Description: "The secret key for authenticating with the S3/MinIO storage service",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "The AWS region for the S3 service",
			},
			"encryption_key": schema.StringAttribute{
				Optional: true,
				Description: "AES-256-CBC passphrase for encrypting or decrypting the backup archive",
			},
		},
	}
}

func (r *eyevinngiteabackuper) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinngiteabackuperModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-gitea-backuper")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-gitea-backuper", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"Operation": plan.Operation.ValueString(),
		"GiteaUrl": plan.Giteaurl.ValueString(),
		"GiteaToken": plan.Giteatoken.ValueString(),
		"S3Endpoint": plan.S3endpoint.ValueString(),
		"S3Bucket": plan.S3bucket.ValueString(),
		"S3ObjectKey": plan.S3objectkey.ValueString(),
		"S3AccessKey": plan.S3accesskey.ValueString(),
		"S3SecretKey": plan.S3secretkey.ValueString(),
		"S3Region": plan.S3region.ValueString(),
		"EncryptionKey": plan.Encryptionkey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-gitea-backuper", instance["name"].(string), serviceAccessToken)
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
	state := eyevinngiteabackuperModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-gitea-backuper"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Operation: plan.Operation,
		Giteaurl: plan.Giteaurl,
		Giteatoken: plan.Giteatoken,
		S3endpoint: plan.S3endpoint,
		S3bucket: plan.S3bucket,
		S3objectkey: plan.S3objectkey,
		S3accesskey: plan.S3accesskey,
		S3secretkey: plan.S3secretkey,
		S3region: plan.S3region,
		Encryptionkey: plan.Encryptionkey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinngiteabackuper) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinngiteabackuper) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinngiteabackuper) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinngiteabackuperModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-gitea-backuper")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-gitea-backuper", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
