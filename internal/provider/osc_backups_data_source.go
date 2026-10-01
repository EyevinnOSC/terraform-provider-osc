package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ datasource.DataSource                   = &BackupsDataSource{}
	_ datasource.DataSourceWithConfigure      = &BackupsDataSource{}
	_ datasource.DataSourceWithValidateConfig = &BackupsDataSource{}
)

func init() {
	RegisteredDataSources = append(RegisteredDataSources, NewBackupsDataSource)
}

func NewBackupsDataSource() datasource.DataSource {
	return &BackupsDataSource{}
}

// BackupsDataSource lists the backups of a database instance.
type BackupsDataSource struct {
	osaasContext *osaasclient.Context
}

type BackupsDataSourceModel struct {
	ID           types.String       `tfsdk:"id"`
	ServiceID    types.String       `tfsdk:"service_id"`
	InstanceName types.String       `tfsdk:"instance_name"`
	Backups      []BackupEntryModel `tfsdk:"backups"`
}

type BackupEntryModel struct {
	Name      types.String `tfsdk:"name"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	Source    types.String `tfsdk:"source"`
	Error     types.String `tfsdk:"error"`
}

func (d *BackupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backups"
}

func (d *BackupsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the backups of a database instance, newest first, both scheduled ones and ones " +
			"taken manually. Use it to find the name of a backup to restore from with the OSC CLI, dashboard or API. " +
			"Schedule backups with `osc_backup_schedule`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, Description: "`service_id/instance_name`."},
			"service_id": schema.StringAttribute{
				Required:    true,
				Description: "Service of the instance, one of `" + strings.Join(backupServices, "`, `") + "`.",
			},
			"instance_name": schema.StringAttribute{Required: true, Description: "Name of the instance."},
			"backups": schema.ListNestedAttribute{
				Computed:    true,
				Description: "The backups, newest first. Empty if the instance has none, or does not exist.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{Computed: true, Description: "Backup name, which identifies it for a restore."},
						"status": schema.StringAttribute{Computed: true, Description: "`Running`, `Complete` or `Failed`. OSC reports some " +
							"finished backups as `SuccessCriteriaMet` or `FailureTarget`, the Kubernetes job conditions that precede " +
							"the final one; they are shown as `Complete` and `Failed`."},
						"created_at": schema.StringAttribute{Computed: true, Description: "When the backup was started, in RFC 3339."},
						"source":     schema.StringAttribute{Computed: true, Description: "`scheduled` or `manual`."},
						"error":      schema.StringAttribute{Computed: true, Description: "Why the backup failed. Empty unless it did."},
					},
				},
			},
		},
	}
}

func (d *BackupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (d *BackupsDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var config BackupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.ServiceID) && !isBackupService(config.ServiceID.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Service cannot be backed up",
			fmt.Sprintf("OSC backs up instances of %s, not %q.", strings.Join(backupServices, ", "), config.ServiceID.ValueString()))
	}
}

func (d *BackupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state BackupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceID, name := state.ServiceID.ValueString(), state.InstanceName.ValueString()
	backups, err := listBackups(d.osaasContext, serviceID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list backups", fmt.Sprintf("Could not list the backups of %s/%s: %s", serviceID, name, err.Error()))
		return
	}
	state.ID = types.StringValue(serviceID + "/" + name)
	state.Backups = make([]BackupEntryModel, 0, len(backups))
	for _, b := range backups {
		state.Backups = append(state.Backups, BackupEntryModel{
			Name:      types.StringValue(b.Name),
			Status:    types.StringValue(backupStatus(b.Status)),
			CreatedAt: types.StringValue(unixTime(b.CreatedAt)),
			Source:    types.StringValue(b.Source),
			Error:     types.StringValue(b.Error),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
