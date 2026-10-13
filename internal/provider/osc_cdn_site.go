package provider

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &CDNSiteResource{}
	_ resource.ResourceWithConfigure      = &CDNSiteResource{}
	_ resource.ResourceWithImportState    = &CDNSiteResource{}
	_ resource.ResourceWithValidateConfig = &CDNSiteResource{}
	_ resource.ResourceWithModifyPlan     = &CDNSiteResource{}
)

// cdnSiteTimeout is how long create and delete wait for the site to settle. Provisioning
// at the CDN vendor takes minutes.
const cdnSiteTimeout = 20 * time.Minute

func init() {
	RegisteredResources = append(RegisteredResources, NewCDNSiteResource)
}

func NewCDNSiteResource() resource.Resource {
	return &CDNSiteResource{}
}

// CDNSiteResource puts OSC's CDN in front of a service instance, a My App or a My Page.
type CDNSiteResource struct {
	osaasContext *osaasclient.Context
}

type CDNSiteResourceModel struct {
	ID           types.String `tfsdk:"id"`
	ServiceID    types.String `tfsdk:"service_id"`
	InstanceName types.String `tfsdk:"instance_name"`
	AppID        types.String `tfsdk:"app_id"`
	PageID       types.String `tfsdk:"page_id"`
	CDNURL       types.String `tfsdk:"cdn_url"`
	Status       types.String `tfsdk:"status"`
}

func (r *CDNSiteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cdn_site"
}

func (r *CDNSiteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Puts OSC's CDN in front of a service instance, a My App or a My Page, and serves it at `cdn_url`. " +
			"Requires the PROFESSIONAL plan, and the workspace must be allowed to use the CDN.\n\n" +
			"Set exactly one origin: `service_id` and `instance_name` for a service instance (only services on the platform's " +
			"allowlist, today `minio-minio`, where only public-read buckets are accelerated), `app_id` for a My App, or " +
			"`page_id` for a public My Page.\n\n" +
			"The CDN caches by the origin's cache headers. Do not send presigned URLs through it: it can keep serving them " +
			"after they expire. A site costs tokens per GB served, per 1,000 requests and per day.\n\n" +
			"Every attribute replaces the site when changed. Creating waits until the site is active.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The CDN site id. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Service id of an instance origin, e.g. `minio-minio`. Set with `instance_name`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"instance_name": schema.StringAttribute{
				Optional:      true,
				Description:   "Name of an instance origin. Set with `service_id`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"app_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Id of a My App origin.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"page_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Id of a My Page origin. A page behind basic auth cannot be accelerated.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cdn_url": schema.StringAttribute{
				Computed:      true,
				Description:   "The URL the CDN serves the origin at.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description: "The site's status: `active` once created. A site the platform has `suspended` is resumed " +
					"on the next apply.",
			},
		},
	}
}

func (r *CDNSiteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *CDNSiteResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config CDNSiteResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	for _, v := range []types.String{config.ServiceID, config.InstanceName, config.AppID, config.PageID} {
		if v.IsUnknown() {
			return
		}
	}
	if isSet(config.ServiceID) != isSet(config.InstanceName) {
		resp.Diagnostics.AddAttributeError(path.Root("instance_name"), "Incomplete instance origin",
			"Set both service_id and instance_name, or neither.")
		return
	}
	n := 0
	for _, set := range []bool{isSet(config.ServiceID), isSet(config.AppID), isSet(config.PageID)} {
		if set {
			n++
		}
	}
	if n != 1 {
		resp.Diagnostics.AddError("Exactly one origin",
			"Set service_id and instance_name, or app_id, or page_id: exactly one origin per CDN site.")
	}
}

func (m *CDNSiteResourceModel) origin() cdnOrigin {
	switch {
	case isSet(m.AppID):
		return cdnOrigin{Kind: "myapp", AppID: m.AppID.ValueString()}
	case isSet(m.PageID):
		return cdnOrigin{Kind: "mypage", PageID: m.PageID.ValueString()}
	default:
		return cdnOrigin{Kind: "instance", ServiceID: m.ServiceID.ValueString(), InstanceName: m.InstanceName.ValueString()}
	}
}

func (m *CDNSiteResourceModel) refresh(site *cdnSite) {
	m.ID = types.StringValue(site.SiteID)
	m.Status = types.StringValue(site.Status)
	if site.CDNURL != "" {
		m.CDNURL = types.StringValue(site.CDNURL)
	}
	optional := func(v string) types.String {
		if v == "" {
			return types.StringNull()
		}
		return types.StringValue(v)
	}
	m.ServiceID = optional(site.Origin.ServiceID)
	m.InstanceName = optional(site.Origin.InstanceName)
	m.AppID = optional(site.Origin.AppID)
	m.PageID = optional(site.Origin.PageID)
}

func (r *CDNSiteResource) waitActive(id string) (*cdnSite, error) {
	return pollCDNSite(func() (*cdnSite, error) { return getCDNSite(r.osaasContext, id) }, id, cdnSiteActive, cdnSiteTimeout, 10*time.Second)
}

func (r *CDNSiteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan CDNSiteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	site, err := createCDNSite(r.osaasContext, plan.origin())
	if err != nil {
		resp.Diagnostics.AddError("Failed to create CDN site",
			fmt.Sprintf("%s. A 402 means the plan does not include the CDN, a 403 that the workspace or service is not allowed, "+
				"a 409 that the origin already has a site or is protected.", err.Error()))
		return
	}
	// Keep the site in state while it provisions, so a failed wait does not leave it untracked.
	plan.refresh(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	active, err := r.waitActive(site.SiteID)
	if active != nil {
		plan.refresh(active)
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
	if err != nil {
		resp.Diagnostics.AddError("CDN site did not become active", err.Error())
	}
}

func (r *CDNSiteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state CDNSiteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	site, exists, err := readConfirmed(func() (*cdnSite, bool, error) {
		return found(getCDNSite(r.osaasContext, state.ID.ValueString()))
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to read CDN site", err.Error())
		return
	}
	if !exists || site.Status == "deleting" {
		resp.State.RemoveResource(ctx)
		return
	}
	state.refresh(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only runs when status alone differs from "active", since every configurable
// attribute replaces the site: a suspended site is resumed, a provisioning one waited for.
func (r *CDNSiteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state CDNSiteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if state.Status.ValueString() == "suspended" {
		if err := resumeCDNSite(r.osaasContext, id); err != nil {
			resp.Diagnostics.AddError("Failed to resume CDN site", err.Error())
			return
		}
	}
	site, err := r.waitActive(id)
	if site != nil {
		state.refresh(site)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if err != nil {
		resp.Diagnostics.AddError("CDN site did not become active", err.Error())
	}
}

// ModifyPlan plans status "active" for an existing site, so a site that is not active
// shows as a change and Update resumes or waits for it.
func (r *CDNSiteResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	var plan CDNSiteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Status = types.StringValue("active")
	resp.Diagnostics.Append(resp.Plan.Set(ctx, &plan)...)
}

func (r *CDNSiteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state CDNSiteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()
	if err := deleteCDNSite(r.osaasContext, id); err != nil {
		resp.Diagnostics.AddError("Failed to delete CDN site", err.Error())
		return
	}
	// Wait until it is gone: a replacement for the same origin is refused while it exists.
	if _, err := pollCDNSite(func() (*cdnSite, error) { return getCDNSite(r.osaasContext, id) }, id, cdnSiteGone, cdnSiteTimeout, 10*time.Second); err != nil {
		resp.Diagnostics.AddError("CDN site was not deleted", err.Error())
	}
}

func (r *CDNSiteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	site, err := getCDNSite(r.osaasContext, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read CDN site", err.Error())
		return
	}
	if site == nil {
		resp.Diagnostics.AddError("CDN site not found", fmt.Sprintf("No CDN site %q in this workspace.", req.ID))
		return
	}
	var model CDNSiteResourceModel
	model.refresh(site)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
