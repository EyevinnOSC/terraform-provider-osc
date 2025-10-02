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
	_ resource.Resource              = &eyevinnadnormalizer{}
	_ resource.ResourceWithConfigure = &eyevinnadnormalizer{}
)

func Neweyevinnadnormalizer() resource.Resource {
	return &eyevinnadnormalizer{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnadnormalizer)
}

func (r *eyevinnadnormalizer) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnadnormalizer is the resource implementation.
type eyevinnadnormalizer struct {
	osaasContext *osaasclient.Context
}

type eyevinnadnormalizerModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Encoreurl         types.String       `tfsdk:"encore_url"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Adserverurl         types.String       `tfsdk:"ad_server_url"`
	Outputbucketurl         types.String       `tfsdk:"output_bucket_url"`
	Keyregex         types.String       `tfsdk:"key_regex"`
	Keyfield         types.String       `tfsdk:"key_field"`
	Encoreprofile         types.String       `tfsdk:"encore_profile"`
	Assetserverurl         types.String       `tfsdk:"asset_server_url"`
	Jitpackaging         bool       `tfsdk:"jit_packaging"`
	Packagingqueuename         types.String       `tfsdk:"packaging_queue_name"`
	Oscaccesstoken         types.String       `tfsdk:"osc_access_token"`
}

func (r *eyevinnadnormalizer) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_ad_normalizer"
}

// Schema defines the schema for the resource.
func (r *eyevinnadnormalizer) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Optimize your ad delivery with Ad Normalizer! Seamlessly transcode and package VAST creatives for your ad server using a Redis-backed workflow. Ensure efficient media processing and reliable ad streaming.`,
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
				Description: "Name of ad-normalizer",
			},
			"encore_url": schema.StringAttribute{
				Required: true,
				Description: "URL of the related encore instance",
			},
			"redis_url": schema.StringAttribute{
				Optional: true,
				Description: "The url to the redis/valkey instance used. Should use the redis protocol and ideally include port",
			},
			"ad_server_url": schema.StringAttribute{
				Required: true,
				Description: "The url to the ad server endpoint. For the test ad server the path should be /api/v1/ads",
			},
			"output_bucket_url": schema.StringAttribute{
				Required: true,
				Description: "The url to the output folder for the packaged assets",
			},
			"key_regex": schema.StringAttribute{
				Optional: true,
				Description: "Defaults to [^a-zA-Z0-9] if not set",
			},
			"key_field": schema.StringAttribute{
				Optional: true,
				Description: "Which field that the normalizer should use as key in valkey/redis. Optional, defaults to universalAdId if not set",
			},
			"encore_profile": schema.StringAttribute{
				Optional: true,
				Description: "Optional, defaults to &#34;program&#34; if not set",
			},
			"asset_server_url": schema.StringAttribute{
				Optional: true,
				Description: "Optional, http version of OUTPUT_BUCKET_URL is used if not set",
			},
			"jit_packaging": schema.BoolAttribute{
				Optional: true,
				Description: "Signals wether packaging of ads is done JIT or if completed jobs should be put on the packaging queue. optional, defaults to false if not provided",
			},
			"packaging_queue_name": schema.StringAttribute{
				Optional: true,
				Description: "Name of the redis queue used for packaging jobs. Optional, defaults to &#34;package&#34; if not provided",
			},
			"osc_access_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinnadnormalizer) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnadnormalizerModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-ad-normalizer")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-ad-normalizer", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"EncoreUrl": plan.Encoreurl.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"AdServerUrl": plan.Adserverurl.ValueString(),
		"OutputBucketUrl": plan.Outputbucketurl.ValueString(),
		"KeyRegex": plan.Keyregex.ValueString(),
		"KeyField": plan.Keyfield.ValueString(),
		"EncoreProfile": plan.Encoreprofile.ValueString(),
		"AssetServerUrl": plan.Assetserverurl.ValueString(),
		"JitPackaging": plan.Jitpackaging,
		"PackagingQueueName": plan.Packagingqueuename.ValueString(),
		"OscAccessToken": plan.Oscaccesstoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-ad-normalizer", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnadnormalizerModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-ad-normalizer"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Encoreurl: plan.Encoreurl,
		Redisurl: plan.Redisurl,
		Adserverurl: plan.Adserverurl,
		Outputbucketurl: plan.Outputbucketurl,
		Keyregex: plan.Keyregex,
		Keyfield: plan.Keyfield,
		Encoreprofile: plan.Encoreprofile,
		Assetserverurl: plan.Assetserverurl,
		Jitpackaging: plan.Jitpackaging,
		Packagingqueuename: plan.Packagingqueuename,
		Oscaccesstoken: plan.Oscaccesstoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnadnormalizer) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnadnormalizer) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnadnormalizer) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnadnormalizerModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-ad-normalizer")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-ad-normalizer", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
