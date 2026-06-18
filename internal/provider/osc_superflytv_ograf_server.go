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
	_ resource.Resource              = &superflytvografserver{}
	_ resource.ResourceWithConfigure = &superflytvografserver{}
)

func Newsuperflytvografserver() resource.Resource {
	return &superflytvografserver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newsuperflytvografserver)
}

func (r *superflytvografserver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// superflytvografserver is the resource implementation.
type superflytvografserver struct {
	osaasContext *osaasclient.Context
}

type superflytvografserverModel struct {
	InstanceUrl       types.String `tfsdk:"instance_url"`
	ServiceId         types.String `tfsdk:"service_id"`
	ExternalIp        types.String `tfsdk:"external_ip"`
	ExternalPort      types.Int32  `tfsdk:"external_port"`
	Name              types.String `tfsdk:"name"`
	S3graphicsurl     types.String `tfsdk:"s3_graphics_url"`
	S3endpointurl     types.String `tfsdk:"s3_endpoint_url"`
	S3accesskeyid     types.String `tfsdk:"s3_access_key_id"`
	S3secretaccesskey types.String `tfsdk:"s3_secret_access_key"`
	S3region          types.String `tfsdk:"s3_region"`
}

func (r *superflytvografserver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_superflytv_ograf_server"
}

// Schema defines the schema for the resource.
func (r *superflytvografserver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your live production with OGraf! Seamlessly render, upload, and control graphics across platforms like OBS and Vmix with ease. Streamline workflows with our intuitive web server and controller.`,
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
				Description: "Name of ograf-server",
			},
			"s3_graphics_url": schema.StringAttribute{
				Optional:    true,
				Description: "The base URL for accessing OGraf graphics stored in an S3-compatible storage service. This would be used by the renderer to load graphics assets from cloud storage rather than local storage.",
			},
			"s3_endpoint_url": schema.StringAttribute{
				Optional:    true,
				Description: "The endpoint URL for the S3-compatible storage service. This allows the server to connect to custom S3 implementations or alternative cloud storage providers beyond AWS S3.",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional:    true,
				Description: "The access key ID for authenticating with the S3 storage service. This is part of the AWS credentials used to securely access the storage bucket containing OGraf graphics.",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional:    true,
				Description: "The secret access key for authenticating with the S3 storage service. This works together with the access key ID to provide secure access to the storage bucket.",
			},
			"s3_region": schema.StringAttribute{
				Optional:    true,
				Description: "The AWS region where the S3 bucket is located. This ensures the server connects to the correct regional endpoint for optimal performance and compliance.",
			},
		},
	}
}

func (r *superflytvografserver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan superflytvografserverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("superflytv-ograf-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "superflytv-ograf-server", serviceAccessToken, map[string]interface{}{
		"name":              plan.Name.ValueString(),
		"S3GraphicsUrl":     plan.S3graphicsurl.ValueString(),
		"S3EndpointUrl":     plan.S3endpointurl.ValueString(),
		"S3AccessKeyId":     plan.S3accesskeyid.ValueString(),
		"S3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"S3Region":          plan.S3region.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "superflytv-ograf-server", instance["name"].(string), serviceAccessToken)
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
	state := superflytvografserverModel{
		InstanceUrl:       types.StringValue(instance["url"].(string)),
		ServiceId:         types.StringValue("superflytv-ograf-server"),
		ExternalIp:        types.StringValue(externalIp),
		ExternalPort:      types.Int32Value(int32(externalPort)),
		Name:              plan.Name,
		S3graphicsurl:     plan.S3graphicsurl,
		S3endpointurl:     plan.S3endpointurl,
		S3accesskeyid:     plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		S3region:          plan.S3region,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *superflytvografserver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *superflytvografserver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *superflytvografserver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state superflytvografserverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("superflytv-ograf-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "superflytv-ograf-server", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
