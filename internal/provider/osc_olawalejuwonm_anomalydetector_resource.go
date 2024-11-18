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
	_ resource.Resource              = &olawalejuwonmanomalydetector{}
	_ resource.ResourceWithConfigure = &olawalejuwonmanomalydetector{}
)

func Newolawalejuwonmanomalydetector() resource.Resource {
	return &olawalejuwonmanomalydetector{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newolawalejuwonmanomalydetector)
}

func (r *olawalejuwonmanomalydetector) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// olawalejuwonmanomalydetector is the resource implementation.
type olawalejuwonmanomalydetector struct {
	osaasContext *osaasclient.Context
}

type olawalejuwonmanomalydetectorModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
}

func (r *olawalejuwonmanomalydetector) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_olawalejuwonm_anomalydetector_resource"
}

// Schema defines the schema for the resource.
func (r *olawalejuwonmanomalydetector) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Safeguard your space with Anomaly Detector, a cutting-edge video surveillance solution. Experience real-time anomaly detection using advanced computer vision, ensuring privacy and reducing false alarms. Enhance security efficiently!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of anomalydetector",
			},
		},
	}
}

func (r *olawalejuwonmanomalydetector) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan olawalejuwonmanomalydetectorModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("olawalejuwonm-anomalydetector")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "olawalejuwonm-anomalydetector", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "olawalejuwonm-anomalydetector", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := olawalejuwonmanomalydetectorModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *olawalejuwonmanomalydetector) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *olawalejuwonmanomalydetector) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *olawalejuwonmanomalydetector) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state olawalejuwonmanomalydetectorModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("olawalejuwonm-anomalydetector")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "olawalejuwonm-anomalydetector", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
