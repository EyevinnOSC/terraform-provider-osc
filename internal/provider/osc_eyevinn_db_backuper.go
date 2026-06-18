package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource              = &eyevinndbbackuper{}
	_ resource.ResourceWithConfigure = &eyevinndbbackuper{}
)

func Neweyevinndbbackuper() resource.Resource {
	return &eyevinndbbackuper{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinndbbackuper)
}

func (r *eyevinndbbackuper) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinndbbackuper is the resource implementation.
type eyevinndbbackuper struct {
	osaasContext *osaasclient.Context
}

type eyevinndbbackuperModel struct {
	InstanceUrl   types.String `tfsdk:"instance_url"`
	ServiceId     types.String `tfsdk:"service_id"`
	ExternalIp    types.String `tfsdk:"external_ip"`
	ExternalPort  types.Int32  `tfsdk:"external_port"`
	Name          types.String `tfsdk:"name"`
	Operation     types.String `tfsdk:"operation"`
	Databaseurl   types.String `tfsdk:"database_url"`
	S3endpoint    types.String `tfsdk:"s3_endpoint"`
	S3bucket      types.String `tfsdk:"s3_bucket"`
	S3objectkey   types.String `tfsdk:"s3_object_key"`
	S3accesskey   types.String `tfsdk:"s3_access_key"`
	S3secretkey   types.String `tfsdk:"s3_secret_key"`
	Encryptionkey types.String `tfsdk:"encryption_key"`
}

func (r *eyevinndbbackuper) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_db_backuper"
}

// Schema defines the schema for the resource.
func (r *eyevinndbbackuper) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your data management with db-backuper—an all-encompassing solution supporting PostgreSQL, MariaDB, Redis, ClickHouse, and CouchDB. Secure backups to S3 with optional AES-256 encryption effortlessly!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed:    true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed:    true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of db-backuper",
			},
			"operation": schema.StringAttribute{
				Required:    true,
				Description: "Specifies the operation to perform - either &#39;backup&#39; to create a database backup or &#39;restore&#39; to restore from a backup",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "Connection URL for the database to backup or restore. The URL scheme determines which database type and tools are used",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional:    true,
				Description: "The endpoint URL for S3-compatible storage where backups will be stored or retrieved from",
			},
			"s3_bucket": schema.StringAttribute{
				Optional:    true,
				Description: "The name of the S3 bucket where backup files will be stored or retrieved from",
			},
			"s3_object_key": schema.StringAttribute{
				Optional:    true,
				Description: "The S3 object key (path within the bucket) for the backup file",
			},
			"s3_access_key": schema.StringAttribute{
				Optional:    true,
				Description: "The access key for authenticating with S3-compatible storage",
			},
			"s3_secret_key": schema.StringAttribute{
				Optional:    true,
				Description: "The secret key for authenticating with S3-compatible storage",
			},
			"encryption_key": schema.StringAttribute{
				Optional:    true,
				Description: "Optional AES-256-CBC encryption key for encrypting backups before upload and decrypting during restore",
			},
		},
	}
}

func (r *eyevinndbbackuper) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinndbbackuperModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-db-backuper")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-db-backuper", serviceAccessToken, map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"Operation":     plan.Operation.ValueString(),
		"DatabaseUrl":   plan.Databaseurl.ValueString(),
		"S3Endpoint":    plan.S3endpoint.ValueString(),
		"S3Bucket":      plan.S3bucket.ValueString(),
		"S3ObjectKey":   plan.S3objectkey.ValueString(),
		"S3AccessKey":   plan.S3accesskey.ValueString(),
		"S3SecretKey":   plan.S3secretkey.ValueString(),
		"EncryptionKey": plan.Encryptionkey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-db-backuper", instance["name"].(string), serviceAccessToken)
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
	state := eyevinndbbackuperModel{
		InstanceUrl:   types.StringValue(instance["url"].(string)),
		ServiceId:     types.StringValue("eyevinn-db-backuper"),
		ExternalIp:    types.StringValue(externalIp),
		ExternalPort:  types.Int32Value(int32(externalPort)),
		Name:          plan.Name,
		Operation:     plan.Operation,
		Databaseurl:   plan.Databaseurl,
		S3endpoint:    plan.S3endpoint,
		S3bucket:      plan.S3bucket,
		S3objectkey:   plan.S3objectkey,
		S3accesskey:   plan.S3accesskey,
		S3secretkey:   plan.S3secretkey,
		Encryptionkey: plan.Encryptionkey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinndbbackuper) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinndbbackuper) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinndbbackuper) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinndbbackuperModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-db-backuper")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-db-backuper", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
