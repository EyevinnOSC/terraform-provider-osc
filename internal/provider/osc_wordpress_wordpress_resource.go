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
	_ resource.Resource              = &wordpresswordpress{}
	_ resource.ResourceWithConfigure = &wordpresswordpress{}
)

func Newwordpresswordpress() resource.Resource {
	return &wordpresswordpress{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newwordpresswordpress)
}

func (r *wordpresswordpress) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// wordpresswordpress is the resource implementation.
type wordpresswordpress struct {
	osaasContext *osaasclient.Context
}

type wordpresswordpressModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Dbhost         types.String       `tfsdk:"db_host"`
	Dbuser         types.String       `tfsdk:"db_user"`
	Dbpassword         types.String       `tfsdk:"db_password"`
	Dbname         types.String       `tfsdk:"db_name"`
	Dbtableprefix         types.String       `tfsdk:"db_table_prefix"`
}

func (r *wordpresswordpress) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_wordpress_wordpress_resource"
}

// Schema defines the schema for the resource.
func (r *wordpresswordpress) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Power your site with WordPress – the core behind 40% of the web. Enjoy seamless installation, robust customization, and unmatched scalability. Elevate your online presence effortlessly today!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of wordpress",
			},
			"db_host": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_user": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_password": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_name": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"db_table_prefix": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *wordpresswordpress) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wordpresswordpressModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("wordpress-wordpress")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "wordpress-wordpress", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"DbHost": plan.Dbhost.ValueString(),
		"DbUser": plan.Dbuser.ValueString(),
		"DbPassword": plan.Dbpassword.ValueString(),
		"DbName": plan.Dbname.ValueString(),
		"DbTablePrefix": plan.Dbtableprefix.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "wordpress-wordpress", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := wordpresswordpressModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Dbhost: plan.Dbhost,
		Dbuser: plan.Dbuser,
		Dbpassword: plan.Dbpassword,
		Dbname: plan.Dbname,
		Dbtableprefix: plan.Dbtableprefix,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *wordpresswordpress) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *wordpresswordpress) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *wordpresswordpress) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wordpresswordpressModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("wordpress-wordpress")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "wordpress-wordpress", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
