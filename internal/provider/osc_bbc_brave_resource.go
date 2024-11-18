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
	_ resource.Resource              = &bbcbrave{}
	_ resource.ResourceWithConfigure = &bbcbrave{}
)

func Newbbcbrave() resource.Resource {
	return &bbcbrave{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbbcbrave)
}

func (r *bbcbrave) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// bbcbrave is the resource implementation.
type bbcbrave struct {
	osaasContext *osaasclient.Context
}

type bbcbraveModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Stunserver         types.String       `tfsdk:"stun_server"`
	Turnserver         types.String       `tfsdk:"turn_server"`
}

func (r *bbcbrave) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_bbc_brave_resource"
}

// Schema defines the schema for the resource.
func (r *bbcbrave) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Brave is a Basic real-time (remote) audio&#x2F;video editor. It allows LIVE video (and&#x2F;or audio) to be received, manipulated, and sent elsewhere. Forwarding RTMP from one place to another, mixing two or more inputs or add basic graphics are some example of usage.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of brave",
			},
			"stun_server": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"turn_server": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *bbcbrave) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan bbcbraveModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("bbc-brave")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "bbc-brave", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"StunServer": plan.Stunserver.ValueString(),
		"TurnServer": plan.Turnserver.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "bbc-brave", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := bbcbraveModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Stunserver: plan.Stunserver,
		Turnserver: plan.Turnserver,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *bbcbrave) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *bbcbrave) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *bbcbrave) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state bbcbraveModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("bbc-brave")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "bbc-brave", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
