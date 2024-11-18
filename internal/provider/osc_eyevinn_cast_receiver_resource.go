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
	_ resource.Resource              = &eyevinncastreceiver{}
	_ resource.ResourceWithConfigure = &eyevinncastreceiver{}
)

func Neweyevinncastreceiver() resource.Resource {
	return &eyevinncastreceiver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinncastreceiver)
}

func (r *eyevinncastreceiver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinncastreceiver is the resource implementation.
type eyevinncastreceiver struct {
	osaasContext *osaasclient.Context
}

type eyevinncastreceiverModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Title         types.String       `tfsdk:"title"`
	Castreceiveroptions         types.String       `tfsdk:"cast_receiver_options"`
	Playbacklogourl         types.String       `tfsdk:"playback_logo_url"`
	Logourl         types.String       `tfsdk:"logo_url"`
	Castmediaplayerstyle         types.String       `tfsdk:"cast_media_player_style"`
}

func (r *eyevinncastreceiver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_cast_receiver_resource"
}

// Schema defines the schema for the resource.
func (r *eyevinncastreceiver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `A basic custom chromecast receiver that can be configured using environment variables. Add your company branding to your own chromecast receiver without writing a single line of code!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of cast-receiver",
			},
			"title": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"cast_receiver_options": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"playback_logo_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"logo_url": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"cast_media_player_style": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *eyevinncastreceiver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinncastreceiverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-cast-receiver")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-cast-receiver", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"title": plan.Title.ValueString(),
		"castReceiverOptions": plan.Castreceiveroptions.ValueString(),
		"playbackLogoUrl": plan.Playbacklogourl.ValueString(),
		"logoUrl": plan.Logourl.ValueString(),
		"castMediaPlayerStyle": plan.Castmediaplayerstyle.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-cast-receiver", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := eyevinncastreceiverModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Title: plan.Title,
		Castreceiveroptions: plan.Castreceiveroptions,
		Playbacklogourl: plan.Playbacklogourl,
		Logourl: plan.Logourl,
		Castmediaplayerstyle: plan.Castmediaplayerstyle,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinncastreceiver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinncastreceiver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinncastreceiver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinncastreceiverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-cast-receiver")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-cast-receiver", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
