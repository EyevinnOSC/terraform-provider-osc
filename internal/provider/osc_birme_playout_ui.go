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
	_ resource.Resource              = &birmeplayoutui{}
	_ resource.ResourceWithConfigure = &birmeplayoutui{}
)

func Newbirmeplayoutui() resource.Resource {
	return &birmeplayoutui{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmeplayoutui)
}

func (r *birmeplayoutui) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmeplayoutui is the resource implementation.
type birmeplayoutui struct {
	osaasContext *osaasclient.Context
}

type birmeplayoutuiModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dburl         types.String       `tfsdk:"db_url"`
	Database         types.String       `tfsdk:"database"`
	Username         types.String       `tfsdk:"username"`
	Password         types.String       `tfsdk:"password"`
	Corsorigins         types.String       `tfsdk:"cors_origins"`
}

func (r *birmeplayoutui) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_playout_ui"
}

// Schema defines the schema for the resource.
func (r *birmeplayoutui) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Elevate your media scheduling with Playout UI! Seamlessly manage playlists with live time display, real-time progress tracking, and backend flexibility. Effortlessly organize, edit, and control playback. Ideal for dynamic environments!`,
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
				Description: "Name of playout-ui",
			},
			"db_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"database": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"username": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"cors_origins": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *birmeplayoutui) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmeplayoutuiModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-playout-ui")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-playout-ui", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbUrl": plan.Dburl.ValueString(),
		"Database": plan.Database.ValueString(),
		"Username": plan.Username.ValueString(),
		"Password": plan.Password.ValueString(),
		"CorsOrigins": plan.Corsorigins.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-playout-ui", instance["name"].(string), serviceAccessToken)
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
	state := birmeplayoutuiModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("birme-playout-ui"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dburl: plan.Dburl,
		Database: plan.Database,
		Username: plan.Username,
		Password: plan.Password,
		Corsorigins: plan.Corsorigins,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmeplayoutui) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmeplayoutui) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmeplayoutui) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmeplayoutuiModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-playout-ui")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-playout-ui", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
