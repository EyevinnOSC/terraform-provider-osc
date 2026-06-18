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
	_ resource.Resource              = &unleashunleash{}
	_ resource.ResourceWithConfigure = &unleashunleash{}
)

func Newunleashunleash() resource.Resource {
	return &unleashunleash{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newunleashunleash)
}

func (r *unleashunleash) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// unleashunleash is the resource implementation.
type unleashunleash struct {
	osaasContext *osaasclient.Context
}

type unleashunleashModel struct {
	InstanceUrl           types.String `tfsdk:"instance_url"`
	ServiceId             types.String `tfsdk:"service_id"`
	ExternalIp            types.String `tfsdk:"external_ip"`
	ExternalPort          types.Int32  `tfsdk:"external_port"`
	Name                  types.String `tfsdk:"name"`
	Databaseurl           types.String `tfsdk:"database_url"`
	Initfrontendapitokens types.String `tfsdk:"init_frontend_api_tokens"`
	Initbackendapitokens  types.String `tfsdk:"init_backend_api_tokens"`
}

func (r *unleashunleash) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_unleash_unleash"
}

// Schema defines the schema for the resource.
func (r *unleashunleash) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unleash your development with Unleash&#39;s feature management platform. Control feature rollouts, test with real data, and deploy seamlessly across various environments with robust integrations and flexible SDKs.`,
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
				Description: "Name of unleash",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "PostgreSQL database connection URL for Unleash to store feature flags, user data, and configuration. Unleash requires a PostgreSQL database to persist all its data including features, strategies, users, and audit logs.",
			},
			"init_frontend_api_tokens": schema.StringAttribute{
				Required:    true,
				Description: "Comma-separated list of API tokens to initialize for frontend/client-side SDK authentication. These tokens are used by frontend SDKs (React, Vue, Svelte, etc.) to connect to Unleash&#39;s frontend API endpoint.",
			},
			"init_backend_api_tokens": schema.StringAttribute{
				Required:    true,
				Description: "Comma-separated list of API tokens to initialize for backend/server-side SDK authentication. These tokens are used by backend SDKs (Node.js, Java, Python, etc.) to connect to Unleash&#39;s main API.",
			},
		},
	}
}

func (r *unleashunleash) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan unleashunleashModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("unleash-unleash")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "unleash-unleash", serviceAccessToken, map[string]interface{}{
		"name":                  plan.Name.ValueString(),
		"DatabaseUrl":           plan.Databaseurl.ValueString(),
		"InitFrontendApiTokens": plan.Initfrontendapitokens.ValueString(),
		"InitBackendApiTokens":  plan.Initbackendapitokens.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "unleash-unleash", instance["name"].(string), serviceAccessToken)
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
	state := unleashunleashModel{
		InstanceUrl:           types.StringValue(instance["url"].(string)),
		ServiceId:             types.StringValue("unleash-unleash"),
		ExternalIp:            types.StringValue(externalIp),
		ExternalPort:          types.Int32Value(int32(externalPort)),
		Name:                  plan.Name,
		Databaseurl:           plan.Databaseurl,
		Initfrontendapitokens: plan.Initfrontendapitokens,
		Initbackendapitokens:  plan.Initbackendapitokens,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *unleashunleash) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *unleashunleash) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *unleashunleash) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state unleashunleashModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("unleash-unleash")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "unleash-unleash", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
