package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ datasource.DataSource                   = &ResourceTagsDataSource{}
	_ datasource.DataSourceWithConfigure      = &ResourceTagsDataSource{}
	_ datasource.DataSourceWithValidateConfig = &ResourceTagsDataSource{}
)

func init() {
	RegisteredDataSources = append(RegisteredDataSources, NewResourceTagsDataSource)
}

func NewResourceTagsDataSource() datasource.DataSource {
	return &ResourceTagsDataSource{}
}

// ResourceTagsDataSource lists the tagged resources of the workspace, for example every
// resource in a project.
type ResourceTagsDataSource struct {
	osaasContext *osaasclient.Context
}

type ResourceTagsDataSourceModel struct {
	ID           types.String              `tfsdk:"id"`
	Tag          types.String              `tfsdk:"tag"`
	ResourceType types.String              `tfsdk:"resource_type"`
	Resources    []TaggedResourceModel     `tfsdk:"resources"`
	Tags         []TagCountDataSourceModel `tfsdk:"tags"`
}

type TaggedResourceModel struct {
	ResourceType types.String `tfsdk:"resource_type"`
	ServiceID    types.String `tfsdk:"service_id"`
	ResourceID   types.String `tfsdk:"resource_id"`
	Tags         types.Set    `tfsdk:"tags"`
}

type TagCountDataSourceModel struct {
	Tag   types.String `tfsdk:"tag"`
	Count types.Int64  `tfsdk:"count"`
}

func (d *ResourceTagsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource_tags"
}

func (d *ResourceTagsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the tagged service instances, My Apps and My Pages of the workspace. A project in OSC " +
			"is a tag, so filtering by `tag` lists every resource in a project. Set tags with the `tags` attribute of " +
			"`osc_instance`, `osc_my_app` and `osc_my_page`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "The filters, for Terraform's use."},
			"tag": schema.StringAttribute{
				Optional:    true,
				Description: "Only list resources carrying this tag. Matched regardless of case.",
			},
			"resource_type": schema.StringAttribute{
				Optional:    true,
				Description: "Only list resources of this type: `instance`, `myapp` or `mypage`.",
			},
			"resources": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The tagged resources matching the filters. Resources without tags are not listed.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"resource_type": schema.StringAttribute{Computed: true, Description: "`instance`, `myapp` or `mypage`."},
						"service_id":    schema.StringAttribute{Computed: true, Description: "The service of an instance; empty for apps and pages."},
						"resource_id":   schema.StringAttribute{Computed: true, Description: "The instance name, app id or page id."},
						"tags":          schema.SetAttribute{Computed: true, ElementType: types.StringType, Description: "The resource's tags."},
					},
				},
			},
			"tags": schema.ListNestedAttribute{
				Computed:    true,
				Description: "Every tag in the workspace, whatever the filters, with the number of resources carrying it.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"tag":   schema.StringAttribute{Computed: true, Description: "The tag."},
						"count": schema.Int64Attribute{Computed: true, Description: "Number of resources carrying it."},
					},
				},
			},
		},
	}
}

func (d *ResourceTagsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (d *ResourceTagsDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config ResourceTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() || !isSet(config.ResourceType) {
		return
	}
	switch config.ResourceType.ValueString() {
	case tagTypeInstance, tagTypeMyApp, tagTypeMyPage:
	default:
		resp.Diagnostics.AddAttributeError(path.Root("resource_type"), "Unknown resource type",
			"resource_type must be instance, myapp or mypage.")
	}
}

func (d *ResourceTagsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state ResourceTagsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	entries, err := listResourceTags(d.osaasContext, state.ResourceType.ValueString(), state.Tag.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to list resource tags", err.Error())
		return
	}
	counts, err := listTagCounts(d.osaasContext)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list tags", err.Error())
		return
	}

	state.ID = types.StringValue(state.ResourceType.ValueString() + "/" + state.Tag.ValueString())
	state.Resources = make([]TaggedResourceModel, 0, len(entries))
	for _, e := range entries {
		state.Resources = append(state.Resources, TaggedResourceModel{
			ResourceType: types.StringValue(e.ResourceType),
			ServiceID:    types.StringValue(e.ServiceID),
			ResourceID:   types.StringValue(e.ResourceID),
			Tags:         tagsToSet(e.Tags),
		})
	}
	state.Tags = make([]TagCountDataSourceModel, 0, len(counts))
	for _, c := range counts {
		state.Tags = append(state.Tags, TagCountDataSourceModel{Tag: types.StringValue(c.Tag), Count: types.Int64Value(int64(c.Count))})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
