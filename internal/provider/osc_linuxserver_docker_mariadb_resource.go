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
	_ resource.Resource              = &linuxserverdockermariadb{}
	_ resource.ResourceWithConfigure = &linuxserverdockermariadb{}
)

func Newlinuxserverdockermariadb() resource.Resource {
	return &linuxserverdockermariadb{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newlinuxserverdockermariadb)
}

func (r *linuxserverdockermariadb) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// linuxserverdockermariadb is the resource implementation.
type linuxserverdockermariadb struct {
	osaasContext *osaasclient.Context
}

type linuxserverdockermariadbModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	Name         types.String       `tfsdk:"name"`
	Rootpassword         types.String       `tfsdk:"root_password"`
	Database         types.String       `tfsdk:"database"`
	User         types.String       `tfsdk:"user"`
	Password         types.String       `tfsdk:"password"`
}

func (r *linuxserverdockermariadb) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_linuxserver_docker_mariadb_resource"
}

// Schema defines the schema for the resource.
func (r *linuxserverdockermariadb) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Unlock the full potential of your database management with LinuxServer.io&#39;s MariaDB Docker container. Featuring seamless updates, security enhancements, and multi-platform support, it&#39;s the ideal solution for efficient and reliable data storage. Minimize downtime and bandwidth usage, and maximize your productivity. Transform your database experience now!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed: true,
				Description: "URL to the created instace",
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Name of database server",
			},
			"root_password": schema.StringAttribute{
				Required: true,
				Description: "Administrator password for database server",
			},
			"database": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"user": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
			"password": schema.StringAttribute{
				Optional: true,
				Description: "",
			},
		},
	}
}

func (r *linuxserverdockermariadb) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan linuxserverdockermariadbModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("linuxserver-docker-mariadb")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "linuxserver-docker-mariadb", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"RootPassword": plan.Rootpassword.ValueString(),
		"Database": plan.Database.ValueString(),
		"User": plan.User.ValueString(),
		"Password": plan.Password.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	// ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "linuxserver-docker-mariadb", instance["name"].(string), serviceAccessToken)
	// if err != nil {
	// 	resp.Diagnostics.AddError("Failed to get ports for service", err.Error())
	// 	return
	// }
	// _ = ports

	// Update the state with the actual data returned from the API
	state := linuxserverdockermariadbModel{
		InstanceUrl: types.StringValue(instance["instance_url"].(string)),
		Name: plan.Name,
		Rootpassword: plan.Rootpassword,
		Database: plan.Database,
		User: plan.User,
		Password: plan.Password,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *linuxserverdockermariadb) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *linuxserverdockermariadb) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *linuxserverdockermariadb) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state linuxserverdockermariadbModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("linuxserver-docker-mariadb")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "linuxserver-docker-mariadb", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
