package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/eyevinn/osaas-client-go"
)

var (
	_ resource.Resource              = &{{_ObjectName}}{}
	_ resource.ResourceWithConfigure = &{{_ObjectName}}{}
)

func New{{_ObjectName}}() resource.Resource {
	return &{{_ObjectName}}{}
}

func init() {
	RegisteredResources = append(RegisteredResources, New{{_ObjectName}})
}

func (r *{{_ObjectName}}) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// {{_ObjectName}} is the resource implementation.
type {{_ObjectName}} struct {
	osaasContext *osaasclient.Context
}

type {{_ObjectName}}Model struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	{{#inputParameters}}
	{{Name}}         {{type}}       `tfsdk:"{{name}}"`
	{{/inputParameters}}
}

func (r *{{_ObjectName}}) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "{{resourceName}}"
}

// Schema defines the schema for the resource.
func (r *{{_ObjectName}}) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			{{#inputParameters}}
			"{{name}}": schema.{{schemaAttribute}}{
				{{flag}}: true,
			},
			{{/inputParameters}}
		},
	}
}

func (r *{{_ObjectName}}) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan {{_ObjectName}}Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("{{serviceId}}")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "{{serviceId}}", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		{{#instanceParameters}}
		"{{name}}": {{value}},
		{{/instanceParameters}}
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "{{serviceId}}", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := {{_ObjectName}}Model{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		{{#inputParameters}}
		{{Name}}: {{{value}}},
		{{/inputParameters}}
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *{{_ObjectName}}) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *{{_ObjectName}}) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *{{_ObjectName}}) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state {{_ObjectName}}Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("{{serviceId}}")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "{{serviceId}}", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
