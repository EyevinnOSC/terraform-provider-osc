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
	_ resource.Resource              = &alexbj75foodrecipecollectorapp{}
	_ resource.ResourceWithConfigure = &alexbj75foodrecipecollectorapp{}
)

func Newalexbj75foodrecipecollectorapp() resource.Resource {
	return &alexbj75foodrecipecollectorapp{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newalexbj75foodrecipecollectorapp)
}

func (r *alexbj75foodrecipecollectorapp) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// alexbj75foodrecipecollectorapp is the resource implementation.
type alexbj75foodrecipecollectorapp struct {
	osaasContext *osaasclient.Context
}

type alexbj75foodrecipecollectorappModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Alloworigin         bool       `tfsdk:"allow_origin"`
	Databaseurl         types.String       `tfsdk:"database_url"`
}

func (r *alexbj75foodrecipecollectorapp) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_alexbj75_food_recipe_collector_app"
}

// Schema defines the schema for the resource.
func (r *alexbj75foodrecipecollectorapp) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly collect and organize all your favorite recipes with our powerful app. Paste any recipe URL, let our backend do the heavy lifting, and enjoy a unified, easy-to-navigate view. Delight in hassle-free culinary exploration today!`,
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
				Description: "Name of food-recipe-collector-app",
			},
			"allow_origin": schema.BoolAttribute{
				Optional: true,
				Description: "Controls Cross-Origin Resource Sharing (CORS) permissions for the API, determining which domains can make requests to the backend",
			},
			"database_url": schema.StringAttribute{
				Required: true,
				Description: "Complete database connection URL containing all necessary connection parameters for the MariaDB instance where recipes are stored",
			},
		},
	}
}

func (r *alexbj75foodrecipecollectorapp) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alexbj75foodrecipecollectorappModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-food-recipe-collector-app")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "alexbj75-food-recipe-collector-app", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"allowOrigin": plan.Alloworigin,
		"databaseUrl": plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "alexbj75-food-recipe-collector-app", instance["name"].(string), serviceAccessToken)
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
	state := alexbj75foodrecipecollectorappModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("alexbj75-food-recipe-collector-app"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Alloworigin: plan.Alloworigin,
		Databaseurl: plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *alexbj75foodrecipecollectorapp) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *alexbj75foodrecipecollectorapp) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *alexbj75foodrecipecollectorapp) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alexbj75foodrecipecollectorappModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-food-recipe-collector-app")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "alexbj75-food-recipe-collector-app", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
