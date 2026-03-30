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
	_ resource.Resource              = &reconurgeflowsint{}
	_ resource.ResourceWithConfigure = &reconurgeflowsint{}
)

func Newreconurgeflowsint() resource.Resource {
	return &reconurgeflowsint{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newreconurgeflowsint)
}

func (r *reconurgeflowsint) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// reconurgeflowsint is the resource implementation.
type reconurgeflowsint struct {
	osaasContext *osaasclient.Context
}

type reconurgeflowsintModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Databaseurl         types.String       `tfsdk:"database_url"`
	Redisurl         types.String       `tfsdk:"redis_url"`
	Authsecret         types.String       `tfsdk:"auth_secret"`
	Mastervaultkeyv1         types.String       `tfsdk:"master_vault_key_v1"`
	Neo4juribolt         types.String       `tfsdk:"neo4j_uri_bolt"`
	Neo4jusername         types.String       `tfsdk:"neo4j_username"`
	Neo4jpassword         types.String       `tfsdk:"neo4j_password"`
}

func (r *reconurgeflowsint) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_reconurge_flowsint"
}

// Schema defines the schema for the resource.
func (r *reconurgeflowsint) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock the power of ethical intelligence with Flowsint, the ultimate open-source OSINT tool. Dive deep into graph-based investigations with cutting-edge enrichers for domains, IPs, organizations, and more!`,
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
				Description: "Name of flowsint",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "PostgreSQL database connection URL used by Flowsint to store user accounts, investigations, scan results, chat messages, and other application data",
			},
			"redis_url": schema.StringAttribute{
				Required: true,
				Description: "Redis connection URL used for caching, session management, and Celery task queue backend for processing enricher jobs asynchronously",
			},
			"auth_secret": schema.StringAttribute{
				Required: true,
				Description: "Secret key used for JWT token signing and user authentication in the FastAPI backend",
			},
			"master_vault_key_v1": schema.StringAttribute{
				Required: true,
				Description: "Master encryption key for the secure vault system that stores API keys and sensitive credentials used by enrichers",
			},
			"neo4j_uri_bolt": schema.StringAttribute{
				Required: true,
				Description: "Neo4j database Bolt protocol connection URI used for storing and querying the OSINT investigation graph data",
			},
			"neo4j_username": schema.StringAttribute{
				Required: true,
				Description: "Username for authenticating to the Neo4j graph database",
			},
			"neo4j_password": schema.StringAttribute{
				Required: true,
				Description: "Password for authenticating to the Neo4j graph database",
			},
		},
	}
}

func (r *reconurgeflowsint) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan reconurgeflowsintModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("reconurge-flowsint")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "reconurge-flowsint", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DatabaseUrl": plan.Databaseurl.ValueString(),
		"RedisUrl": plan.Redisurl.ValueString(),
		"AuthSecret": plan.Authsecret.ValueString(),
		"MasterVaultKeyV1": plan.Mastervaultkeyv1.ValueString(),
		"Neo4jUriBolt": plan.Neo4juribolt.ValueString(),
		"Neo4jUsername": plan.Neo4jusername.ValueString(),
		"Neo4jPassword": plan.Neo4jpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "reconurge-flowsint", instance["name"].(string), serviceAccessToken)
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
	state := reconurgeflowsintModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("reconurge-flowsint"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Databaseurl: plan.Databaseurl,
		Redisurl: plan.Redisurl,
		Authsecret: plan.Authsecret,
		Mastervaultkeyv1: plan.Mastervaultkeyv1,
		Neo4juribolt: plan.Neo4juribolt,
		Neo4jusername: plan.Neo4jusername,
		Neo4jpassword: plan.Neo4jpassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *reconurgeflowsint) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *reconurgeflowsint) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *reconurgeflowsint) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state reconurgeflowsintModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("reconurge-flowsint")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "reconurge-flowsint", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
