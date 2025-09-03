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
	_ resource.Resource              = &eyevinnplayeranalyticseventsink{}
	_ resource.ResourceWithConfigure = &eyevinnplayeranalyticseventsink{}
)

func Neweyevinnplayeranalyticseventsink() resource.Resource {
	return &eyevinnplayeranalyticseventsink{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinnplayeranalyticseventsink)
}

func (r *eyevinnplayeranalyticseventsink) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// eyevinnplayeranalyticseventsink is the resource implementation.
type eyevinnplayeranalyticseventsink struct {
	osaasContext *osaasclient.Context
}

type eyevinnplayeranalyticseventsinkModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Sqsqueueurl         types.String       `tfsdk:"sqs_queue_url"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Sqsendpoint         types.String       `tfsdk:"sqs_endpoint"`
	Allowedorigins         types.String       `tfsdk:"allowed_origins"`
}

func (r *eyevinnplayeranalyticseventsink) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_eyevinn_player_analytics_eventsink"
}

// Schema defines the schema for the resource.
func (r *eyevinnplayeranalyticseventsink) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock seamless video analytics with Eyevinn Player Analytics Eventsink! Streamline data collection from video players and enhance performance insights. Experience modular flexibility and AWS integration today!`,
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
				Description: "Name of player-analytics-eventsink",
			},
			"sqs_queue_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_access_key_id": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"aws_secret_access_key": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"sqs_endpoint": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"allowed_origins": schema.StringAttribute{
				Optional: true,
				Description: "Provide a comma separated list of origins to allow. If empty allow all",
			},
		},
	}
}

func (r *eyevinnplayeranalyticseventsink) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan eyevinnplayeranalyticseventsinkModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-player-analytics-eventsink")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "eyevinn-player-analytics-eventsink", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"SqsQueueUrl": plan.Sqsqueueurl.ValueString(),
		"AwsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"AwsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"SqsEndpoint": plan.Sqsendpoint.ValueString(),
		"AllowedOrigins": plan.Allowedorigins.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "eyevinn-player-analytics-eventsink", instance["name"].(string), serviceAccessToken)
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
	state := eyevinnplayeranalyticseventsinkModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("eyevinn-player-analytics-eventsink"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Sqsqueueurl: plan.Sqsqueueurl,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Sqsendpoint: plan.Sqsendpoint,
		Allowedorigins: plan.Allowedorigins,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *eyevinnplayeranalyticseventsink) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *eyevinnplayeranalyticseventsink) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *eyevinnplayeranalyticseventsink) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state eyevinnplayeranalyticseventsinkModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("eyevinn-player-analytics-eventsink")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "eyevinn-player-analytics-eventsink", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
