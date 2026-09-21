package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ datasource.DataSource              = &ServiceDataSource{}
	_ datasource.DataSourceWithConfigure = &ServiceDataSource{}
)

func init() {
	RegisteredDataSources = append(RegisteredDataSources, NewServiceDataSource)
}

func NewServiceDataSource() datasource.DataSource {
	return &ServiceDataSource{}
}

// ServiceDataSource exposes a service's catalog entry, including the parameter schema
// accepted by osc_instance.
type ServiceDataSource struct {
	osaasContext *osaasclient.Context
}

type ServiceDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServiceID   types.String `tfsdk:"service_id"`
	Subscribe   types.Bool   `tfsdk:"subscribe"`
	Subscribed  types.Bool   `tfsdk:"subscribed"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Category    types.String `tfsdk:"category"`
	ServiceType types.String `tfsdk:"service_type"`
	Status      types.String `tfsdk:"status"`
	APIURL      types.String `tfsdk:"api_url"`
	Parameters  types.List   `tfsdk:"parameters"`
}

var serviceParameterAttrTypes = map[string]attr.Type{
	"name":        types.StringType,
	"label":       types.StringType,
	"description": types.StringType,
	"type":        types.StringType,
	"required":    types.BoolType,
	"sensitive":   types.BoolType,
	"default":     types.StringType,
	"enum":        types.ListType{ElemType: types.StringType},
	"pattern":     types.StringType,
}

func (d *ServiceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (d *ServiceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads a service from the Open Source Cloud catalog, including the parameters an " +
			"`osc_instance` of that service accepts.\n\n" +
			"Subscribed services are read live from OSC. Other services are read from the catalog mirror published " +
			"with this provider, which is refreshed weekly; `subscribed` tells which one you got. Set `subscribe = true` " +
			"to subscribe the workspace to the service first, which is what creating an `osc_instance` does anyway.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Same as service_id.",
			},
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "The OSC service id, e.g. `valkey-io-valkey`.",
			},
			"subscribe": schema.BoolAttribute{
				Optional:    true,
				Description: "Subscribe the workspace to the service if it is not already subscribed. Defaults to false.",
			},
			"subscribed":   schema.BoolAttribute{Computed: true, Description: "Whether the workspace is subscribed to the service. When false the data comes from the catalog mirror."},
			"title":        schema.StringAttribute{Computed: true, Description: "Human readable title."},
			"description":  schema.StringAttribute{Computed: true, Description: "Service description."},
			"category":     schema.StringAttribute{Computed: true, Description: "Catalog category."},
			"service_type": schema.StringAttribute{Computed: true, Description: "`instance` for long running services, `job` for run to completion services."},
			"status":       schema.StringAttribute{Computed: true, Description: "Publication status in the catalog."},
			"api_url":      schema.StringAttribute{Computed: true, Description: "Base URL of the service's instance API."},
			"parameters": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Parameters accepted in `osc_instance.parameters` for this service. The `name` option is excluded since it is a top level attribute of `osc_instance`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name":        schema.StringAttribute{Computed: true, Description: "Parameter name, use as the key in `parameters`."},
						"label":       schema.StringAttribute{Computed: true, Description: "Display label."},
						"description": schema.StringAttribute{Computed: true, Description: "What the parameter does."},
						"type":        schema.StringAttribute{Computed: true, Description: "One of string, boolean, enum, list."},
						"required":    schema.BoolAttribute{Computed: true, Description: "Whether the parameter must be set."},
						"sensitive":   schema.BoolAttribute{Computed: true, Description: "Whether the service marks the parameter as sensitive. Put such values in `sensitive_parameters`."},
						"default":     schema.StringAttribute{Computed: true, Description: "Default value suggested by the service, if any."},
						"enum":        schema.ListAttribute{ElementType: types.StringType, Computed: true, Description: "Allowed values for enum parameters."},
						"pattern":     schema.StringAttribute{Computed: true, Description: "Regular expression the value must match, if any."},
					},
				},
			},
		},
	}
}

func (d *ServiceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	osaasContext, ok := req.ProviderData.(*osaasclient.Context)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *osaasclient.Context, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}
	d.osaasContext = osaasContext
}

func (d *ServiceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config ServiceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var service *catalogService
	var err error
	if config.Subscribe.ValueBool() {
		service, err = ensureSubscribed(d.osaasContext, config.ServiceID.ValueString())
		if err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Could not subscribe to service", err.Error())
			return
		}
	} else {
		service, err = findSubscribedService(d.osaasContext, config.ServiceID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read OSC catalog", err.Error())
			return
		}
	}
	subscribed := service != nil
	if service == nil {
		mirrored, suggestions, merr := mirrorService(ctx, config.ServiceID.ValueString())
		if mirrored == nil {
			resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Unknown service",
				unknownServiceDetail(config.ServiceID.ValueString(), suggestions, merr))
			return
		}
		service = mirrored
	}

	params := make([]attr.Value, 0, len(service.ServiceInstanceOptions))
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Name == "name" {
			continue
		}
		enum := make([]attr.Value, 0, len(opt.Enum))
		for _, e := range opt.Enum {
			enum = append(enum, types.StringValue(e))
		}
		obj, diags := types.ObjectValue(serviceParameterAttrTypes, map[string]attr.Value{
			"name":        types.StringValue(opt.Name),
			"label":       types.StringValue(opt.Label),
			"description": types.StringValue(opt.Description),
			"type":        types.StringValue(opt.Type),
			"required":    types.BoolValue(opt.Mandatory),
			"sensitive":   types.BoolValue(opt.Sensitive),
			"default":     types.StringValue(opt.Default),
			"enum":        types.ListValueMust(types.StringType, enum),
			"pattern":     types.StringValue(opt.RegexValidator),
		})
		resp.Diagnostics.Append(diags...)
		params = append(params, obj)
	}
	list, diags := types.ListValue(types.ObjectType{AttrTypes: serviceParameterAttrTypes}, params)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	state := ServiceDataSourceModel{
		ID:          types.StringValue(service.ServiceId),
		ServiceID:   types.StringValue(service.ServiceId),
		Subscribe:   config.Subscribe,
		Subscribed:  types.BoolValue(subscribed),
		Title:       types.StringValue(service.Metadata.Title),
		Description: types.StringValue(service.Metadata.Description),
		Category:    types.StringValue(service.Metadata.Category),
		ServiceType: types.StringValue(service.ServiceType),
		Status:      types.StringValue(service.Status),
		APIURL:      types.StringValue(service.ApiUrl),
		Parameters:  list,
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
