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
	_ resource.Resource              = &ossappsdynamicog{}
	_ resource.ResourceWithConfigure = &ossappsdynamicog{}
)

func Newossappsdynamicog() resource.Resource {
	return &ossappsdynamicog{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newossappsdynamicog)
}

func (r *ossappsdynamicog) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// ossappsdynamicog is the resource implementation.
type ossappsdynamicog struct {
	osaasContext *osaasclient.Context
}

type ossappsdynamicogModel struct {
	InstanceUrl         types.String `tfsdk:"instance_url"`
	ServiceId           types.String `tfsdk:"service_id"`
	ExternalIp          types.String `tfsdk:"external_ip"`
	ExternalPort        types.Int32  `tfsdk:"external_port"`
	Name                types.String `tfsdk:"name"`
	Nextbeamanalyticsid types.String `tfsdk:"next_beam_analytics_id"`
	Nextdocsaiid        types.String `tfsdk:"next_docs_ai_id"`
}

func (r *ossappsdynamicog) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_oss_apps_dynamic_og"
}

// Schema defines the schema for the resource.
func (r *ossappsdynamicog) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Instantly enhance your content with Dynamic OG&#39;s AI-powered dynamic Open Graph images! Save time on design, and choose from weekly template updates to keep your website visually engaging and fresh.`,
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
				Description: "Name of dynamic-og",
			},
			"next_beam_analytics_id": schema.StringAttribute{
				Optional:    true,
				Description: "Configuration ID for Beam Analytics integration to track website usage and performance metrics for your Dynamic OG application",
			},
			"next_docs_ai_id": schema.StringAttribute{
				Optional:    true,
				Description: "Configuration ID for DocsAI chatbot integration to enhance user interaction and provide automated assistance within your Dynamic OG application",
			},
		},
	}
}

func (r *ossappsdynamicog) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ossappsdynamicogModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("oss-apps-dynamic-og")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "oss-apps-dynamic-og", serviceAccessToken, map[string]interface{}{
		"name":                plan.Name.ValueString(),
		"nextBeamAnalyticsId": plan.Nextbeamanalyticsid.ValueString(),
		"nextDocsAiId":        plan.Nextdocsaiid.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "oss-apps-dynamic-og", instance["name"].(string), serviceAccessToken)
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
	state := ossappsdynamicogModel{
		InstanceUrl:         types.StringValue(instance["url"].(string)),
		ServiceId:           types.StringValue("oss-apps-dynamic-og"),
		ExternalIp:          types.StringValue(externalIp),
		ExternalPort:        types.Int32Value(int32(externalPort)),
		Name:                plan.Name,
		Nextbeamanalyticsid: plan.Nextbeamanalyticsid,
		Nextdocsaiid:        plan.Nextdocsaiid,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *ossappsdynamicog) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *ossappsdynamicog) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *ossappsdynamicog) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ossappsdynamicogModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("oss-apps-dynamic-og")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "oss-apps-dynamic-og", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
