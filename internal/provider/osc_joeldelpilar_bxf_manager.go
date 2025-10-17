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
	_ resource.Resource              = &joeldelpilarbxfmanager{}
	_ resource.ResourceWithConfigure = &joeldelpilarbxfmanager{}
)

func Newjoeldelpilarbxfmanager() resource.Resource {
	return &joeldelpilarbxfmanager{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newjoeldelpilarbxfmanager)
}

func (r *joeldelpilarbxfmanager) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// joeldelpilarbxfmanager is the resource implementation.
type joeldelpilarbxfmanager struct {
	osaasContext *osaasclient.Context
}

type joeldelpilarbxfmanagerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	S3endpoint         types.String       `tfsdk:"s3_endpoint"`
	S3region         types.String       `tfsdk:"s3_region"`
	S3accesskeyid         types.String       `tfsdk:"s3_access_key_id"`
	S3secretaccesskey         types.String       `tfsdk:"s3_secret_access_key"`
	S3bucketname         types.String       `tfsdk:"s3_bucket_name"`
}

func (r *joeldelpilarbxfmanager) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_joeldelpilar_bxf_manager"
}

// Schema defines the schema for the resource.
func (r *joeldelpilarbxfmanager) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your broadcast file management with BXF Manager. Effortlessly edit, format, and store BXF files with cloud support, all within a user-friendly interface! Save time and keep organized.`,
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
				Description: "Name of bxf-manager",
			},
			"s3_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_region": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_access_key_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_secret_access_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"s3_bucket_name": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *joeldelpilarbxfmanager) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan joeldelpilarbxfmanagerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("joeldelpilar-bxf-manager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "joeldelpilar-bxf-manager", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"s3Endpoint": plan.S3endpoint.ValueString(),
		"s3Region": plan.S3region.ValueString(),
		"s3AccessKeyId": plan.S3accesskeyid.ValueString(),
		"s3SecretAccessKey": plan.S3secretaccesskey.ValueString(),
		"s3BucketName": plan.S3bucketname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "joeldelpilar-bxf-manager", instance["name"].(string), serviceAccessToken)
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
	state := joeldelpilarbxfmanagerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("joeldelpilar-bxf-manager"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		S3endpoint: plan.S3endpoint,
		S3region: plan.S3region,
		S3accesskeyid: plan.S3accesskeyid,
		S3secretaccesskey: plan.S3secretaccesskey,
		S3bucketname: plan.S3bucketname,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *joeldelpilarbxfmanager) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *joeldelpilarbxfmanager) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *joeldelpilarbxfmanager) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state joeldelpilarbxfmanagerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("joeldelpilar-bxf-manager")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "joeldelpilar-bxf-manager", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
