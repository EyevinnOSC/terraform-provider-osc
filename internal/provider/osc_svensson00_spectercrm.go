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
	_ resource.Resource              = &svensson00spectercrm{}
	_ resource.ResourceWithConfigure = &svensson00spectercrm{}
)

func Newsvensson00spectercrm() resource.Resource {
	return &svensson00spectercrm{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newsvensson00spectercrm)
}

func (r *svensson00spectercrm) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// svensson00spectercrm is the resource implementation.
type svensson00spectercrm struct {
	osaasContext *osaasclient.Context
}

type svensson00spectercrmModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Databaseurl         types.String       `tfsdk:"database_url"`
	Jwtsecret         types.String       `tfsdk:"jwt_secret"`
	Refreshtokensecret         types.String       `tfsdk:"refresh_token_secret"`
}

func (r *svensson00spectercrm) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_svensson00_spectercrm"
}

// Schema defines the schema for the resource.
func (r *svensson00spectercrm) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Revolutionize your CRM management with SpecterCRM&#39;s seamless multi-tenant SaaS architecture! Effortlessly handle organizations, contacts, and deals while benefiting from advanced deduplication, comprehensive reporting, and robust API integrations. Upgrade to the all-in-one CRM solution today for streamlined, data-driven success!`,
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
				Description: "Name of spectercrm",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "PostgreSQL database connection string used by Prisma ORM to connect to the database. This is the primary database configuration for the multi-tenant CRM application.",
			},
			"jwt_secret": schema.StringAttribute{
				Optional: true,
				Description: "Secret key used to sign and verify JWT access tokens for user authentication. This ensures the security and integrity of authentication tokens.",
			},
			"refresh_token_secret": schema.StringAttribute{
				Optional: true,
				Description: "Secret key used to sign and verify JWT refresh tokens, which are used to obtain new access tokens without requiring users to re-authenticate.",
			},
		},
	}
}

func (r *svensson00spectercrm) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan svensson00spectercrmModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("svensson00-spectercrm")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "svensson00-spectercrm", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"JwtSecret": plan.Jwtsecret.ValueString(),
		"RefreshTokenSecret": plan.Refreshtokensecret.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "svensson00-spectercrm", instance["name"].(string), serviceAccessToken)
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
	state := svensson00spectercrmModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("svensson00-spectercrm"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Databaseurl: plan.Databaseurl,
		Jwtsecret: plan.Jwtsecret,
		Refreshtokensecret: plan.Refreshtokensecret,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *svensson00spectercrm) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *svensson00spectercrm) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *svensson00spectercrm) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state svensson00spectercrmModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("svensson00-spectercrm")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "svensson00-spectercrm", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
