package provider

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ provider.Provider                  = &oscProvider{}
	_ provider.ProviderWithListResources = &oscProvider{}
)

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
	Workspace   types.String `tfsdk:"workspace"`
}

func (p *oscProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "osc"
	resp.Version = p.version
}

func (p *oscProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages service instances, secrets, My Apps, My Pages, custom domains and parameter stores in " +
			"[Eyevinn Open Source Cloud](https://www.osaas.io) (OSC). A single generic `osc_instance` resource creates an " +
			"instance of any service in the OSC catalog, with the service's parameters read from the catalog at plan and apply time.",
		Attributes: map[string]schema.Attribute{
			"pat": schema.StringAttribute{
				Sensitive: true,
				Optional:  true,
				MarkdownDescription: "Access token used when communicating with the OSC API. Resolved in this order: this attribute, " +
					"the `OSC_ACCESS_TOKEN` environment variable, then the token saved by `osc login` (`~/.osc/token`, or " +
					"`~/.osc/token-<environment>` for non-prod). Tokens from `osc login` are short lived; use a personal access " +
					"token for CI and agents.",
			},
			"environment": schema.StringAttribute{
				Optional:    true,
				Description: "Which OSC environment to use, e.g. 'dev' or 'prod'. Defaults to 'prod'.",
			},
			"workspace": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "Workspace (tenant) id this configuration targets. Falls back to the `OSC_WORKSPACE` environment " +
					"variable. Every OSC access token is bound to one workspace, and the provider refuses to run if the resolved " +
					"token belongs to a different workspace. When neither is set the plan warns which workspace the token " +
					"belongs to. Set it whenever the user has access to more than one workspace. Use the `osc_workspace` " +
					"data source to see the workspace of the current token.",
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

	if config.Workspace.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("workspace"),
			"Unknown workspace",
			"The workspace must be known at plan time so the provider can verify the access token targets it. "+
				"Set it statically or from a variable rather than from another resource.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	environment := "prod"
	if !config.Environment.IsNull() && config.Environment.ValueString() != "" {
		environment = config.Environment.ValueString()
	}

	configuredPat := ""
	if !config.Pat.IsNull() {
		configuredPat = config.Pat.ValueString()
	}
	pat, source, err := resolveToken(configuredPat, environment)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("pat"), "Missing OSC access token", err.Error())
		return
	}

	claims, err := decodeTokenClaims(pat)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("pat"), "Invalid OSC access token",
			fmt.Sprintf("The token from %s could not be decoded: %s", source, err.Error()))
		return
	}
	if claims.expired(time.Now()) {
		resp.Diagnostics.AddAttributeError(path.Root("pat"), "Expired OSC access token",
			fmt.Sprintf("The token from %s expired at %s. Run `osc login` again, or set OSC_ACCESS_TOKEN to a personal access token for workspace %q.",
				source, time.Unix(claims.Expires, 0).UTC().Format(time.RFC3339), claims.TenantID))
		return
	}
	workspace, workspaceSource := "", ""
	if !config.Workspace.IsNull() && config.Workspace.ValueString() != "" {
		workspace, workspaceSource = config.Workspace.ValueString(), "the provider workspace attribute"
	} else if env := os.Getenv("OSC_WORKSPACE"); env != "" {
		workspace, workspaceSource = env, "the OSC_WORKSPACE environment variable"
	}
	switch {
	case workspace == "":
		resp.Diagnostics.AddAttributeWarning(path.Root("workspace"), "Workspace not pinned",
			fmt.Sprintf("No workspace is set, so this configuration applies to whichever workspace the access token belongs to. "+
				"The token from %s belongs to workspace %q. Set workspace = %q on the provider, or OSC_WORKSPACE, "+
				"to make the provider refuse tokens for other workspaces.", source, claims.TenantID, claims.TenantID))
	case workspace != claims.TenantID:
		resp.Diagnostics.AddAttributeError(path.Root("workspace"), "Access token belongs to a different workspace",
			fmt.Sprintf("This configuration targets workspace %q (from %s) but the token from %s belongs to workspace %q. "+
				"Nothing was changed. Use a token issued for workspace %q, or change the workspace if %q is the intended target.",
				workspace, workspaceSource, source, claims.TenantID, workspace, claims.TenantID))
		return
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
	resp.ListResourceData = client
}

var RegisteredResources []func() resource.Resource
var RegisteredDataSources []func() datasource.DataSource
var RegisteredListResources []func() list.ListResource

func (p *oscProvider) Resources(ctx context.Context) []func() resource.Resource {
	return RegisteredResources
}

func (p *oscProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return RegisteredDataSources
}

func (p *oscProvider) ListResources(ctx context.Context) []func() list.ListResource {
	return RegisteredListResources
}
