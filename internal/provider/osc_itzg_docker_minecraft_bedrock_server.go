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
	_ resource.Resource              = &itzgdockerminecraftbedrockserver{}
	_ resource.ResourceWithConfigure = &itzgdockerminecraftbedrockserver{}
)

func Newitzgdockerminecraftbedrockserver() resource.Resource {
	return &itzgdockerminecraftbedrockserver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newitzgdockerminecraftbedrockserver)
}

func (r *itzgdockerminecraftbedrockserver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// itzgdockerminecraftbedrockserver is the resource implementation.
type itzgdockerminecraftbedrockserver struct {
	osaasContext *osaasclient.Context
}

type itzgdockerminecraftbedrockserverModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Gamemode         types.String       `tfsdk:"game_mode"`
	Maxplayers         types.String       `tfsdk:"max_players"`
	Leveltype         types.String       `tfsdk:"level_type"`
	Variables         types.String       `tfsdk:"variables"`
}

func (r *itzgdockerminecraftbedrockserver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_itzg_docker_minecraft_bedrock_server"
}

// Schema defines the schema for the resource.
func (r *itzgdockerminecraftbedrockserver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unleash the full potential of multiplayer gaming with Itzg&#39;s Minecraft Bedrock Server Docker. Effortlessly run and upgrade your server with cutting-edge game features. Your world, your rules—simplified!`,
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
				Description: "Name of docker-minecraft-bedrock-server",
			},
			"game_mode": schema.StringAttribute{
				Optional: true,
				Description: "Sets the game mode for the Bedrock server, controlling the gameplay experience for players",
			},
			"max_players": schema.StringAttribute{
				Optional: true,
				Description: "Defines the maximum number of players that can connect to the server simultaneously",
			},
			"level_type": schema.StringAttribute{
				Optional: true,
				Description: "Specifies the type of world/level to generate for the server",
			},
			"variables": schema.StringAttribute{
				Optional: true,
				Description: "Allows setting custom server variables as comma-separated key-value pairs or full JSON string for advanced server configuration",
			},
		},
	}
}

func (r *itzgdockerminecraftbedrockserver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itzgdockerminecraftbedrockserverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("itzg-docker-minecraft-bedrock-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "itzg-docker-minecraft-bedrock-server", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"GameMode": plan.Gamemode.ValueString(),
		"MaxPlayers": plan.Maxplayers.ValueString(),
		"LevelType": plan.Leveltype.ValueString(),
		"Variables": plan.Variables.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "itzg-docker-minecraft-bedrock-server", instance["name"].(string), serviceAccessToken)
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
	state := itzgdockerminecraftbedrockserverModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("itzg-docker-minecraft-bedrock-server"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Gamemode: plan.Gamemode,
		Maxplayers: plan.Maxplayers,
		Leveltype: plan.Leveltype,
		Variables: plan.Variables,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *itzgdockerminecraftbedrockserver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *itzgdockerminecraftbedrockserver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *itzgdockerminecraftbedrockserver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itzgdockerminecraftbedrockserverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("itzg-docker-minecraft-bedrock-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "itzg-docker-minecraft-bedrock-server", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
