package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/list"
	"github.com/hashicorp/terraform-plugin-framework/list/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ list.ListResource              = &InstanceResource{}
	_ list.ListResourceWithConfigure = &InstanceResource{}
)

func init() {
	RegisteredListResources = append(RegisteredListResources, NewInstanceListResource)
}

// NewInstanceListResource exposes osc_instance to `terraform query`. The managed resource
// and the list resource share one implementation, as the framework intends.
func NewInstanceListResource() list.ListResource {
	return &InstanceResource{}
}

type InstanceListModel struct {
	ServiceID types.String `tfsdk:"service_id"`
}

func (r *InstanceResource) ListResourceConfigSchema(_ context.Context, _ list.ListResourceSchemaRequest, resp *list.ListResourceSchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists the service instances that exist in the workspace, for `terraform query`. " +
			"Every instance of every service the workspace is subscribed to is returned unless `service_id` narrows " +
			"it to one service. Run `terraform query -generate-config-out=generated.tf` to write `import` blocks " +
			"and complete `osc_instance` resource blocks, parameters included, for everything that is not yet under " +
			"Terraform management. Terraform leaves sensitive values out of generated configuration, so fill in " +
			"`sensitive_parameters` before applying; see the provider overview for the full procedure.\n\n" +
			"Requires Terraform 1.14 or later.",
		Attributes: map[string]schema.Attribute{
			"service_id": schema.StringAttribute{
				Optional:    true,
				Description: "Only list instances of this service, e.g. `valkey-io-valkey`. Lists all subscribed services when unset.",
			},
		},
	}
}

func (r *InstanceResource) List(ctx context.Context, req list.ListRequest, stream *list.ListResultsStream) {
	var config InstanceListModel
	diags := req.Config.Get(ctx, &config)
	if diags.HasError() {
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}
	if r.osaasContext == nil {
		diags.AddError("Provider not configured", "The OSC provider must be configured before listing instances.")
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}

	services, err := listSubscriptions(r.osaasContext)
	if err != nil {
		diags.AddError("Failed to read OSC catalog", err.Error())
		stream.Results = list.ListResultsStreamDiagnostics(diags)
		return
	}
	if !config.ServiceID.IsNull() && config.ServiceID.ValueString() != "" {
		want := config.ServiceID.ValueString()
		filtered := services[:0]
		for _, s := range services {
			if s.ServiceId == want {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			diags.AddAttributeWarning(path.Root("service_id"), "Service not subscribed",
				fmt.Sprintf("The workspace is not subscribed to service %q, so it has no instances of it.", want))
			stream.Results = list.ListResultsStreamDiagnostics(diags)
			return
		}
		services = filtered
	}

	stream.Results = func(yield func(list.ListResult) bool) {
		var count int64
		for i := range services {
			service := &services[i]
			if service.ServiceType != "" && service.ServiceType != "instance" {
				continue
			}
			token, err := r.osaasContext.GetServiceAccessToken(service.ServiceId)
			if err != nil {
				if !yield(warningResult(ctx, req, "Failed to get service access token",
					fmt.Sprintf("Instances of service %q were skipped: %s", service.ServiceId, err.Error()))) {
					return
				}
				continue
			}
			instances, err := listInstances(service, token)
			if err != nil {
				if !yield(warningResult(ctx, req, "Failed to list instances",
					fmt.Sprintf("Instances of service %q were skipped: %s", service.ServiceId, err.Error()))) {
					return
				}
				continue
			}
			for _, instance := range instances {
				if req.Limit > 0 && count >= req.Limit {
					return
				}
				name, _ := instance["name"].(string)
				if name == "" {
					continue
				}
				result := req.NewListResult(ctx)
				result.DisplayName = service.ServiceId + "/" + name
				result.Diagnostics.Append(result.Identity.Set(ctx, InstanceIdentityModel{
					ServiceID: types.StringValue(service.ServiceId),
					Name:      types.StringValue(name),
				})...)
				if req.IncludeResource {
					model, d := r.modelFromInstance(service, token, instance)
					result.Diagnostics.Append(d...)
					result.Diagnostics.Append(result.Resource.Set(ctx, &model)...)
				}
				count++
				if !yield(result) {
					return
				}
			}
		}
	}
}

// warningResult wraps a per-service problem as a warning so that one unreachable service
// does not hide the instances of every other service.
func warningResult(ctx context.Context, req list.ListRequest, summary, detail string) list.ListResult {
	result := req.NewListResult(ctx)
	result.Diagnostics.AddWarning(summary, detail)
	return result
}
