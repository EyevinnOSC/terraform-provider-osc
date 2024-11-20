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
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Type         types.String       `tfsdk:"type"`
	Url         types.String       `tfsdk:"url"`
	Optsusedemuxedaudio         bool       `tfsdk:"optsuse_demuxed_audio"`
	Optsusevttsubtitles         bool       `tfsdk:"optsuse_vtt_subtitles"`
	Optsdefaultslateuri         types.String       `tfsdk:"optsdefault_slate_uri"`
	Optslanglist         string       `tfsdk:"optslang_list"`
	Optslanglistsubs         string       `tfsdk:"optslang_list_subs"`
	Optspreset         types.String       `tfsdk:"optspreset"`
	Optsprerollurl         types.String       `tfsdk:"optsprerollurl"`
	Optsprerollduration         types.String       `tfsdk:"optsprerollduration"`
	Optswebhookapikey         types.String       `tfsdk:"optswebhookapikey"`
}

func (r *channelengine) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_channel_engine"
}

// Schema defines the schema for the resource.
func (r *channelengine) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Based on VOD2Live Technology you can generate a numerous amounts of FAST channels with a fraction of energy consumption compared to live transcoded FAST channels`,
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
				Description: "Enter channel name",
			},
			"type": schema.StringAttribute{
				Required: true,
				Description: "Plugin type",
			},
			"url": schema.StringAttribute{
				Required: true,
				Description: "URL of VOD, playlist to loop or WebHook",
			},
			"optsuse_demuxed_audio": schema.BoolAttribute{
				Optional: true,
				Description: "Use demuxed audio",
			},
			"optsuse_vtt_subtitles": schema.BoolAttribute{
				Optional: true,
				Description: "Use VTT subtitles",
			},
			"optsdefault_slate_uri": schema.StringAttribute{
				Optional: true,
				Description: "URI to default slate",
			},
			"optslang_list": schema.StringAttribute{
				Optional: true,
				Description: "Comma separated list of languages",
			},
			"optslang_list_subs": schema.StringAttribute{
				Optional: true,
				Description: "Comma separated list of subtitle languages",
			},
			"optspreset": schema.StringAttribute{
				Optional: true,
				Description: "Channel preset",
			},
			"optsprerollurl": schema.StringAttribute{
				Optional: true,
				Description: "URL to preroll",
			},
			"optsprerollduration": schema.StringAttribute{
				Optional: true,
				Description: "Duration of preroll in milliseconds",
			},
			"optswebhookapikey": schema.StringAttribute{
				Optional: true,
				Description: "WebHook api key",
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
		"type": plan.Type,
		"url": plan.Url.ValueString(),
		"opts.useDemuxedAudio": plan.Optsusedemuxedaudio,
		"opts.useVttSubtitles": plan.Optsusevttsubtitles,
		"opts.defaultSlateUri": plan.Optsdefaultslateuri.ValueString(),
		"opts.langList": plan.Optslanglist,
		"opts.langListSubs": plan.Optslanglistsubs,
		"opts.preset": plan.Optspreset,
		"opts.preroll.url": plan.Optsprerollurl.ValueString(),
		"opts.preroll.duration": plan.Optsprerollduration.ValueString(),
		"opts.webhook.apikey": plan.Optswebhookapikey.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "channel-engine", instance["name"].(string), serviceAccessToken)
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
	state := channelengineModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("channel-engine"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Type: plan.Type,
		Url: plan.Url,
		Optsusedemuxedaudio: plan.Optsusedemuxedaudio,
		Optsusevttsubtitles: plan.Optsusevttsubtitles,
		Optsdefaultslateuri: plan.Optsdefaultslateuri,
		Optslanglist: plan.Optslanglist,
		Optslanglistsubs: plan.Optslanglistsubs,
		Optspreset: plan.Optspreset,
		Optsprerollurl: plan.Optsprerollurl,
		Optsprerollduration: plan.Optsprerollduration,
		Optswebhookapikey: plan.Optswebhookapikey,
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
