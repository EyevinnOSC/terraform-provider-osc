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
	_ resource.Resource              = &hasuragraphqlengine{}
	_ resource.ResourceWithConfigure = &hasuragraphqlengine{}
)

func Newhasuragraphqlengine() resource.Resource {
	return &hasuragraphqlengine{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newhasuragraphqlengine)
}

func (r *hasuragraphqlengine) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// hasuragraphqlengine is the resource implementation.
type hasuragraphqlengine struct {
	osaasContext *osaasclient.Context
}

type hasuragraphqlengineModel struct {
	InstanceUrl      types.String `tfsdk:"instance_url"`
	ServiceId        types.String `tfsdk:"service_id"`
	ExternalIp       types.String `tfsdk:"external_ip"`
	ExternalPort     types.Int32  `tfsdk:"external_port"`
	Name             types.String `tfsdk:"name"`
	Databaseurl      types.String `tfsdk:"database_url"`
	Adminsecret      types.String `tfsdk:"admin_secret"`
	Enableconsole    bool         `tfsdk:"enable_console"`
	Jwtsecret        types.String `tfsdk:"jwt_secret"`
	Unauthorizedrole types.String `tfsdk:"unauthorized_role"`
}

func (r *hasuragraphqlengine) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_hasura_graphql_engine"
}

// Schema defines the schema for the resource.
func (r *hasuragraphqlengine) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your application development with Hasura GraphQL Engine! Experience real-time data access and seamless integration with top databases through secure, composable APIs. Empower innovation today!`,
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
				Description: "Name of graphql-engine",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "Connection string for the primary database that Hasura will connect to. This database will be used for storing Hasura&#39;s metadata and can also serve as a data source for GraphQL operations.",
			},
			"admin_secret": schema.StringAttribute{
				Required:    true,
				Description: "Secret key that provides admin access to the Hasura GraphQL Engine. This is used to authenticate requests that require administrative privileges, such as managing metadata, schema changes, and accessing the Hasura Console.",
			},
			"enable_console": schema.BoolAttribute{
				Optional:    true,
				Description: "Controls whether the Hasura Console web interface is enabled and accessible. When enabled, provides a graphical interface for managing schemas, permissions, and testing GraphQL queries.",
			},
			"jwt_secret": schema.StringAttribute{
				Optional:    true,
				Description: "Configuration for JWT (JSON Web Token) based authentication. Defines the secret key or public key used to verify JWT tokens sent by clients for authentication and authorization.",
			},
			"unauthorized_role": schema.StringAttribute{
				Optional:    true,
				Description: "Defines the default role to be used for unauthenticated requests. When set, allows anonymous users to access the GraphQL API with the permissions assigned to this role.",
			},
		},
	}
}

func (r *hasuragraphqlengine) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan hasuragraphqlengineModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("hasura-graphql-engine")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "hasura-graphql-engine", serviceAccessToken, map[string]interface{}{
		"name":             plan.Name.ValueString(),
		"DatabaseUrl":      plan.Databaseurl.ValueString(),
		"AdminSecret":      plan.Adminsecret.ValueString(),
		"EnableConsole":    plan.Enableconsole,
		"JwtSecret":        plan.Jwtsecret.ValueString(),
		"UnauthorizedRole": plan.Unauthorizedrole.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "hasura-graphql-engine", instance["name"].(string), serviceAccessToken)
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
	state := hasuragraphqlengineModel{
		InstanceUrl:      types.StringValue(instance["url"].(string)),
		ServiceId:        types.StringValue("hasura-graphql-engine"),
		ExternalIp:       types.StringValue(externalIp),
		ExternalPort:     types.Int32Value(int32(externalPort)),
		Name:             plan.Name,
		Databaseurl:      plan.Databaseurl,
		Adminsecret:      plan.Adminsecret,
		Enableconsole:    plan.Enableconsole,
		Jwtsecret:        plan.Jwtsecret,
		Unauthorizedrole: plan.Unauthorizedrole,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *hasuragraphqlengine) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *hasuragraphqlengine) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *hasuragraphqlengine) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state hasuragraphqlengineModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("hasura-graphql-engine")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "hasura-graphql-engine", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
