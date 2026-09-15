package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ datasource.DataSource              = &WorkspaceDataSource{}
	_ datasource.DataSourceWithConfigure = &WorkspaceDataSource{}
)

func init() {
	RegisteredDataSources = append(RegisteredDataSources, NewWorkspaceDataSource)
}

func NewWorkspaceDataSource() datasource.DataSource {
	return &WorkspaceDataSource{}
}

// WorkspaceDataSource reports which workspace the provider's access token belongs to.
type WorkspaceDataSource struct {
	osaasContext *osaasclient.Context
}

type WorkspaceDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Workspace   types.String `tfsdk:"workspace"`
	UserID      types.String `tfsdk:"user_id"`
	Environment types.String `tfsdk:"environment"`
	TokenType   types.String `tfsdk:"token_type"`
	TokenExpiry types.String `tfsdk:"token_expiry"`
}

func (d *WorkspaceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workspace"
}

func (d *WorkspaceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reports the OSC workspace (tenant) and user that the provider's access token belongs to. " +
			"Every OSC access token is bound to exactly one workspace, and all resources are created in that workspace. " +
			"Use this to assert where a configuration will be applied, or set `workspace` on the provider to make the " +
			"provider refuse tokens for any other workspace.",
		Attributes: map[string]schema.Attribute{
			"id":           schema.StringAttribute{Computed: true, Description: "Same as workspace."},
			"workspace":    schema.StringAttribute{Computed: true, Description: "Workspace (tenant) id the token is bound to."},
			"user_id":      schema.StringAttribute{Computed: true, Description: "Id of the user the token was issued to."},
			"environment":  schema.StringAttribute{Computed: true, Description: "OSC environment the provider talks to."},
			"token_type":   schema.StringAttribute{Computed: true, Description: "`pat` for a personal access token, `oauth` for a short lived token from `osc login`."},
			"token_expiry": schema.StringAttribute{Computed: true, Description: "Expiry time of the token in RFC 3339, or empty if it does not expire."},
		},
	}
}

func (d *WorkspaceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkspaceDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	claims, err := decodeTokenClaims(d.osaasContext.GetPersonalAccessToken())
	if err != nil {
		resp.Diagnostics.AddError("Could not decode access token", err.Error())
		return
	}
	tokenType := claims.Type
	if tokenType == "" {
		tokenType = "pat"
	}
	expiry := ""
	if claims.Expires != 0 {
		expiry = time.Unix(claims.Expires, 0).UTC().Format(time.RFC3339)
	}
	state := WorkspaceDataSourceModel{
		ID:          types.StringValue(claims.TenantID),
		Workspace:   types.StringValue(claims.TenantID),
		UserID:      types.StringValue(claims.UserID),
		Environment: types.StringValue(d.osaasContext.GetEnvironment()),
		TokenType:   types.StringValue(tokenType),
		TokenExpiry: types.StringValue(expiry),
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
