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
	_ resource.Resource              = &eyevinnsgaiadproxy{}
	_ resource.ResourceWithConfigure = &eyevinnsgaiadproxy{}
)

func Neweyevinnsgaiadproxy() resource.Resource {
	return &eyevinnsgaiadproxy{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnsgaiadproxy)
}

func (r *eyevinnsgaiadproxy) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnsgaiadproxy is the resource implementation.
type eyevinnsgaiadproxy struct {
	osaasContext *osaasclient.Context
}

type eyevinnsgaiadproxyModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Vastendpoint         types.String       `tfsdk:"vast_endpoint"`
	Originurl         types.String       `tfsdk:"origin_url"`
	Insertionmode         types.String       `tfsdk:"insertion_mode"`
	Defaultadduration         types.String       `tfsdk:"default_ad_duration"`
	Defaultrepeatingcycle         types.String       `tfsdk:"default_repeating_cycle"`
	Defaultadnumber         types.String       `tfsdk:"default_ad_number"`
	Testasseturl         types.String       `tfsdk:"test_asset_url"`
}

func (r *eyevinnsgaiadproxy) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_sgai_ad_proxy"
}

// Schema defines the schema for the resource.
func (r *eyevinnsgaiadproxy) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost viewer engagement with our Server-Guided Ad Insertion Proxy! Automatically embed ads into video streams with precision timing. Enhance monetization effortlessly while maintaining a seamless user experience.`,
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
				Description: "Name of sgai-ad-proxy",
			},
			"vast_endpoint": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"origin_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"insertion_mode": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"default_ad_duration": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"default_repeating_cycle": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"default_ad_number": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"test_asset_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnsgaiadproxy) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnsgaiadproxyModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-sgai-ad-proxy")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"VastEndpoint": plan.Vastendpoint.ValueString(),
		"OriginUrl": plan.Originurl.ValueString(),
		"InsertionMode": plan.Insertionmode.ValueString(),
		"DefaultAdDuration": plan.Defaultadduration.ValueString(),
		"DefaultRepeatingCycle": plan.Defaultrepeatingcycle.ValueString(),
		"DefaultAdNumber": plan.Defaultadnumber.ValueString(),
		"TestAssetUrl": plan.Testasseturl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnsgaiadproxyModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-sgai-ad-proxy"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Vastendpoint: plan.Vastendpoint,
		Originurl: plan.Originurl,
		Insertionmode: plan.Insertionmode,
		Defaultadduration: plan.Defaultadduration,
		Defaultrepeatingcycle: plan.Defaultrepeatingcycle,
		Defaultadnumber: plan.Defaultadnumber,
		Testasseturl: plan.Testasseturl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnsgaiadproxy) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnsgaiadproxy) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnsgaiadproxy) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnsgaiadproxyModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-sgai-ad-proxy")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-sgai-ad-proxy", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
