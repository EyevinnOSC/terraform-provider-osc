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
	_ resource.Resource              = &locustiolocust{}
	_ resource.ResourceWithConfigure = &locustiolocust{}
)

func Newlocustiolocust() resource.Resource {
	return &locustiolocust{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newlocustiolocust)
}

func (r *locustiolocust) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// locustiolocust is the resource implementation.
type locustiolocust struct {
	osaasContext *osaasclient.Context
}

type locustiolocustModel struct {
	InstanceUrl   types.String `tfsdk:"instance_url"`
	ServiceId     types.String `tfsdk:"service_id"`
	ExternalIp    types.String `tfsdk:"external_ip"`
	ExternalPort  types.Int32  `tfsdk:"external_port"`
	Name          types.String `tfsdk:"name"`
	Locustfileurl types.String `tfsdk:"locustfile_url"`
}

func (r *locustiolocust) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_locustio_locust"
}

// Schema defines the schema for the resource.
func (r *locustiolocust) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost your performance with Locust! This open-source tool empowers you to conduct efficient load testing using the simplicity of Python. Monitor in real-time with a friendly UI and scale effortlessly!`,
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
				Description: "Name of locust",
			},
			"locustfile_url": schema.StringAttribute{
				Required:    true,
				Description: "Url to the location of your locustfile.py",
			},
		},
	}
}

func (r *locustiolocust) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan locustiolocustModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("locustio-locust")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "locustio-locust", serviceAccessToken, map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"LocustfileUrl": plan.Locustfileurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "locustio-locust", instance["name"].(string), serviceAccessToken)
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
	state := locustiolocustModel{
		InstanceUrl:   types.StringValue(instance["url"].(string)),
		ServiceId:     types.StringValue("locustio-locust"),
		ExternalIp:    types.StringValue(externalIp),
		ExternalPort:  types.Int32Value(int32(externalPort)),
		Name:          plan.Name,
		Locustfileurl: plan.Locustfileurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *locustiolocust) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *locustiolocust) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *locustiolocust) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state locustiolocustModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("locustio-locust")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "locustio-locust", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
