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
	_ resource.Resource              = &atelierelivepublicorchestrationgui{}
	_ resource.ResourceWithConfigure = &atelierelivepublicorchestrationgui{}
)

func Newatelierelivepublicorchestrationgui() resource.Resource {
	return &atelierelivepublicorchestrationgui{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newatelierelivepublicorchestrationgui)
}

func (r *atelierelivepublicorchestrationgui) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// atelierelivepublicorchestrationgui is the resource implementation.
type atelierelivepublicorchestrationgui struct {
	osaasContext *osaasclient.Context
}

type atelierelivepublicorchestrationguiModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Mongodburi         types.String       `tfsdk:"mongo_db_uri"`
	Apiurl         types.String       `tfsdk:"api_url"`
	Apicredentials         types.String       `tfsdk:"api_credentials"`
	Nextauthsecret         types.String       `tfsdk:"next_auth_secret"`
}

func (r *atelierelivepublicorchestrationgui) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_ateliere_live_public_orchestration_gui"
}

// Schema defines the schema for the resource.
func (r *atelierelivepublicorchestrationgui) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Ateliere Live GUI is a cutting-edge tool for streamlined development. With Node.js 18.x compatibility and easy Ubuntu installation, it ensures a seamless workflow. Explore the external documentation for enhanced productivity. Get your hands on this game-changer today!`,
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
				Description: "Name of orchestration-gui",
			},
			"mongo_db_uri": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"api_url": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"api_credentials": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"next_auth_secret": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *atelierelivepublicorchestrationgui) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan atelierelivepublicorchestrationguiModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ateliere-live-public-orchestration-gui")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "ateliere-live-public-orchestration-gui", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"MongoDbUri": plan.Mongodburi.ValueString(),
		"ApiUrl": plan.Apiurl.ValueString(),
		"ApiCredentials": plan.Apicredentials.ValueString(),
		"NextAuthSecret": plan.Nextauthsecret.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "ateliere-live-public-orchestration-gui", instance["name"].(string), serviceAccessToken)
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
	state := atelierelivepublicorchestrationguiModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("ateliere-live-public-orchestration-gui"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Mongodburi: plan.Mongodburi,
		Apiurl: plan.Apiurl,
		Apicredentials: plan.Apicredentials,
		Nextauthsecret: plan.Nextauthsecret,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *atelierelivepublicorchestrationgui) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *atelierelivepublicorchestrationgui) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *atelierelivepublicorchestrationgui) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state atelierelivepublicorchestrationguiModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("ateliere-live-public-orchestration-gui")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "ateliere-live-public-orchestration-gui", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
