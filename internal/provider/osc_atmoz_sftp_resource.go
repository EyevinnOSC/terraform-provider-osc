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
	_ resource.Resource              = &atmozsftp{}
	_ resource.ResourceWithConfigure = &atmozsftp{}
)

func Newatmozsftp() resource.Resource {
	return &atmozsftp{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newatmozsftp)
}

func (r *atmozsftp) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// atmozsftp is the resource implementation.
type atmozsftp struct {
	osaasContext *osaasclient.Context
}

type atmozsftpModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Username         types.String       `tfsdk:"username"`
	Password         types.String       `tfsdk:"password"`
}

func (r *atmozsftp) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_atmoz_sftp_resource"
}

// Schema defines the schema for the resource.
func (r *atmozsftp) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Effortlessly manage secure file transfers with our user-friendly SFTP server powered by OpenSSH. Ideal for sharing files securely using SSH, it integrates easily with Docker, ensuring both security and simplicity.
`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of sftp",
			},
			"username": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *atmozsftp) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan atmozsftpModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("atmoz-sftp")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "atmoz-sftp", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"Username": plan.Username.ValueString(),
		"Password": plan.Password.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "atmoz-sftp", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := atmozsftpModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Username: plan.Username,
		Password: plan.Password,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *atmozsftp) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *atmozsftp) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *atmozsftp) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state atmozsftpModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("atmoz-sftp")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "atmoz-sftp", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
