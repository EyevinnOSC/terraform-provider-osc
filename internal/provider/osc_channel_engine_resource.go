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
	_ resource.Resource              = &channelengine{}
	_ resource.ResourceWithConfigure = &channelengine{}
)

func Newchannelengine() resource.Resource {
	return &channelengine{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newchannelengine)
}

func (r *channelengine) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// channelengine is the resource implementation.
type channelengine struct {
	osaasContext *osaasclient.Context
}

type channelengineModel struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Url         types.String       `tfsdk:"url"`
	Opts.defaultslateuri         types.String       `tfsdk:"opts.default_slate_uri"`
	Opts.preroll.url         types.String       `tfsdk:"opts.preroll.url"`
	Opts.preroll.duration         types.String       `tfsdk:"opts.preroll.duration"`
	Opts.webhook.apikey         types.String       `tfsdk:"opts.webhook.apikey"`
}

func (r *channelengine) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_channel_engine_resource"
}

// Schema defines the schema for the resource.
func (r *channelengine) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"url": schema.StringAttribute{
				Required: true,
			},
			"opts.default_slate_uri": schema.StringAttribute{
				Optional: true,
			},
			"opts.preroll.url": schema.StringAttribute{
				Optional: true,
			},
			"opts.preroll.duration": schema.StringAttribute{
				Optional: true,
			},
			"opts.webhook.apikey": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *channelengine) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan channelengineModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("channel-engine")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "channel-engine", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"url": plan.Url.ValueString(),
		"opts.defaultSlateUri": plan.Opts.defaultslateuri.ValueString(),
		"opts.preroll.url": plan.Opts.preroll.url.ValueString(),
		"opts.preroll.duration": plan.Opts.preroll.duration.ValueString(),
		"opts.webhook.apikey": plan.Opts.webhook.apikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "channel-engine", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := channelengineModel{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Url: plan.Url,
		Opts.defaultslateuri: plan.Opts.defaultslateuri,
		Opts.preroll.url: plan.Opts.preroll.url,
		Opts.preroll.duration: plan.Opts.preroll.duration,
		Opts.webhook.apikey: plan.Opts.webhook.apikey,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *channelengine) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *channelengine) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *channelengine) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state channelengineModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("channel-engine")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "channel-engine", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
