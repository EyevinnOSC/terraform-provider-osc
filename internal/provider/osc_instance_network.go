package provider

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Use a pointer: an absent boolean must never be mistaken for private access.
type internalEndpoint struct {
	ServiceDNS   string `json:"serviceDns"`
	PublicAccess *bool  `json:"publicAccess"`
}

func getInternalEndpoint(service *catalogService, name, token string) (*internalEndpoint, error) {
	host, err := instanceHost(service)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("https://%s/internal-endpoint/%s", host, url.PathEscape(name))
	var out internalEndpoint
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodGet, u, h, v, nil, &out); err != nil {
		return nil, err
	}
	if out.PublicAccess == nil || out.ServiceDNS == "" {
		return nil, fmt.Errorf("internal endpoint response is missing publicAccess or serviceDns")
	}
	return &out, nil
}

func setCreateExposure(body map[string]interface{}, access types.Bool) {
	if !access.IsNull() && !access.IsUnknown() {
		body["publicAccess"] = access.ValueBool()
	}
}

// Until the exposure-update API is verified, do not guess at a PATCH or replace
// a stateful service. This is also checked in Update, before any side effect.
func validateExposureChange(plan, state types.Bool) diag.Diagnostics {
	var diags diag.Diagnostics
	if !plan.IsNull() && !plan.IsUnknown() && !plan.Equal(state) {
		diags.AddAttributeError(path.Root("public_access"), "Changing network exposure is not supported",
			"This provider supports public_access at creation and reads it from OSC. Change existing instance exposure through OSC, then refresh Terraform state. No replacement is planned for this setting.")
	}
	return diags
}

func refreshNetwork(service *catalogService, token string, model *InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	endpoint, err := getInternalEndpoint(service, model.Name.ValueString(), token)
	if err != nil {
		// Older services may not implement this endpoint. Preserve compatibility
		// only when network exposure was never configured/observed.
		if isNotFound(err) && (model.PublicAccess.IsNull() || model.PublicAccess.IsUnknown()) {
			model.PublicAccess = types.BoolNull()
			model.InternalHost = types.StringValue("")
			return diags
		}
		diags.AddError("Could not verify network exposure", err.Error())
		return diags
	}
	model.PublicAccess = types.BoolValue(*endpoint.PublicAccess)
	model.InternalHost = types.StringValue(endpoint.ServiceDNS)
	return diags
}
