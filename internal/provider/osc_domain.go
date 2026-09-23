package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

var (
	_ resource.Resource                = &DomainResource{}
	_ resource.ResourceWithConfigure   = &DomainResource{}
	_ resource.ResourceWithImportState = &DomainResource{}
)

func init() {
	RegisteredResources = append(RegisteredResources, NewDomainResource)
}

func NewDomainResource() resource.Resource {
	return &DomainResource{}
}

// DomainResource maps a custom domain to a service instance or a My App.
type DomainResource struct {
	osaasContext *osaasclient.Context
}

type DomainResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Domain       types.String `tfsdk:"domain"`
	ServiceID    types.String `tfsdk:"service_id"`
	InstanceName types.String `tfsdk:"instance_name"`
	OriginPath   types.String `tfsdk:"origin_path"`
}

func (r *DomainResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain"
}

func (r *DomainResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Maps a custom domain to a service instance or a My App. OSC issues the TLS certificate once " +
			"the domain resolves to OSC, so create the DNS record (a CNAME to the instance's or app's hostname) first.\n\n" +
			"For a My App use `service_id = osc_my_app.<name>.domain_service_id` and `instance_name = osc_my_app.<name>.id`. " +
			"For a My Page use `custom_domain` on `osc_my_page` instead.\n\n" +
			"All attributes replace the mapping when changed. An instance can have several domains.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Identifier in the form `service_id/instance_name/domain`. Use it with `terraform import`.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"domain": schema.StringAttribute{
				Required:      true,
				Description:   "The fully qualified domain, e.g. `www.example.com`. Internationalized domains are converted to punycode by OSC.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_id": schema.StringAttribute{
				Required:      true,
				Description:   "Service id of the instance, e.g. `minio-minio`, or `eyevinn-web-runner` for a My App.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"instance_name": schema.StringAttribute{
				Required:      true,
				Description:   "Name of the instance, or the id of the My App.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"origin_path": schema.StringAttribute{
				Optional: true,
				Description: "Path prefix on the instance that the domain's root maps to, e.g. `/my-bucket` to serve one " +
					"bucket of a storage instance at the domain.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *DomainResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.osaasContext = configureContext(req.ProviderData, &resp.Diagnostics)
}

func domainID(serviceID, instanceName, domain string) string {
	return serviceID + "/" + instanceName + "/" + domain
}

func (r *DomainResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	serviceID, instance, domain := plan.ServiceID.ValueString(), plan.InstanceName.ValueString(), plan.Domain.ValueString()
	if err := createDomain(r.osaasContext, serviceID, instance, domain, plan.OriginPath.ValueString()); err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("domain"), "Failed to create domain mapping",
			fmt.Sprintf("Could not map %q to %s/%s: %s. A 409 means the domain is already mapped, a 402 that the plan does not include custom domains.",
				domain, serviceID, instance, err.Error()))
		return
	}
	plan.ID = types.StringValue(domainID(serviceID, instance, domain))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mapping, err := findDomain(r.osaasContext, state.ServiceID.ValueString(), state.InstanceName.ValueString(), state.Domain.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read domains", err.Error())
		return
	}
	if mapping == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	if mapping.OriginPath != "" || !state.OriginPath.IsNull() {
		state.OriginPath = types.StringValue(mapping.OriginPath)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Every attribute requires replacement, so there is nothing to update in place.
	var plan DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *DomainResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state DomainResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := deleteDomain(r.osaasContext, state.ServiceID.ValueString(), state.InstanceName.ValueString(), state.Domain.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to delete domain mapping", err.Error())
	}
}

func (r *DomainResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, "/", 3)
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		resp.Diagnostics.AddError("Invalid import id",
			fmt.Sprintf("Expected \"service_id/instance_name/domain\", e.g. \"eyevinn-web-runner/myapp/www.example.com\", got %q.", req.ID))
		return
	}
	mapping, err := findDomain(r.osaasContext, parts[0], parts[1], parts[2])
	if err != nil {
		resp.Diagnostics.AddError("Failed to read domains", err.Error())
		return
	}
	if mapping == nil {
		resp.Diagnostics.AddError("Domain mapping not found", fmt.Sprintf("No mapping of %q to %s/%s in this workspace.", parts[2], parts[0], parts[1]))
		return
	}
	model := DomainResourceModel{
		ID:           types.StringValue(req.ID),
		Domain:       types.StringValue(mapping.Domain),
		ServiceID:    types.StringValue(mapping.ServiceID),
		InstanceName: types.StringValue(mapping.InstanceName),
		OriginPath:   types.StringNull(),
	}
	if mapping.OriginPath != "" {
		model.OriginPath = types.StringValue(mapping.OriginPath)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
