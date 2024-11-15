// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	osaasclient "github.com/eyevinn/osaas-client-go"
)

// Ensure ScaffoldingProvider satisfies various provider interfaces.
var _ provider.Provider = &oscProvider{}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &oscProvider{
			version: version,
		}
	}
}

// ScaffoldingProvider defines the provider implementation.
type oscProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
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
		Attributes: map[string]schema.Attribute{
			"pat": schema.StringAttribute{
				Sensitive: true,
				Required:  true,
			},
			"environment": schema.StringAttribute{
				Optional: true,
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
		NewRetransferResource,
		NewSecretResource,
	}
}

func (p *oscProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return nil
}
