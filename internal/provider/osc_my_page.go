package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                   = &MyPageResource{}
	_ resource.ResourceWithConfigure      = &MyPageResource{}
	_ resource.ResourceWithImportState    = &MyPageResource{}
	_ resource.ResourceWithValidateConfig = &MyPageResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewMyPageResource)
}

func NewMyPageResource() resource.Resource {
	return &MyPageResource{}
}

// MyPageResource manages a My Page static site. Terraform owns the site, its domain and
// its access control; the files are published by whatever builds them.
type MyPageResource struct {
	osaasContext *osaasclient.Context
}

type MyPageResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	CustomDomain      types.String `tfsdk:"custom_domain"`
	BasicAuthUsername types.String `tfsdk:"basic_auth_username"`
	BasicAuthPassword types.String `tfsdk:"basic_auth_password"`
	URL               types.String `tfsdk:"url"`
	Status            types.String `tfsdk:"status"`
}

var myPageName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,50}[a-z0-9]$`)

func (r *MyPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_my_page"
}

func (r *MyPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a My Page static site, served at `https://<name>.pages.osaas.io/`.\n\n" +
			"Terraform creates the site and manages its custom domain and basic auth. It does not manage the files: " +
			"publish them from CI with the OSC CLI or the My Page API (upload, then publish; publish with prune to " +
			"replace a release). A new site is a draft with nothing served until the first publish.\n\n" +
			"Destroying the resource deletes the site and every file in it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "The page id. Use it, or the name, with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Site name and subdomain: 3-52 lowercase letters, digits and hyphens, starting and ending with a " +
					"letter or digit. Names are unique across all of OSC, not per workspace, and some are reserved. " +
					"Changing it replaces the site.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"custom_domain": schema.StringAttribute{
				Optional: true,
				Description: "A fully qualified domain to serve the site at, in addition to its pages.osaas.io URL. Point a " +
					"CNAME at the site's hostname before applying so the certificate can be issued.",
			},
			"basic_auth_username": schema.StringAttribute{
				Optional:    true,
				Description: "Require HTTP basic auth with this username: 1-64 letters, digits, dots, underscores and hyphens. Requires `basic_auth_password`.",
			},
			"basic_auth_password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Basic auth password, 12-72 bytes. Stored hashed by OSC and never readable, so changes made outside Terraform are not detected.",
			},
			"url": schema.StringAttribute{
				Computed:      true,
				Description:   "The site's public URL.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "`draft` until files have been published, then `live`.",
			},
		},
	}
}

func (r *MyPageResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func (r *MyPageResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var config MyPageResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if isSet(config.Name) && !myPageName.MatchString(config.Name.ValueString()) {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Invalid page name",
			"A page name is 3-52 lowercase letters, digits and hyphens, and starts and ends with a letter or digit.")
	}
	if config.BasicAuthUsername.IsUnknown() || config.BasicAuthPassword.IsUnknown() {
		return
	}
	if isSet(config.BasicAuthUsername) != isSet(config.BasicAuthPassword) {
		resp.Diagnostics.AddAttributeError(path.Root("basic_auth_password"), "Incomplete basic auth",
			"Set both basic_auth_username and basic_auth_password, or neither.")
	}
	if n := len(config.BasicAuthPassword.ValueString()); isSet(config.BasicAuthPassword) && (n < 12 || n > 72) {
		resp.Diagnostics.AddAttributeError(path.Root("basic_auth_password"), "Invalid basic auth password",
			"The password must be 12-72 bytes.")
	}
}

func (r *MyPageResource) refresh(page *myPage, model *MyPageResourceModel) {
	model.ID = types.StringValue(page.id())
	if page.Name != "" {
		model.Name = types.StringValue(page.Name)
	}
	url := page.URL
	if url == "" {
		url = fmt.Sprintf("https://%s.pages.osaas.io/", model.Name.ValueString())
	}
	model.URL = types.StringValue(url)
	model.Status = types.StringValue(page.Status)
	// Only a reported domain is recorded: a detach made outside Terraform is not detected.
	if d := page.customDomain(); d != "" {
		model.CustomDomain = types.StringValue(d)
	}
}

func (r *MyPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan MyPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	page, err := createMyPage(r.osaasContext, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("name"), "Failed to create page",
			fmt.Sprintf("Could not create page %q: %s. Page names are unique across all of OSC; a 409 means the name is taken.", plan.Name.ValueString(), err.Error()))
		return
	}
	if page.id() == "" {
		page.Name = plan.Name.ValueString()
	}

	state := plan
	r.refresh(page, &state)
	state.CustomDomain = plan.CustomDomain
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)

	if isSet(plan.CustomDomain) {
		if err := setMyPageDomain(r.osaasContext, page.id(), plan.CustomDomain.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("custom_domain"), "Failed to attach custom domain", err.Error())
			return
		}
	}
	if isSet(plan.BasicAuthUsername) {
		if err := setMyPageAuth(r.osaasContext, page.id(), plan.BasicAuthUsername.ValueString(), plan.BasicAuthPassword.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("basic_auth_username"), "Failed to enable basic auth", err.Error())
			return
		}
	}

	if current, err := getMyPage(r.osaasContext, page.id()); err == nil && current != nil {
		r.refresh(current, &state)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MyPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state MyPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	page, err := getMyPage(r.osaasContext, state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read page", err.Error())
		return
	}
	if page == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	r.refresh(page, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *MyPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state MyPageResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := state.ID.ValueString()

	if plan.CustomDomain.ValueString() != state.CustomDomain.ValueString() {
		// A page has at most one custom domain; replace it by detaching the old one first.
		if isSet(state.CustomDomain) {
			if err := setMyPageDomain(r.osaasContext, id, ""); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("custom_domain"), "Failed to detach custom domain", err.Error())
				return
			}
		}
		if isSet(plan.CustomDomain) {
			if err := setMyPageDomain(r.osaasContext, id, plan.CustomDomain.ValueString()); err != nil {
				resp.Diagnostics.AddAttributeError(path.Root("custom_domain"), "Failed to attach custom domain", err.Error())
				return
			}
		}
	}

	if !plan.BasicAuthUsername.Equal(state.BasicAuthUsername) || !plan.BasicAuthPassword.Equal(state.BasicAuthPassword) {
		if err := setMyPageAuth(r.osaasContext, id, plan.BasicAuthUsername.ValueString(), plan.BasicAuthPassword.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("basic_auth_username"), "Failed to change basic auth", err.Error())
			return
		}
	}

	newState := plan
	newState.ID = state.ID
	newState.URL = state.URL
	newState.Status = state.Status
	if page, err := getMyPage(r.osaasContext, id); err == nil && page != nil {
		r.refresh(page, &newState)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &newState)...)
}

func (r *MyPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state MyPageResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteMyPage(r.osaasContext, state.ID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete page", err.Error())
	}
}

// ImportState takes the page id or name. Basic auth credentials cannot be read back, so
// they start out unset; set them in the configuration to manage them.
func (r *MyPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	page, err := getMyPage(r.osaasContext, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read page", err.Error())
		return
	}
	if page == nil {
		resp.Diagnostics.AddError("Page not found", fmt.Sprintf("The workspace has no page %q.", req.ID))
		return
	}
	model := MyPageResourceModel{
		Name:              types.StringValue(req.ID),
		CustomDomain:      types.StringNull(),
		BasicAuthUsername: types.StringNull(),
		BasicAuthPassword: types.StringNull(),
	}
	r.refresh(page, &model)
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
