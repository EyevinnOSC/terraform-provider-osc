package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var _ provider.Provider = &oscProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &oscProvider{
			version: version,
		}
	}
}

type oscProvider struct {
	version string
}

type oscProviderModel struct {
	Pat         types.String `tfsdk:"pat"`
	Environment types.String `tfsdk:"environment"`
}

func (p *oscProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "osc"
	resp.Version = p.version
}

func (p *oscProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Open Source Cloud Provider",
		Attributes: map[string]schema.Attribute{
			"pat": schema.StringAttribute{
				Sensitive: true,
				Required:  true,
				Description: "Personal Access Token to be used when communicating with the OSC API",
			},
			"environment": schema.StringAttribute{
				Optional: true,
				Description: "Which Environment to use e.g. 'dev' or 'prod'",
			},
		},
	}
}

func (p *oscProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config oscProviderModel
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)

	if config.Pat.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("pat"),
			"Unknown personal access token",
			"The provider cannot create the OSC API client as there is an unknown configuration value for the OSC personal access token. "+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the OSC_ACCESS_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	pat := ""
	if !config.Pat.IsNull() {
		pat = config.Pat.ValueString()
	}

	if pat == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("pat"),
			"Missing OSC personal access token",
			"The provider cannot create the OSC API client as there is an missing configuration value for the OSC personal access token. "+
				"Set the value in the configuration, or use the OSC_PAT environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	environment := ""
	if !config.Environment.IsNull() {
		environment = config.Environment.ValueString()
	}

	osaasConfig := &osaasclient.ContextConfig{
		PersonalAccessToken: pat,
		Environment:         environment,
	}
	client, err := osaasclient.NewContext(osaasConfig)

	if err != nil {
		resp.Diagnostics.AddError("Failed to create OSC client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client

}

func (p *oscProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewEncoreInstanceResource,
		NewValkeyInstanceResource,
		NewEncoreCallbackListenerInstanceResource,
		NewEncoreTransferInstanceResource,
		NewSecretResource,
	}
}

func (p *oscProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}
