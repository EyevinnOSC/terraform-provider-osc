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
	_ resource.Resource              = &mickaelkerjeanfilestash{}
	_ resource.ResourceWithConfigure = &mickaelkerjeanfilestash{}
)

func Newmickaelkerjeanfilestash() resource.Resource {
	return &mickaelkerjeanfilestash{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newmickaelkerjeanfilestash)
}

func (r *mickaelkerjeanfilestash) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// mickaelkerjeanfilestash is the resource implementation.
type mickaelkerjeanfilestash struct {
	osaasContext *osaasclient.Context
}

type mickaelkerjeanfilestashModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Adminpassword         types.String       `tfsdk:"admin_password"`
	Configsecret         types.String       `tfsdk:"config_secret"`
	Dropboxclientid         types.String       `tfsdk:"dropbox_client_id"`
	Gdriveclientid         types.String       `tfsdk:"gdrive_client_id"`
	Gdriveclientsecret         types.String       `tfsdk:"gdrive_client_secret"`
}

func (r *mickaelkerjeanfilestash) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_mickael_kerjean_filestash_resource"
}

// Schema defines the schema for the resource.
func (r *mickaelkerjeanfilestash) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Transform your data management with Filestash, a versatile file manager that integrates seamlessly with multiple cloud services and protocols. Enjoy blazing speed, user-friendly interfaces, and plugin flexibility.`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of filestash",
			},
			"admin_password": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"config_secret": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"dropbox_client_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"gdrive_client_id": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"gdrive_client_secret": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *mickaelkerjeanfilestash) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mickaelkerjeanfilestashModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("mickael-kerjean-filestash")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "mickael-kerjean-filestash", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
		"ConfigSecret": plan.Configsecret.ValueString(),
		"DropboxClientId": plan.Dropboxclientid.ValueString(),
		"GdriveClientId": plan.Gdriveclientid.ValueString(),
		"GdriveClientSecret": plan.Gdriveclientsecret.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "mickael-kerjean-filestash", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := mickaelkerjeanfilestashModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Adminpassword: plan.Adminpassword,
		Configsecret: plan.Configsecret,
		Dropboxclientid: plan.Dropboxclientid,
		Gdriveclientid: plan.Gdriveclientid,
		Gdriveclientsecret: plan.Gdriveclientsecret,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *mickaelkerjeanfilestash) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *mickaelkerjeanfilestash) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *mickaelkerjeanfilestash) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mickaelkerjeanfilestashModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("mickael-kerjean-filestash")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "mickael-kerjean-filestash", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
