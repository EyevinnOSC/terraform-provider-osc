package provider

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// configureContext returns the OSC client the provider hands to its resources, or nil
// before the provider is configured.
func configureContext(providerData interface{}, diags *diag.Diagnostics) *osaasclient.Context {
	if providerData == nil {
		return nil
	}
	osaasContext, ok := providerData.(*osaasclient.Context)
	if !ok {
		diags.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *osaasclient.Context, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return nil
	}
	return osaasContext
}

// isSet reports whether a string attribute has a known, non-empty value.
func isSet(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != ""
}
