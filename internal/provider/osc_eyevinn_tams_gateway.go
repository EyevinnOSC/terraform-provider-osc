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
	_ resource.Resource              = &eyevinntamsgateway{}
	_ resource.ResourceWithConfigure = &eyevinntamsgateway{}
)

func Neweyevinntamsgateway() resource.Resource {
	return &eyevinntamsgateway{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinntamsgateway)
}

func (r *eyevinntamsgateway) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinntamsgateway is the resource implementation.
type eyevinntamsgateway struct {
	osaasContext *osaasclient.Context
}

type eyevinntamsgatewayModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dburl         types.String       `tfsdk:"db_url"`
	Dbusername         types.String       `tfsdk:"db_username"`
	Dbpassword         types.String       `tfsdk:"db_password"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	S3bucket         types.String       `tfsdk:"s3_bucket"`
	S3endpointurl         types.String       `tfsdk:"s3_endpoint_url"`
	Awsregion         types.String       `tfsdk:"aws_region"`
	Corsorigin         types.String       `tfsdk:"cors_origin"`
	Loglevel         types.String       `tfsdk:"log_level"`
}

func (r *eyevinntamsgateway) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_tams_gateway"
}

// Schema defines the schema for the resource.
func (r *eyevinntamsgateway) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Revolutionize your media management with TAMS Gateway—effortlessly store and index segmented media flows. Enhance efficiency and access powerfully with an integrated database and flexible service support.`,
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
				Description: "Name of tams-gateway",
			},
			"db_url": schema.StringAttribute{
				Required: true,
				Description: "The URL connection string for the CouchDB database that stores the TAMS segment index and metadata",
			},
			"db_username": schema.StringAttribute{
				Required: true,
				Description: "The username for authenticating with the CouchDB database",
			},
			"db_password": schema.StringAttribute{
				Required: true,
				Description: "The password for authenticating with the CouchDB database",
			},
			"aws_access_key_id": schema.StringAttribute{
				Required: true,
				Description: "The access key ID for authenticating with the S3-compatible storage service",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Required: true,
				Description: "The secret access key for authenticating with the S3-compatible storage service",
			},
			"s3_bucket": schema.StringAttribute{
				Required: true,
				Description: "Configuration option for s3bucket",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional: true,
				Description: "The endpoint URL for the S3-compatible storage service where media segments are stored",
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for awsregion",
			},
			"cors_origin": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for corsorigin",
			},
			"log_level": schema.StringAttribute{
				Optional: true,
				Description: "Logging or debugging configuration",
			},
		},
	}
}

func (r *eyevinntamsgateway) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinntamsgatewayModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-tams-gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-tams-gateway", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbUrl": plan.Dburl.ValueString(),
		"DbUsername": plan.Dbusername.ValueString(),
		"DbPassword": plan.Dbpassword.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"S3Bucket": plan.S3bucket.ValueString(),
		"S3EndpointUrl": plan.S3endpointurl.ValueString(),
		"AwsRegion": plan.Awsregion.ValueString(),
		"CorsOrigin": plan.Corsorigin.ValueString(),
		"LogLevel": plan.Loglevel.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-tams-gateway", instance["name"].(string), serviceAccessToken)
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
	state := eyevinntamsgatewayModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-tams-gateway"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dburl: plan.Dburl,
		Dbusername: plan.Dbusername,
		Dbpassword: plan.Dbpassword,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		S3bucket: plan.S3bucket,
		S3endpointurl: plan.S3endpointurl,
		Awsregion: plan.Awsregion,
		Corsorigin: plan.Corsorigin,
		Loglevel: plan.Loglevel,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinntamsgateway) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinntamsgateway) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinntamsgateway) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinntamsgatewayModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-tams-gateway")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-tams-gateway", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
