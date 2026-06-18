package provider

import (
	"context"
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource              = &apacheairflow{}
	_ resource.ResourceWithConfigure = &apacheairflow{}
)

func Newapacheairflow() resource.Resource {
	return &apacheairflow{}
}

func init() {
	RegisteredResources = append(RegisteredResources, Newapacheairflow)
}

func (r *apacheairflow) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// apacheairflow is the resource implementation.
type apacheairflow struct {
	osaasContext *osaasclient.Context
}

type apacheairflowModel struct {
	InstanceUrl   types.String `tfsdk:"instance_url"`
	ServiceId     types.String `tfsdk:"service_id"`
	ExternalIp    types.String `tfsdk:"external_ip"`
	ExternalPort  types.Int32  `tfsdk:"external_port"`
	Name          types.String `tfsdk:"name"`
	Adminpassword types.String `tfsdk:"admin_password"`
	Databaseurl   types.String `tfsdk:"database_url"`
}

func (r *apacheairflow) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "osc_apache_airflow"
}

// Schema defines the schema for the resource.
func (r *apacheairflow) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: `Discover Apache Airflow, the ultimate platform for programmatically authoring, scheduling, and monitoring workflows. Transform complex tasks into manageable, streamlined operations with dynamic and extensible DAGs. Enhance your workflow efficiency today!`,
		Attributes: map[string]schema.Attribute{
			"instance_url": schema.StringAttribute{
				Computed:    true,
				Description: "URL to the created instace",
			},
			"service_id": schema.StringAttribute{
				Computed:    true,
				Description: "The service id for the created instance",
			},
			"external_ip": schema.StringAttribute{
				Computed:    true,
				Description: "The external Ip of the created instance (if available).",
			},
			"external_port": schema.Int32Attribute{
				Computed:    true,
				Description: "The external Port of the created instance (if available).",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of airflow",
			},
			"admin_password": schema.StringAttribute{
				Optional:    true,
				Description: "Password for the administrative user account in Apache Airflow. This is typically used to access the web UI and perform administrative operations.",
			},
			"database_url": schema.StringAttribute{
				Optional:    true,
				Description: "Connection string for the metadata database that Airflow uses to store DAG information, task states, and other operational data. Supports PostgreSQL, MySQL, and SQLite databases.",
			},
		},
	}
}

func (r *apacheairflow) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apacheairflowModel
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("apache-airflow")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	instance, err := osaasclient.CreateInstance(r.osaasContext, "apache-airflow", serviceAccessToken, map[string]interface{}{
		"name":          plan.Name.ValueString(),
		"AdminPassword": plan.Adminpassword.ValueString(),
		"DatabaseUrl":   plan.Databaseurl.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create instance", err.Error())
		return
	}

	ports, err := osaasclient.GetPortsForInstance(r.osaasContext, "apache-airflow", instance["name"].(string), serviceAccessToken)
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
	state := apacheairflowModel{
		InstanceUrl:   types.StringValue(instance["url"].(string)),
		ServiceId:     types.StringValue("apache-airflow"),
		ExternalIp:    types.StringValue(externalIp),
		ExternalPort:  types.Int32Value(int32(externalPort)),
		Name:          plan.Name,
		Adminpassword: plan.Adminpassword,
		Databaseurl:   plan.Databaseurl,
	}

	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

}

// Read refreshes the Terraform state with the latest data.
func (r *apacheairflow) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *apacheairflow) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *apacheairflow) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apacheairflowModel
	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceAccessToken, err := r.osaasContext.GetServiceAccessToken("apache-airflow")
	if err != nil {
		resp.Diagnostics.AddError("Failed to get service access token", err.Error())
		return
	}

	err = osaasclient.RemoveInstance(r.osaasContext, "apache-airflow", state.Name.ValueString(), serviceAccessToken)
	if err != nil {
		resp.Diagnostics.AddError("Failed to delete instance", err.Error())
		return
	}
}
