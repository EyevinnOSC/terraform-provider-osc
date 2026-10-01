package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/identityschema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &BackupScheduleResource{}
	_ resource.ResourceWithConfigure      = &BackupScheduleResource{}
	_ resource.ResourceWithImportState    = &BackupScheduleResource{}
	_ resource.ResourceWithIdentity       = &BackupScheduleResource{}
	_ resource.ResourceWithValidateConfig = &BackupScheduleResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewBackupScheduleResource)
}

func NewBackupScheduleResource() resource.Resource {
	return &BackupScheduleResource{}
}

// BackupScheduleResource manages the backup policy of a database instance: whether the
// platform backs it up, when, and for how long it keeps the backups.
type BackupScheduleResource struct {
	osaasContext *osaasclient.Context
}

type BackupScheduleResourceModel struct {
	ID               types.String `tfsdk:"id"`
	ServiceID        types.String `tfsdk:"service_id"`
	InstanceName     types.String `tfsdk:"instance_name"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Schedule         types.String `tfsdk:"schedule"`
	RetentionDays    types.Int64  `tfsdk:"retention_days"`
	LastBackupAt     types.String `tfsdk:"last_backup_at"`
	LastAttemptAt    types.String `tfsdk:"last_attempt_at"`
	LastError        types.String `tfsdk:"last_error"`
	CredentialStatus types.String `tfsdk:"credential_status"`
}

type BackupScheduleIdentityModel struct {
	ServiceID    types.String `tfsdk:"service_id"`
	InstanceName types.String `tfsdk:"instance_name"`
}

func (r *BackupScheduleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_schedule"
}

func (r *BackupScheduleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages scheduled backups of a database instance: whether OSC backs it up, when, and how long " +
			"it keeps the backups. Supported services: `" + strings.Join(backupServices, "`, `") + "`. Scheduled " +
			"backups require a paid OSC plan.\n\n" +
			"Backups are stored in OSC-managed object storage (MinIO). The OSC documentation does not say where that " +
			"storage is located relative to the database, for example whether it is in another cluster or region, or " +
			"what happens to the backups when the instance is deleted. For data you cannot lose, also take backups " +
			"you store yourself, for example with periodic exports.\n\n" +
			"OSC has no way to remove a backup schedule. Destroying this resource turns scheduled backups off and " +
			"leaves the schedule and retention in OSC; it deletes no backups. OSC also keeps a schedule after its " +
			"instance is deleted, so destroy the schedule along with the instance, or an instance created later with " +
			"the same name is backed up by it. List backups with the `osc_backups` data source.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier in the form `service_id/instance_name`. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_id": schema.StringAttribute{
				Required:      true,
				Description:   "Service of the instance, one of `" + strings.Join(backupServices, "`, `") + "`. Changing it replaces the schedule.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"instance_name": schema.StringAttribute{
				Required:      true,
				Description:   "Name of the instance to back up. It must exist when the schedule is created. Changing it replaces the schedule.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether scheduled backups run. `false` keeps the schedule and retention in OSC but takes no backups.",
			},
			"schedule": schema.StringAttribute{
				Optional: true,
				Computed: true,
				Default:  stringdefault.StaticString(defaultBackupSchedule),
				Description: "When backups run, as a five-field cron expression in UTC (minute hour day-of-month month " +
					"day-of-week), e.g. `0 3 * * 0` for Sundays at 03:00. Fields take numbers, `*`, ranges, lists and " +
					"steps; month and day names and macros such as `@daily` are not accepted. Defaults to daily at 02:00 UTC.",
			},
			"retention_days": schema.Int64Attribute{
				Optional: true,
				Computed: true,
				Default:  int64default.StaticInt64(defaultBackupRetention),
				Description: "How many days backups are kept. When a scheduled backup completes, OSC deletes the older " +
					"backups of the instance, including backups taken manually. `0` is what the OSC dashboard stores " +
					"for a blank retention, which its documentation describes as keeping backups indefinitely. Defaults to 30.",
			},
			"last_backup_at": schema.StringAttribute{
				Computed: true,
				Description: "When the last backup, scheduled or manual, was started, in RFC 3339. OSC records the time " +
					"whether or not the backup succeeds; the `osc_backups` data source shows its status. Empty if none has been.",
			},
			"last_attempt_at": schema.StringAttribute{
				Computed:    true,
				Description: "When the schedule last ran, in RFC 3339. Empty if it has not.",
			},
			"last_error": schema.StringAttribute{
				Computed:    true,
				Description: "Error of the most recent failed scheduled backup. Empty if it succeeded.",
			},
			"credential_status": schema.StringAttribute{
				Computed: true,
				Description: "Health of the access token OSC stores to run scheduled backups: `ok`, `renewed` (it was " +
					"about to expire and was renewed), or `expired` (backups fail until it is replaced; contact OSC " +
					"support, or disable and enable the schedule). Empty until OSC has checked it.",
			},
		},
	}
}

func (r *BackupScheduleResource) IdentitySchema(_ context.Context, _ resource.IdentitySchemaRequest, resp *resource.IdentitySchemaResponse) {
	resp.IdentitySchema = identityschema.Schema{
		Attributes: map[string]identityschema.Attribute{
			"service_id":    identityschema.StringAttribute{RequiredForImport: true, Description: "The service of the instance."},
			"instance_name": identityschema.StringAttribute{RequiredForImport: true, Description: "The instance name."},
		},
	}
}

func setBackupIdentity(ctx context.Context, identity *tfsdk.ResourceIdentity, model BackupScheduleResourceModel) diag.Diagnostics {
	if identity == nil {
		return nil
	}
	return identity.Set(ctx, BackupScheduleIdentityModel{ServiceID: model.ServiceID, InstanceName: model.InstanceName})
}

func (r *BackupScheduleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *BackupScheduleResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config BackupScheduleResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.ServiceID) && !isBackupService(config.ServiceID.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("service_id"), "Service cannot be backed up",
			fmt.Sprintf("OSC backs up instances of %s, not %q.", strings.Join(backupServices, ", "), config.ServiceID.ValueString()))
	}
	if !config.Schedule.IsNull() && !config.Schedule.IsUnknown() {
		if msg := invalidCron(config.Schedule.ValueString()); msg != "" {
			resp.Diagnostics.AddAttributeError(path.Root("schedule"), "Invalid schedule",
				fmt.Sprintf("Schedule %q %s. OSC stores any schedule without checking it, and takes no backups on one it cannot run.",
					config.Schedule.ValueString(), msg))
		}
	}
	if !config.RetentionDays.IsNull() && !config.RetentionDays.IsUnknown() && config.RetentionDays.ValueInt64() < 0 {
		resp.Diagnostics.AddAttributeError(path.Root("retention_days"), "Invalid retention",
			"retention_days must be 0 or more.")
	}
}

// refresh copies the policy into the model.
func (r *BackupScheduleResource) refresh(p *backupPolicy, model *BackupScheduleResourceModel) {
	model.ID = types.StringValue(p.ServiceID + "/" + p.InstanceName)
	model.ServiceID = types.StringValue(p.ServiceID)
	model.InstanceName = types.StringValue(p.InstanceName)
	model.Enabled = types.BoolValue(p.Enabled)
	model.Schedule = types.StringValue(p.Schedule)
	model.RetentionDays = types.Int64Value(int64(p.RetentionDays))
	model.LastBackupAt = types.StringValue(unixTime(p.LastBackupAt))
	model.LastAttemptAt = types.StringValue(unixTime(p.LastAttemptAt))
	model.LastError = types.StringValue(p.LastError)
	model.CredentialStatus = types.StringValue(p.CredentialStatus)
}

// instanceExists reports whether the instance runs or is suspended. The platform accepts a
// policy for any name, and a policy it accepts can never be removed again.
func instanceExists(ctx *osaasclient.Context, serviceID, name string) (bool, error) {
	service, err := findSubscribedService(ctx, serviceID)
	if err != nil {
		return false, fmt.Errorf("could not read the OSC catalog: %w", err)
	}
	if service != nil {
		token, err := ctx.GetServiceAccessToken(service.ServiceId)
		if err != nil {
			return false, fmt.Errorf("could not get a service access token: %w", err)
		}
		instance, err := getInstance(service, name, token)
		if err != nil {
			return false, err
		}
		if instance != nil {
			return true, nil
		}
	}
	// OSC drops the subscription while every instance of a service is suspended.
	suspended, err := findSuspended(ctx, serviceID, name)
	if err != nil {
		return false, fmt.Errorf("could not read the suspended instances: %w", err)
	}
	return suspended != nil, nil
}

func (r *BackupScheduleResource) write(plan BackupScheduleResourceModel) error {
	return putBackupPolicy(r.osaasContext, plan.ServiceID.ValueString(), plan.InstanceName.ValueString(),
		plan.Enabled.ValueBool(), plan.Schedule.ValueString(), plan.RetentionDays.ValueInt64())
}

// readBack reads the policy just written into state, keeping the plan if it cannot be read.
func (r *BackupScheduleResource) readBack(plan BackupScheduleResourceModel) (BackupScheduleResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	state := plan
	state.ID = types.StringValue(plan.ServiceID.ValueString() + "/" + plan.InstanceName.ValueString())
	p, _, err := readConfirmed(func() (*backupPolicy, bool, error) {
		return found(findBackupPolicy(r.osaasContext, plan.ServiceID.ValueString(), plan.InstanceName.ValueString()))
	})
	if err != nil || p == nil {
		diags.AddError("Failed to read backup schedule", fmt.Sprintf("The schedule of %s was written but could not be read back: %v", state.ID.ValueString(), err))
		for _, v := range []*types.String{&state.LastBackupAt, &state.LastAttemptAt, &state.LastError, &state.CredentialStatus} {
			if v.IsUnknown() {
				*v = types.StringValue("")
			}
		}
		return state, diags
	}
	r.refresh(p, &state)
	return state, diags
}

func (r *BackupScheduleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan BackupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceID, name := plan.ServiceID.ValueString(), plan.InstanceName.ValueString()

	exists, err := instanceExists(r.osaasContext, serviceID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check the instance", fmt.Sprintf("Could not check that instance %q of service %q exists: %s", name, serviceID, err.Error()))
		return
	}
	if !exists {
		resp.Diagnostics.AddAttributeError(path.Root("instance_name"), "Instance not found",
			fmt.Sprintf("Service %q has no instance named %q in this workspace. OSC would accept a schedule for it, and a schedule cannot be removed once written.", serviceID, name))
		return
	}

	if err := r.write(plan); err != nil {
		resp.Diagnostics.AddError("Failed to create backup schedule", fmt.Sprintf("Could not schedule backups of %s/%s: %s", serviceID, name, err.Error()))
		return
	}
	state, diags := r.readBack(plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setBackupIdentity(ctx, resp.Identity, state)...)
}

func (r *BackupScheduleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	p, exists, err := readConfirmed(func() (*backupPolicy, bool, error) {
		return found(findBackupPolicy(r.osaasContext, state.ServiceID.ValueString(), state.InstanceName.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read backup schedule", err.Error())
		return
	}
	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}
	r.refresh(p, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setBackupIdentity(ctx, resp.Identity, state)...)
}

func (r *BackupScheduleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan BackupScheduleResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.write(plan); err != nil {
		resp.Diagnostics.AddError("Failed to update backup schedule",
			fmt.Sprintf("Could not change the backup schedule of %s/%s: %s", plan.ServiceID.ValueString(), plan.InstanceName.ValueString(), err.Error()))
		return
	}
	state, diags := r.readBack(plan)
	resp.Diagnostics.Append(diags...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	resp.Diagnostics.Append(setBackupIdentity(ctx, resp.Identity, state)...)
}

// Delete turns scheduled backups off. OSC cannot remove a schedule, and the backups taken
// so far are kept.
func (r *BackupScheduleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state BackupScheduleResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := disableBackupPolicy(r.osaasContext, state.ServiceID.ValueString(), state.InstanceName.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to disable backup schedule", err.Error())
	}
}

// ImportState accepts the id "service_id/instance_name", or the identity from an import
// block. Only an instance whose schedule has been written can be imported.
func (r *BackupScheduleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var serviceID, name string
	switch {
	case req.ID != "":
		parts := strings.SplitN(req.ID, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			resp.Diagnostics.AddError("Invalid import id",
				fmt.Sprintf("Expected \"service_id/instance_name\", e.g. \"birme-osc-postgresql/mydb\", got %q.", req.ID))
			return
		}
		serviceID, name = parts[0], parts[1]
	case req.Identity != nil:
		var identity BackupScheduleIdentityModel
		resp.Diagnostics.Append(req.Identity.Get(ctx, &identity)...)
		if resp.Diagnostics.HasError() {
			return
		}
		serviceID, name = identity.ServiceID.ValueString(), identity.InstanceName.ValueString()
	}
	if serviceID == "" || name == "" {
		resp.Diagnostics.AddError("Invalid import identity", "Both service_id and instance_name are required.")
		return
	}
	p, err := findBackupPolicy(r.osaasContext, serviceID, name)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read backup schedule", err.Error())
		return
	}
	if p == nil {
		resp.Diagnostics.AddError("Backup schedule not found",
			fmt.Sprintf("Instance %q of service %q has never had a backup schedule. Create one with this resource instead of importing.", name, serviceID))
		return
	}
	var model BackupScheduleResourceModel
	r.refresh(p, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
	resp.Diagnostics.Append(setBackupIdentity(ctx, resp.Identity, model)...)
}
