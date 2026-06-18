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
	_ resource.Resource              = &birmebucketcommander{}
	_ resource.ResourceWithConfigure = &birmebucketcommander{}
)

func Newbirmebucketcommander() resource.Resource {
	return &birmebucketcommander{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmebucketcommander)
}

func (r *birmebucketcommander) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmebucketcommander is the resource implementation.
type birmebucketcommander struct {
	osaasContext *osaasclient.Context
}

type birmebucketcommanderModel struct {
	InstanceUrl    types.String `tfsdk:"instance_url"`
	ServiceId      types.String `tfsdk:"service_id"`
	ExternalIp     types.String `tfsdk:"external_ip"`
	ExternalPort   types.Int32  `tfsdk:"external_port"`
	Name           types.String `tfsdk:"name"`
	Oscaccesstoken types.String `tfsdk:"osc_access_token"`
}

func (r *birmebucketcommander) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_bucket_commander"
}

// Schema defines the schema for the resource.
func (r *birmebucketcommander) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Manage your S3 buckets effortlessly with Bucket Commander, offering a Norton Commander-inspired dual-pane interface. Experience seamless navigation, secure credential management, and quick file operations.`,
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
				Description: "Name of bucket-commander",
			},
			"osc_access_token": schema.StringAttribute{
				Required:    true,
				Description: "Access token for Open Source Cloud services, required for S3-to-S3 file copy operations with real-time job monitoring",
			},
		},
	}
}

func (r *birmebucketcommander) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmebucketcommanderModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-bucket-commander")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-bucket-commander", serviceAccessToken, map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-bucket-commander", instance["name"].(string), serviceAccessToken)
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
	state := birmebucketcommanderModel{
		InstanceUrl:    types.StringValue(instance["url"].(string)),
		ServiceId:      types.StringValue("birme-bucket-commander"),
		ExternalIp:     types.StringValue(externalIp),
		ExternalPort:   types.Int32Value(int32(externalPort)),
		Name:           plan.Name,
		Oscaccesstoken: plan.Oscaccesstoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmebucketcommander) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmebucketcommander) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmebucketcommander) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmebucketcommanderModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-bucket-commander")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-bucket-commander", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
