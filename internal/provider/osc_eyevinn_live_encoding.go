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
	_ resource.Resource              = &eyevinnliveencoding{}
	_ resource.ResourceWithConfigure = &eyevinnliveencoding{}
)

func Neweyevinnliveencoding() resource.Resource {
	return &eyevinnliveencoding{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnliveencoding)
}

func (r *eyevinnliveencoding) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnliveencoding is the resource implementation.
type eyevinnliveencoding struct {
	osaasContext *osaasclient.Context
}

type eyevinnliveencodingModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Hlsonly         bool       `tfsdk:"hls_only"`
	Streamkey         types.String       `tfsdk:"stream_key"`
	Inputurl         types.String       `tfsdk:"input_url"`
	Inputdialtimeout         types.String       `tfsdk:"input_dial_timeout"`
	Outputurl         types.String       `tfsdk:"output_url"`
	Ladder         types.String       `tfsdk:"ladder"`
	Framerate         types.String       `tfsdk:"framerate"`
	Ratecontrol         types.String       `tfsdk:"rate_control"`
	Maxratefactor         types.String       `tfsdk:"maxrate_factor"`
	Bufsizefactor         types.String       `tfsdk:"bufsize_factor"`
	Segmentduration         types.String       `tfsdk:"segment_duration"`
	Subtitleurl         types.String       `tfsdk:"subtitle_url"`
	Subtitlelanguage         types.String       `tfsdk:"subtitle_language"`
	Subtitlename         types.String       `tfsdk:"subtitle_name"`
	Subtitledefault         types.String       `tfsdk:"subtitle_default"`
	Segmenttype         types.String       `tfsdk:"segment_type"`
	Programdatetime         types.String       `tfsdk:"program_date_time"`
}

func (r *eyevinnliveencoding) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_live_encoding"
}

// Schema defines the schema for the resource.
func (r *eyevinnliveencoding) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your live streaming with Eyevinn Live Encoding: Open-source, ffmpeg-based, and ready for HLS &amp; MPEG-DASH. Streamline now, CDN-ready.`,
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
				Description: "Name of live-encoding",
			},
			"hls_only": schema.BoolAttribute{
				Optional: true,
				Description: "When enabled only output HLS",
			},
			"stream_key": schema.StringAttribute{
				Optional: true,
				Description: "Configure encoder to push to rtmp://&lt;host&gt;/live/&lt;StreamKey&gt;",
			},
			"input_url": schema.StringAttribute{
				Optional: true,
				Description: "URL endpoint for external service",
			},
			"input_dial_timeout": schema.StringAttribute{
				Optional: true,
				Description: "Timeout value in milliseconds or seconds",
			},
			"output_url": schema.StringAttribute{
				Optional: true,
				Description: "If specified push to CDN origin",
			},
			"ladder": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for ladder",
			},
			"framerate": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for framerate",
			},
			"rate_control": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for ratecontrol",
			},
			"maxrate_factor": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for maxratefactor",
			},
			"bufsize_factor": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for bufsizefactor",
			},
			"segment_duration": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for segmentduration",
			},
			"subtitle_url": schema.StringAttribute{
				Optional: true,
				Description: "URL endpoint for external service",
			},
			"subtitle_language": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for subtitlelanguage",
			},
			"subtitle_name": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for subtitlename",
			},
			"subtitle_default": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for subtitledefault",
			},
			"segment_type": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for segmenttype",
			},
			"program_date_time": schema.StringAttribute{
				Optional: true,
				Description: "Configuration option for programdatetime",
			},
		},
	}
}

func (r *eyevinnliveencoding) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnliveencodingModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-live-encoding")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-live-encoding", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"HlsOnly": plan.Hlsonly,
		"StreamKey": plan.Streamkey.ValueString(),
		"InputUrl": plan.Inputurl.ValueString(),
		"InputDialTimeout": plan.Inputdialtimeout.ValueString(),
		"OutputUrl": plan.Outputurl.ValueString(),
		"Ladder": plan.Ladder.ValueString(),
		"Framerate": plan.Framerate.ValueString(),
		"RateControl": plan.Ratecontrol.ValueString(),
		"MaxrateFactor": plan.Maxratefactor.ValueString(),
		"BufsizeFactor": plan.Bufsizefactor.ValueString(),
		"SegmentDuration": plan.Segmentduration.ValueString(),
		"SubtitleUrl": plan.Subtitleurl.ValueString(),
		"SubtitleLanguage": plan.Subtitlelanguage.ValueString(),
		"SubtitleName": plan.Subtitlename.ValueString(),
		"SubtitleDefault": plan.Subtitledefault.ValueString(),
		"SegmentType": plan.Segmenttype.ValueString(),
		"ProgramDateTime": plan.Programdatetime.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-live-encoding", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnliveencodingModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-live-encoding"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Hlsonly: plan.Hlsonly,
		Streamkey: plan.Streamkey,
		Inputurl: plan.Inputurl,
		Inputdialtimeout: plan.Inputdialtimeout,
		Outputurl: plan.Outputurl,
		Ladder: plan.Ladder,
		Framerate: plan.Framerate,
		Ratecontrol: plan.Ratecontrol,
		Maxratefactor: plan.Maxratefactor,
		Bufsizefactor: plan.Bufsizefactor,
		Segmentduration: plan.Segmentduration,
		Subtitleurl: plan.Subtitleurl,
		Subtitlelanguage: plan.Subtitlelanguage,
		Subtitlename: plan.Subtitlename,
		Subtitledefault: plan.Subtitledefault,
		Segmenttype: plan.Segmenttype,
		Programdatetime: plan.Programdatetime,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnliveencoding) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnliveencoding) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnliveencoding) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnliveencodingModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-live-encoding")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-live-encoding", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
