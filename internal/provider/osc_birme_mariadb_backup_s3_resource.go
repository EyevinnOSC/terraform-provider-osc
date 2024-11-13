package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"

	osaasclient "github.com/eyevinn/osaas-client-go"
)

var (
	_ resource.Resource              = &birmemariadbbackups3{}
	_ resource.ResourceWithConfigure = &birmemariadbbackups3{}
)

func Newbirmemariadbbackups3() resource.Resource {
	return &birmemariadbbackups3{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newbirmemariadbbackups3)
}

func (r *birmemariadbbackups3) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// birmemariadbbackups3 is the resource implementation.
type birmemariadbbackups3 struct {
	osaasContext *osaasclient.Context
}

type birmemariadbbackups3Model struct {
	Name             types.String   `tfsdk:"name"`
	Url              types.String   `tfsdk:"url"`
	Mariadburl         types.String       `tfsdk:"maria_db_url"`
	Cmdlineargs         types.String       `tfsdk:"cmd_line_args"`
	Awsaccesskeyid         types.String       `tfsdk:"aws_access_key_id"`
	Awssecretaccesskey         types.String       `tfsdk:"aws_secret_access_key"`
	Awssessiontoken         types.String       `tfsdk:"aws_session_token"`
	Awsregion         types.String       `tfsdk:"aws_region"`
}

func (r *birmemariadbbackups3) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_birme_mariadb_backup_s3_resource"
}

// Schema defines the schema for the resource.
func (r *birmemariadbbackups3) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				Required: true,
			},
			"url": schema.StringAttribute{
				Computed: true,
			},
			"maria_db_url": schema.StringAttribute{
				Required: true,
			},
			"cmd_line_args": schema.StringAttribute{
				Required: true,
			},
			"aws_access_key_id": schema.StringAttribute{
				Optional: true,
			},
			"aws_secret_access_key": schema.StringAttribute{
				Optional: true,
			},
			"aws_session_token": schema.StringAttribute{
				Optional: true,
			},
			"aws_region": schema.StringAttribute{
				Optional: true,
			},
		},
	}
}

func (r *birmemariadbbackups3) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan birmemariadbbackups3Model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-mariadb-backup-s3")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "birme-mariadb-backup-s3", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"MariaDbUrl": plan.Mariadburl.ValueString(),
		"cmdLineArgs": plan.Cmdlineargs.ValueString(),
		"awsAccessKeyId": plan.Awsaccesskeyid.ValueString(),
		"awsSecretAccessKey": plan.Awssecretaccesskey.ValueString(),
		"awsSessionToken": plan.Awssessiontoken.ValueString(),
		"awsRegion": plan.Awsregion.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "birme-mariadb-backup-s3", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := birmemariadbbackups3Model{
		Name: types.StringValue(instance["name"].(string)),
		Url: types.StringValue(instance["url"].(string)),
		Mariadburl: plan.Mariadburl,
		Cmdlineargs: plan.Cmdlineargs,
		Awsaccesskeyid: plan.Awsaccesskeyid,
		Awssecretaccesskey: plan.Awssecretaccesskey,
		Awssessiontoken: plan.Awssessiontoken,
		Awsregion: plan.Awsregion,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *birmemariadbbackups3) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *birmemariadbbackups3) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *birmemariadbbackups3) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state birmemariadbbackups3Model
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("birme-mariadb-backup-s3")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "birme-mariadb-backup-s3", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
