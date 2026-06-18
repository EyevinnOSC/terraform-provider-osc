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
	_ resource.Resource              = &centrifugalcentrifugo{}
	_ resource.ResourceWithConfigure = &centrifugalcentrifugo{}
)

func Newcentrifugalcentrifugo() resource.Resource {
	return &centrifugalcentrifugo{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newcentrifugalcentrifugo)
}

func (r *centrifugalcentrifugo) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// centrifugalcentrifugo is the resource implementation.
type centrifugalcentrifugo struct {
	osaasContext *osaasclient.Context
}

type centrifugalcentrifugoModel struct {
	InstanceUrl        types.String `tfsdk:"instance_url"`
	ServiceId          types.String `tfsdk:"service_id"`
	ExternalIp         types.String `tfsdk:"external_ip"`
	ExternalPort       types.Int32  `tfsdk:"external_port"`
	Name               types.String `tfsdk:"name"`
	Tokenhmacsecretkey types.String `tfsdk:"token_hmac_secret_key"`
	Adminpassword      types.String `tfsdk:"admin_password"`
	Apikey             types.String `tfsdk:"api_key"`
	Redisurl           types.String `tfsdk:"redis_url"`
}

func (r *centrifugalcentrifugo) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_centrifugal_centrifugo"
}

// Schema defines the schema for the resource.
func (r *centrifugalcentrifugo) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost your app&#39;s real-time capabilities with Centrifugo, an open-source messaging server supporting WebSocket, HTTP-streaming, and more. Scale effortlessly, integrate with any backend, and enhance user engagement today!`,
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
				Description: "Name of centrifugo",
			},
			"token_hmac_secret_key": schema.StringAttribute{
				Required:    true,
				Description: "Secret key used for HMAC signing of JWT tokens for connection authentication",
			},
			"admin_password": schema.StringAttribute{
				Required:    true,
				Description: "Password required to access Centrifugo&#39;s embedded admin web UI",
			},
			"api_key": schema.StringAttribute{
				Optional:    true,
				Description: "Authentication key for accessing Centrifugo&#39;s HTTP and GRPC server API",
			},
			"redis_url": schema.StringAttribute{
				Optional:    true,
				Description: "Connection URL for Redis server used for built-in scalability and message brokering",
			},
		},
	}
}

func (r *centrifugalcentrifugo) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan centrifugalcentrifugoModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("centrifugal-centrifugo")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "centrifugal-centrifugo", serviceAccessToken, map[string]interface{}{
		"name":               plan.Name.ValueString(),
		"TokenHmacSecretKey": plan.Tokenhmacsecretkey.ValueString(),
		"AdminPassword":      plan.Adminpassword.ValueString(),
		"ApiKey":             plan.Apikey.ValueString(),
		"RedisUrl":           plan.Redisurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "centrifugal-centrifugo", instance["name"].(string), serviceAccessToken)
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
	state := centrifugalcentrifugoModel{
		InstanceUrl:        types.StringValue(instance["url"].(string)),
		ServiceId:          types.StringValue("centrifugal-centrifugo"),
		ExternalIp:         types.StringValue(externalIp),
		ExternalPort:       types.Int32Value(int32(externalPort)),
		Name:               plan.Name,
		Tokenhmacsecretkey: plan.Tokenhmacsecretkey,
		Adminpassword:      plan.Adminpassword,
		Apikey:             plan.Apikey,
		Redisurl:           plan.Redisurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *centrifugalcentrifugo) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *centrifugalcentrifugo) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *centrifugalcentrifugo) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state centrifugalcentrifugoModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("centrifugal-centrifugo")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "centrifugal-centrifugo", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
