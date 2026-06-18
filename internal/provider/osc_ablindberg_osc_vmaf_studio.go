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
	_ resource.Resource              = &ablindbergoscvmafstudio{}
	_ resource.ResourceWithConfigure = &ablindbergoscvmafstudio{}
)

func Newablindbergoscvmafstudio() resource.Resource {
	return &ablindbergoscvmafstudio{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newablindbergoscvmafstudio)
}

func (r *ablindbergoscvmafstudio) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ablindbergoscvmafstudio is the resource implementation.
type ablindbergoscvmafstudio struct {
	osaasContext *osaasclient.Context
}

type ablindbergoscvmafstudioModel struct {
	InstanceUrl    types.String `tfsdk:"instance_url"`
	ServiceId      types.String `tfsdk:"service_id"`
	ExternalIp     types.String `tfsdk:"external_ip"`
	ExternalPort   types.Int32  `tfsdk:"external_port"`
	Name           types.String `tfsdk:"name"`
	Oscaccesstoken types.String `tfsdk:"osc_access_token"`
}

func (r *ablindbergoscvmafstudio) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_ablindberg_osc_vmaf_studio"
}

// Schema defines the schema for the resource.
func (r *ablindbergoscvmafstudio) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your video quality assessment with OSC VMAF Studio, a cloud-based tool leveraging OSC and Eyevinn EasyVMAF. Enjoy effortless S3 storage management, detailed VMAF analysis, and secure credentials.`,
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
				Description: "Name of osc-vmaf-studio",
			},
			"osc_access_token": schema.StringAttribute{
				Optional:    true,
				Description: "Personal Access Token for authenticating with OSC (Open Source Cloud) services, specifically required for accessing Eyevinn EasyVMAF service that performs the VMAF video quality analysis",
			},
		},
	}
}

func (r *ablindbergoscvmafstudio) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ablindbergoscvmafstudioModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ablindberg-osc-vmaf-studio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "ablindberg-osc-vmaf-studio", serviceAccessToken, map[string]interface{}{
		"name":           plan.Name.ValueString(),
		"oscAccessToken": plan.Oscaccesstoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "ablindberg-osc-vmaf-studio", instance["name"].(string), serviceAccessToken)
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
	state := ablindbergoscvmafstudioModel{
		InstanceUrl:    types.StringValue(instance["url"].(string)),
		ServiceId:      types.StringValue("ablindberg-osc-vmaf-studio"),
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
func (r *ablindbergoscvmafstudio) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ablindbergoscvmafstudio) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ablindbergoscvmafstudio) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ablindbergoscvmafstudioModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ablindberg-osc-vmaf-studio")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "ablindberg-osc-vmaf-studio", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
