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
	_ resource.Resource              = &keycloakkeycloak{}
	_ resource.ResourceWithConfigure = &keycloakkeycloak{}
)

func Newkeycloakkeycloak() resource.Resource {
	return &keycloakkeycloak{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newkeycloakkeycloak)
}

func (r *keycloakkeycloak) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// keycloakkeycloak is the resource implementation.
type keycloakkeycloak struct {
	osaasContext *osaasclient.Context
}

type keycloakkeycloakModel struct {
	InstanceUrl   types.String `tfsdk:"instance_url"`
	ServiceId     types.String `tfsdk:"service_id"`
	ExternalIp    types.String `tfsdk:"external_ip"`
	ExternalPort  types.Int32  `tfsdk:"external_port"`
	Name          types.String `tfsdk:"name"`
	Databaseurl   types.String `tfsdk:"database_url"`
	Adminuser     types.String `tfsdk:"admin_user"`
	Adminpassword types.String `tfsdk:"admin_password"`
}

func (r *keycloakkeycloak) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_keycloak_keycloak"
}

// Schema defines the schema for the resource.
func (r *keycloakkeycloak) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly add authentication to your applications with Keycloak. Secure services, manage users, and implement strong authentication—all with minimal setup. Transform your identity management now!`,
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
				Description: "Name of keycloak",
			},
			"database_url": schema.StringAttribute{
				Required:    true,
				Description: "",
			},
			"admin_user": schema.StringAttribute{
				Required:    true,
				Description: "",
			},
			"admin_password": schema.StringAttribute{
				Required:    true,
				Description: "",
			},
		},
	}
}

func (r *keycloakkeycloak) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan keycloakkeycloakModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("keycloak-keycloak")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "keycloak-keycloak", serviceAccessToken, map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"DatabaseUrl":   plan.Databaseurl.ValueString(),
		"AdminUser":     plan.Adminuser.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "keycloak-keycloak", instance["name"].(string), serviceAccessToken)
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
	state := keycloakkeycloakModel{
		InstanceUrl:   types.StringValue(instance["url"].(string)),
		ServiceId:     types.StringValue("keycloak-keycloak"),
		ExternalIp:    types.StringValue(externalIp),
		ExternalPort:  types.Int32Value(int32(externalPort)),
		Name:          plan.Name,
		Databaseurl:   plan.Databaseurl,
		Adminuser:     plan.Adminuser,
		Adminpassword: plan.Adminpassword,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *keycloakkeycloak) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *keycloakkeycloak) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *keycloakkeycloak) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state keycloakkeycloakModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("keycloak-keycloak")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "keycloak-keycloak", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
