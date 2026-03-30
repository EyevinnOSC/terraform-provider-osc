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
	_ resource.Resource              = &tuomokuspxgc{}
	_ resource.ResourceWithConfigure = &tuomokuspxgc{}
)

func Newtuomokuspxgc() resource.Resource {
	return &tuomokuspxgc{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newtuomokuspxgc)
}

func (r *tuomokuspxgc) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// tuomokuspxgc is the resource implementation.
type tuomokuspxgc struct {
	osaasContext *osaasclient.Context
}

type tuomokuspxgcModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Username         types.String       `tfsdk:"username"`
	Password         types.String       `tfsdk:"password"`
	S3templatesurl         types.String       `tfsdk:"s3_templates_url"`
	S3projectsurl         types.String       `tfsdk:"s3_projects_url"`
	S3pluginsurl         types.String       `tfsdk:"s3_plugins_url"`
	S3mediaurl         types.String       `tfsdk:"s3_media_url"`
	S3endpointurl         types.String       `tfsdk:"s3_endpoint_url"`
	S3accesskeyid         types.String       `tfsdk:"s3_access_key_id"`
	S3secretaccesskey         types.String       `tfsdk:"s3_secret_access_key"`
	S3region         types.String       `tfsdk:"s3_region"`
}

func (r *tuomokuspxgc) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_tuomoku_spx_gc"
}

// Schema defines the schema for the resource.
func (r *tuomokuspxgc) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your live productions with SPX Graphics Controller! Seamlessly manage HTML graphics across platforms like OBS, vMix, and more. Ideal for stunning live streams and broadcasts with powerful customization.`,
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
				Description: "Name of spx-gc",
			},
			"username": schema.StringAttribute{
				Optional: true,
				Description: "Username for SPX authentication. If provided along with password, users will be required to login to access the application.",
			},
			"password": schema.StringAttribute{
				Optional: true,
				Description: "Password for SPX authentication. Works in conjunction with username to enable login protection for the application.",
			},
			"s3_templates_url": schema.StringAttribute{
				Optional: true,
				Description: "S3 bucket URL or path for storing and retrieving HTML graphics templates used by SPX for live production graphics.",
			},
			"s3_projects_url": schema.StringAttribute{
				Optional: true,
				Description: "S3 bucket URL or path for storing SPX projects and rundowns data that would normally be stored in the DATAROOT folder.",
			},
			"s3_plugins_url": schema.StringAttribute{
				Optional: true,
				Description: "Configures the S3 bucket URL for storing SPX plugins and extensions. Plugins provide additional functionality like custom controls and user interface panels.",
			},
			"s3_media_url": schema.StringAttribute{
				Optional: true,
				Description: "Configures the S3 bucket URL for storing media assets like images, videos, and other files used by graphics templates.",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional: true,
				Description: "Custom S3-compatible endpoint URL for accessing object storage services other than AWS S3, such as MinIO, DigitalOcean Spaces, or other S3-compatible storage providers.",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "AWS access key ID for authenticating with S3 services to access templates, projects, and media assets stored in cloud storage.",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "AWS secret access key for authenticating with S3 services, paired with the access key ID for secure cloud storage access.",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "AWS region identifier specifying the geographical region where the S3 buckets are located for optimal performance and compliance.",
			},
		},
	}
}

func (r *tuomokuspxgc) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan tuomokuspxgcModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("tuomoku-spx-gc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "tuomoku-spx-gc", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"Username": plan.Username.ValueString(),
		"Password": plan.Password.ValueString(),
		"S3TemplatesUrl": plan.S3templatesurl.ValueString(),
		"S3ProjectsUrl": plan.S3projectsurl.ValueString(),
		"S3PluginsUrl": plan.S3pluginsurl.ValueString(),
		"S3MediaUrl": plan.S3mediaurl.ValueString(),
		"S3EndpointUrl": plan.S3endpointurl.ValueString(),
		"S3AccessKeyId": plan.S3accesskeyid.ValueString(),
		"S3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"S3Region": plan.S3region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "tuomoku-spx-gc", instance["name"].(string), serviceAccessToken)
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
	state := tuomokuspxgcModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("tuomoku-spx-gc"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Username: plan.Username,
		Password: plan.Password,
		S3templatesurl: plan.S3templatesurl,
		S3projectsurl: plan.S3projectsurl,
		S3pluginsurl: plan.S3pluginsurl,
		S3mediaurl: plan.S3mediaurl,
		S3endpointurl: plan.S3endpointurl,
		S3accesskeyid: plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		S3region: plan.S3region,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *tuomokuspxgc) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *tuomokuspxgc) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *tuomokuspxgc) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state tuomokuspxgcModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("tuomoku-spx-gc")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "tuomoku-spx-gc", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
