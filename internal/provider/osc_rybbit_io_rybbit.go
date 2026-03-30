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
	_ resource.Resource              = &rybbitiorybbit{}
	_ resource.ResourceWithConfigure = &rybbitiorybbit{}
)

func Newrybbitiorybbit() resource.Resource {
	return &rybbitiorybbit{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newrybbitiorybbit)
}

func (r *rybbitiorybbit) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// rybbitiorybbit is the resource implementation.
type rybbitiorybbit struct {
	osaasContext *osaasclient.Context
}

type rybbitiorybbitModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Postgreshost         types.String       `tfsdk:"postgres_host"`
	Postgresport         types.String       `tfsdk:"postgres_port"`
	Postgresuser         types.String       `tfsdk:"postgres_user"`
	Postgrespassword         types.String       `tfsdk:"postgres_password"`
	Postgresdb         types.String       `tfsdk:"postgres_db"`
	Clickhousehost         types.String       `tfsdk:"clickhouse_host"`
	Clickhousedb         types.String       `tfsdk:"clickhouse_db"`
	Clickhousepassword         types.String       `tfsdk:"clickhouse_password"`
	Betterauthsecret         types.String       `tfsdk:"better_auth_secret"`
	Redishost         types.String       `tfsdk:"redis_host"`
	Redisport         types.String       `tfsdk:"redis_port"`
	Redispassword         types.String       `tfsdk:"redis_password"`
	Disablesignup         bool       `tfsdk:"disable_signup"`
	Mapboxtoken         types.String       `tfsdk:"mapbox_token"`
	Resendapikey         types.String       `tfsdk:"resend_api_key"`
}

func (r *rybbitiorybbit) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_rybbit_io_rybbit"
}

// Schema defines the schema for the resource.
func (r *rybbitiorybbit) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your web analytics with Rybbit! This open-source, privacy-friendly alternative to Google Analytics is easy to set up and use. Gain insights with advanced features like session replays and real-time dashboards.`,
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
				Description: "Name of rybbit",
			},
			"postgres_host": schema.StringAttribute{
				Required: true,
				Description: "The hostname or IP address of your PostgreSQL database server. PostgreSQL is used by Rybbit to store user accounts, site configurations, organization settings, and other application metadata.",
			},
			"postgres_port": schema.StringAttribute{
				Optional: true,
				Description: "The port number on which your PostgreSQL database server is listening. If not specified, the default PostgreSQL port (5432) will be used.",
			},
			"postgres_user": schema.StringAttribute{
				Required: true,
				Description: "The username for authenticating with your PostgreSQL database. This user must have the necessary permissions to create, read, update, and delete data in the specified database.",
			},
			"postgres_password": schema.StringAttribute{
				Required: true,
				Description: "The password for authenticating with your PostgreSQL database using the specified username.",
			},
			"postgres_db": schema.StringAttribute{
				Required: true,
				Description: "The name of the PostgreSQL database that Rybbit will use to store its application data. This database will contain tables for users, sites, organizations, and other metadata.",
			},
			"clickhouse_host": schema.StringAttribute{
				Required: true,
				Description: "The hostname or IP address of your ClickHouse database server. ClickHouse is used by Rybbit to store and analyze high-volume analytics data including pageviews, events, sessions, and user interactions.",
			},
			"clickhouse_db": schema.StringAttribute{
				Optional: true,
				Description: "The name of the ClickHouse database that Rybbit will use for storing analytics data. If not specified, a default database name will be used.",
			},
			"clickhouse_password": schema.StringAttribute{
				Required: true,
				Description: "The password for authenticating with your ClickHouse database server. This is required for secure access to the analytics database.",
			},
			"better_auth_secret": schema.StringAttribute{
				Required: true,
				Description: "A secret key used by Rybbit&#39;s authentication system to encrypt and sign tokens, sessions, and other security-related data. This should be a long, random string.",
			},
			"redis_host": schema.StringAttribute{
				Optional: true,
				Description: "The hostname or IP address of your Redis server. Redis is used by Rybbit for caching, session storage, and improving application performance.",
			},
			"redis_port": schema.StringAttribute{
				Optional: true,
				Description: "The port number on which your Redis server is listening. If not specified, the default Redis port (6379) will be used.",
			},
			"redis_password": schema.StringAttribute{
				Optional: true,
				Description: "The password for authenticating with your Redis server, if authentication is enabled on your Redis instance.",
			},
			"disable_signup": schema.BoolAttribute{
				Optional: true,
				Description: "When set to true, prevents new users from creating accounts through the signup process. Useful for private installations where you want to control user access.",
			},
			"mapbox_token": schema.StringAttribute{
				Optional: true,
				Description: "Your Mapbox API token for enabling advanced map visualizations in Rybbit&#39;s analytics dashboard. Required for the geographic analytics features including the interactive globe and detailed location maps.",
			},
			"resend_api_key": schema.StringAttribute{
				Optional: true,
				Description: "Your Resend API key for sending transactional emails such as password resets, account invitations, and other notifications from your Rybbit installation.",
			},
		},
	}
}

func (r *rybbitiorybbit) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan rybbitiorybbitModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("rybbit-io-rybbit")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "rybbit-io-rybbit", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PostgresHost": plan.Postgreshost.ValueString(),
		"PostgresPort": plan.Postgresport.ValueString(),
		"PostgresUser": plan.Postgresuser.ValueString(),
		"PostgresPassword": plan.Postgrespassword.ValueString(),
		"PostgresDb": plan.Postgresdb.ValueString(),
		"ClickhouseHost": plan.Clickhousehost.ValueString(),
		"ClickhouseDb": plan.Clickhousedb.ValueString(),
		"ClickhousePassword": plan.Clickhousepassword.ValueString(),
		"BetterAuthSecret": plan.Betterauthsecret.ValueString(),
		"RedisHost": plan.Redishost.ValueString(),
		"RedisPort": plan.Redisport.ValueString(),
		"RedisPassword": plan.Redispassword.ValueString(),
		"DisableSignup": plan.Disablesignup,
		"MapboxToken": plan.Mapboxtoken.ValueString(),
		"ResendApiKey": plan.Resendapikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "rybbit-io-rybbit", instance["name"].(string), serviceAccessToken)
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
	state := rybbitiorybbitModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("rybbit-io-rybbit"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Postgreshost: plan.Postgreshost,
		Postgresport: plan.Postgresport,
		Postgresuser: plan.Postgresuser,
		Postgrespassword: plan.Postgrespassword,
		Postgresdb: plan.Postgresdb,
		Clickhousehost: plan.Clickhousehost,
		Clickhousedb: plan.Clickhousedb,
		Clickhousepassword: plan.Clickhousepassword,
		Betterauthsecret: plan.Betterauthsecret,
		Redishost: plan.Redishost,
		Redisport: plan.Redisport,
		Redispassword: plan.Redispassword,
		Disablesignup: plan.Disablesignup,
		Mapboxtoken: plan.Mapboxtoken,
		Resendapikey: plan.Resendapikey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *rybbitiorybbit) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *rybbitiorybbit) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *rybbitiorybbit) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state rybbitiorybbitModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("rybbit-io-rybbit")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "rybbit-io-rybbit", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
