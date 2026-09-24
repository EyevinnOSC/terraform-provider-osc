package provider

import (
	"context"
	"errors"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &MailboxResource{}
	_ resource.ResourceWithConfigure      = &MailboxResource{}
	_ resource.ResourceWithImportState    = &MailboxResource{}
	_ resource.ResourceWithValidateConfig = &MailboxResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewMailboxResource)
}

func NewMailboxResource() resource.Resource {
	return &MailboxResource{}
}

// MailboxResource manages the workspace mailbox. A workspace has at most one, so the
// resource has no name of its own; its id is the tenant id.
type MailboxResource struct {
	osaasContext *osaasclient.Context
}

type MailboxResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Password       types.String `tfsdk:"password"`
	Email          types.String `tfsdk:"email"`
	SMTPHost       types.String `tfsdk:"smtp_host"`
	SMTPPort       types.Int64  `tfsdk:"smtp_port"`
	SMTPEncryption types.String `tfsdk:"smtp_encryption"`
	IMAPHost       types.String `tfsdk:"imap_host"`
	IMAPPort       types.Int64  `tfsdk:"imap_port"`
	IMAPEncryption types.String `tfsdk:"imap_encryption"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (r *MailboxResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mailbox"
}

func (r *MailboxResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	computed := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			Computed:      true,
			Description:   description,
			PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		}
	}
	port := func(description string) schema.Int64Attribute {
		return schema.Int64Attribute{
			Computed:      true,
			Description:   description,
			PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the workspace mailbox, an address at `{tenantId}@users.osaas.io` with SMTP and IMAP " +
			"access. It is how a My App sends mail on OSC: pass the SMTP attributes and the password to the app through " +
			"`osc_parameter`, and send as `email`. The platform relays the mail; it cannot send as another domain.\n\n" +
			"A workspace has at most one mailbox, and creating one requires a paid plan. To manage a mailbox that already " +
			"exists, import it.\n\n" +
			"Destroying the mailbox deletes every mail stored in it.",
		Attributes: map[string]schema.Attribute{
			"id": computed("The tenant id. Use any value with `terraform import`; the workspace's mailbox is imported."),
			"password": schema.StringAttribute{
				Required:  true,
				Sensitive: true,
				Description: "Password for SMTP and IMAP, at least 12 characters. Changing it changes the password in place; " +
					"the platform allows five changes an hour, and a new password takes a few seconds to take effect.",
			},
			"email":           computed("The mailbox address, and the only sender address its SMTP server accepts. It is also the SMTP and IMAP user name."),
			"smtp_host":       computed("SMTP server hostname."),
			"smtp_port":       port("SMTP server port."),
			"smtp_encryption": computed("SMTP encryption, as the platform names it, e.g. `STARTTLS`."),
			"imap_host":       computed("IMAP server hostname."),
			"imap_port":       port("IMAP server port."),
			"imap_encryption": computed("IMAP encryption, as the platform names it."),
			"created_at":      computed("When the mailbox was created (ISO 8601)."),
		},
	}
}

func (r *MailboxResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *MailboxResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config MailboxResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.Password) && len(config.Password.ValueString()) < 12 {
		resp.Diagnostics.AddAttributeError(path.Root("password"), "Mailbox password too short",
			"The platform requires a mailbox password of at least 12 characters.")
	}
}

// applyMailbox copies what the platform reports into the model. The password is never
// returned, so it is left as it is.
func applyMailbox(m *mailbox, model *MailboxResourceModel) {
	model.ID = types.StringValue(m.TenantID)
	model.Email = types.StringValue(m.Email)
	model.SMTPHost = types.StringValue(m.SMTP.Server)
	model.SMTPPort = types.Int64Value(int64(m.SMTP.Port))
	model.SMTPEncryption = types.StringValue(m.SMTP.Encryption)
	model.IMAPHost = types.StringValue(m.IMAP.Server)
	model.IMAPPort = types.Int64Value(int64(m.IMAP.Port))
	model.IMAPEncryption = types.StringValue(m.IMAP.Encryption)
	model.CreatedAt = types.StringValue(m.CreatedAt)
}

func (r *MailboxResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MailboxResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	created, err := createMailbox(r.osaasContext, plan.Password.ValueString())
	if err != nil {
		var ae *apiError
		if errors.As(err, &ae) && ae.StatusCode == http.StatusConflict {
			resp.Diagnostics.AddError("The workspace already has a mailbox",
				"A workspace has at most one mailbox. Bring it under Terraform with `terraform import <address> mailbox`; "+
					"the password in the configuration is then set on the next apply.")
			return
		}
		resp.Diagnostics.AddError("Failed to create mailbox", err.Error())
		return
	}
	state := plan
	applyMailbox(created, &state)
	// The create response may leave out fields; the mailbox itself has them all.
	if m, err := getMailbox(r.osaasContext); err == nil && m != nil {
		applyMailbox(m, &state)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MailboxResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MailboxResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	m, err := getMailbox(r.osaasContext)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read mailbox", err.Error())
		return
	}
	if m == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	applyMailbox(m, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MailboxResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MailboxResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The password is the only attribute that can change.
	if !plan.Password.Equal(state.Password) {
		if err := setMailboxPassword(r.osaasContext, plan.Password.ValueString()); err != nil {
			resp.Diagnostics.AddError("Failed to change mailbox password", err.Error())
			return
		}
	}
	state.Password = plan.Password
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MailboxResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	if err := deleteMailbox(r.osaasContext); err != nil {
		resp.Diagnostics.AddError("Failed to delete mailbox", err.Error())
	}
}

// ImportState imports the workspace's mailbox whatever id is given. The password cannot
// be read, so it stays unknown to Terraform until the next apply sets it.
func (r *MailboxResource) ImportState(ctx context.Context, _ resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	m, err := getMailbox(r.osaasContext)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read mailbox", err.Error())
		return
	}
	if m == nil {
		resp.Diagnostics.AddError("Mailbox not found", "The workspace has no mailbox to import.")
		return
	}
	model := MailboxResourceModel{Password: types.StringNull()}
	applyMailbox(m, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
