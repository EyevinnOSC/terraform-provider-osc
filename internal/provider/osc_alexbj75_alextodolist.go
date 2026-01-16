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
	_ resource.Resource              = &alexbj75alextodolist{}
	_ resource.ResourceWithConfigure = &alexbj75alextodolist{}
)

func Newalexbj75alextodolist() resource.Resource {
	return &alexbj75alextodolist{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newalexbj75alextodolist)
}

func (r *alexbj75alextodolist) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// alexbj75alextodolist is the resource implementation.
type alexbj75alextodolist struct {
	osaasContext *osaasclient.Context
}

type alexbj75alextodolistModel struct {
	InstanceUrl              types.String   `tfsdk:"instance_url"`
	ServiceId              types.String   `tfsdk:"service_id"`
	ExternalIp				types.String		`tfsdk:"external_ip"`
	ExternalPort			types.Int32	`tfsdk:"external_port"`
	Name         types.String       `tfsdk:"name"`
	Dbhost         types.String       `tfsdk:"db_host"`
	Dbport         types.String       `tfsdk:"db_port"`
	Dbuser         types.String       `tfsdk:"db_user"`
	Dbpassword         types.String       `tfsdk:"db_password"`
	Dbname         types.String       `tfsdk:"db_name"`
}

func (r *alexbj75alextodolist) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_alexbj75_alextodolist"
}

// Schema defines the schema for the resource.
func (r *alexbj75alextodolist) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Boost your productivity with our full-stack Todo List Application! Featuring a sleek UI, robust Node.js backend, and seamless MariaDB integration, it&#39;s the perfect tool for managing tasks effortlessly.`,
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
				Description: "Name of alextodolist",
			},
			"db_host": schema.StringAttribute{
				Required: true,
				Description: "",
			},
			"db_port": schema.StringAttribute{
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
				Required: true,
				Description: "",
			},
		},
	}
}

func (r *alexbj75alextodolist) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alexbj75alextodolistModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-alextodolist")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "alexbj75-alextodolist", serviceAccessToken, map[string]interface{}{
		"name": plan.Name.ValueString(),
		"dbHost": plan.Dbhost.ValueString(),
		"dbPort": plan.Dbport.ValueString(),
		"dbUser": plan.Dbuser.ValueString(),
		"dbPassword": plan.Dbpassword.ValueString(),
		"dbName": plan.Dbname.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "alexbj75-alextodolist", instance["name"].(string), serviceAccessToken)
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
	state := alexbj75alextodolistModel{
		InstanceUrl: types.StringValue(instance["url"].(string)),
		ServiceId: types.StringValue("alexbj75-alextodolist"),
		ExternalIp: types.StringValue(externalIp),
		ExternalPort: types.Int32Value(int32(externalPort)),
		Name: plan.Name,
		Dbhost: plan.Dbhost,
		Dbport: plan.Dbport,
		Dbuser: plan.Dbuser,
		Dbpassword: plan.Dbpassword,
		Dbname: plan.Dbname,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *alexbj75alextodolist) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *alexbj75alextodolist) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *alexbj75alextodolist) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alexbj75alextodolistModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("alexbj75-alextodolist")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "alexbj75-alextodolist", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
