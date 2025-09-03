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
	_ resource.Resource              = &itzgdockerminecraftserver{}
	_ resource.ResourceWithConfigure = &itzgdockerminecraftserver{}
)

func Newitzgdockerminecraftserver() resource.Resource {
	return &itzgdockerminecraftserver{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newitzgdockerminecraftserver)
}

func (r *itzgdockerminecraftserver) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// itzgdockerminecraftserver is the resource implementation.
type itzgdockerminecraftserver struct {
	osaasContext *osaasclient.Context
}

type itzgdockerminecraftserverModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Accepteula         bool       `tfsdk:"accept_eula"`
	Rconpassword         types.String       `tfsdk:"rcon_password"`
	Mode         types.String       `tfsdk:"mode"`
	Difficulty         types.String       `tfsdk:"difficulty"`
	Maxworldsize         types.String       `tfsdk:"max_world_size"`
	Allownether         bool       `tfsdk:"allow_nether"`
	Announceplayerachievements         bool       `tfsdk:"announce_player_achievements"`
	Enablecommandblock         bool       `tfsdk:"enable_command_block"`
	Forcegamemode         bool       `tfsdk:"force_gamemode"`
	Generalstructures         bool       `tfsdk:"general_structures"`
	Hardcore         bool       `tfsdk:"hardcore"`
	Spawnanimals         bool       `tfsdk:"spawn_animals"`
	Spawnmonsters         bool       `tfsdk:"spawn_monsters"`
	Spawnnpcs         bool       `tfsdk:"spawn_npcs"`
}

func (r *itzgdockerminecraftserver) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_itzg_docker_minecraft_server"
}

// Schema defines the schema for the resource.
func (r *itzgdockerminecraftserver) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Experience seamless Minecraft server management with our Docker solution! Easily deploy, customize, and scale your servers with robust support for different versions, mods, and plugins. Perfect for dedicated gamers and server admins alike!`,
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
				Description: "Name of docker-minecraft-server",
			},
			"accept_eula": schema.BoolAttribute{
				Required: true,
				Description: "",
			},
			"rcon_password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"mode": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"difficulty": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"max_world_size": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"allow_nether": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"announce_player_achievements": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"enable_command_block": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"force_gamemode": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"general_structures": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"hardcore": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"spawn_animals": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"spawn_monsters": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
			"spawn_npcs": schema.BoolAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *itzgdockerminecraftserver) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan itzgdockerminecraftserverModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("itzg-docker-minecraft-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "itzg-docker-minecraft-server", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AcceptEula": plan.Accepteula,
		"RconPassword": plan.Rconpassword.ValueString(),
		"Mode": plan.Mode,
		"Difficulty": plan.Difficulty,
		"MaxWorldSize": plan.Maxworldsize.ValueString(),
		"AllowNether": plan.Allownether,
		"AnnouncePlayerAchievements": plan.Announceplayerachievements,
		"EnableCommandBlock": plan.Enablecommandblock,
		"ForceGamemode": plan.Forcegamemode,
		"GeneralStructures": plan.Generalstructures,
		"Hardcore": plan.Hardcore,
		"SpawnAnimals": plan.Spawnanimals,
		"SpawnMonsters": plan.Spawnmonsters,
		"SpawnNpcs": plan.Spawnnpcs,
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "itzg-docker-minecraft-server", instance["name"].(string), serviceAccessToken)
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
	state := itzgdockerminecraftserverModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("itzg-docker-minecraft-server"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Accepteula: plan.Accepteula,
		Rconpassword: plan.Rconpassword,
		Mode: plan.Mode,
		Difficulty: plan.Difficulty,
		Maxworldsize: plan.Maxworldsize,
		Allownether: plan.Allownether,
		Announceplayerachievements: plan.Announceplayerachievements,
		Enablecommandblock: plan.Enablecommandblock,
		Forcegamemode: plan.Forcegamemode,
		Generalstructures: plan.Generalstructures,
		Hardcore: plan.Hardcore,
		Spawnanimals: plan.Spawnanimals,
		Spawnmonsters: plan.Spawnmonsters,
		Spawnnpcs: plan.Spawnnpcs,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *itzgdockerminecraftserver) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *itzgdockerminecraftserver) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *itzgdockerminecraftserver) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state itzgdockerminecraftserverModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("itzg-docker-minecraft-server")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "itzg-docker-minecraft-server", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
