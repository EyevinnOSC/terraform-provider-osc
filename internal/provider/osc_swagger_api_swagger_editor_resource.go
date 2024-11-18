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
	_ resource.Resource              = &swaggerapiswaggereditor{}
	_ resource.ResourceWithConfigure = &swaggerapiswaggereditor{}
)

func Newswaggerapiswaggereditor() resource.Resource {
	return &swaggerapiswaggereditor{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newswaggerapiswaggereditor)
}

func (r *swaggerapiswaggereditor) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// swaggerapiswaggereditor is the resource implementation.
type swaggerapiswaggereditor struct {
	osaasContext *osaasclient.Context
}

type swaggerapiswaggereditorModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Apidefinitionurl         types.String       `tfsdk:"api_definition_url"`
}

func (r *swaggerapiswaggereditor) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_swagger_api_swagger_editor_resource"
}

// Schema defines the schema for the resource.
func (r *swaggerapiswaggereditor) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Next generation Swagger Editor is here! Edit OpenAPI definitions in JSON or YAML format in your browser and preview documentation in real time. Generate valid OpenAPI definitions for full Swagger tooling support. Upgrade to SwaggerEditor@5 for OpenAPI 3.1.0 support and enjoy a brand-new version built from the ground up. Get your Swagger Editor now!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of swagger-editor",
			},
			"api_definition_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *swaggerapiswaggereditor) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan swaggerapiswaggereditorModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("swagger-api-swagger-editor")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "swagger-api-swagger-editor", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"ApiDefinitionUrl": plan.Apidefinitionurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "swagger-api-swagger-editor", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := swaggerapiswaggereditorModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Apidefinitionurl: plan.Apidefinitionurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *swaggerapiswaggereditor) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *swaggerapiswaggereditor) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *swaggerapiswaggereditor) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state swaggerapiswaggereditorModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("swagger-api-swagger-editor")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "swagger-api-swagger-editor", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
