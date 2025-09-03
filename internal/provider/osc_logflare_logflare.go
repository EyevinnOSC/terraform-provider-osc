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
	_ resource.Resource              = &logflarelogflare{}
	_ resource.ResourceWithConfigure = &logflarelogflare{}
)

func Newlogflarelogflare() resource.Resource {
	return &logflarelogflare{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newlogflarelogflare)
}

func (r *logflarelogflare) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// logflarelogflare is the resource implementation.
type logflarelogflare struct {
	osaasContext *osaasclient.Context
}

type logflarelogflareModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Postgresbackendurl         types.String       `tfsdk:"postgres_backend_url"`
	Dbschema         types.String       `tfsdk:"db_schema"`
	Dbencryptionkey         types.String       `tfsdk:"db_encryption_key"`
	Apikey         types.String       `tfsdk:"api_key"`
	Publicaccesstoken         types.String       `tfsdk:"public_access_token"`
	Privateaccesstoken         types.String       `tfsdk:"private_access_token"`
}

func (r *logflarelogflare) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_logflare_logflare"
}

// Schema defines the schema for the resource.
func (r *logflarelogflare) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Streamline your log management with Logflare! Integrate effortlessly, visualize in your browser, and leverage your existing BigQuery setup for seamless data insights. Elevate logging today!`,
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
				Description: "Name of logflare",
			},
			"postgres_backend_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_schema": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"db_encryption_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"api_key": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"public_access_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"private_access_token": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *logflarelogflare) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan logflarelogflareModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("logflare-logflare")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "logflare-logflare", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"PostgresBackendUrl": plan.Postgresbackendurl.ValueString(),
		"DbSchema": plan.Dbschema.ValueString(),
		"DbEncryptionKey": plan.Dbencryptionkey.ValueString(),
		"ApiKey": plan.Apikey.ValueString(),
		"PublicAccessToken": plan.Publicaccesstoken.ValueString(),
		"PrivateAccessToken": plan.Privateaccesstoken.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "logflare-logflare", instance["name"].(string), serviceAccessToken)
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
	state := logflarelogflareModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("logflare-logflare"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Postgresbackendurl: plan.Postgresbackendurl,
		Dbschema: plan.Dbschema,
		Dbencryptionkey: plan.Dbencryptionkey,
		Apikey: plan.Apikey,
		Publicaccesstoken: plan.Publicaccesstoken,
		Privateaccesstoken: plan.Privateaccesstoken,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *logflarelogflare) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *logflarelogflare) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *logflarelogflare) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state logflarelogflareModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("logflare-logflare")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "logflare-logflare", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
