package provider

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	valkeyServiceID    = "valkey-io-valkey"
	valkeyPasswordKey  = "Password"
	generatedPassBytes = 32
)

// generatePassword returns a random URL safe password (base64url without padding, so it
// needs no escaping inside a redis:// URL).
func generatePassword() (string, error) {
	b := make([]byte, generatedPassBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// isCreateOrReplace reports whether the plan creates the instance or replaces it, i.e.
// there is no prior state or the prior state is for another service or instance name.
func isCreateOrReplace(plan InstanceResourceModel, state *InstanceResourceModel) bool {
	if state == nil {
		return true
	}
	return !plan.ServiceID.Equal(state.ServiceID) || !plan.Name.Equal(state.Name)
}

// planValkeyPassword returns the sensitive_parameters to plan for a Valkey instance that
// is created or replaced without a password. The password is planned as an unknown value
// and generated in Create, because Terraform requires a plan to be deterministic: a value
// generated at plan time would differ between the plan and the re-plan done at apply.
// Once created, the generated value lives in state, so later plans see a password and
// inject nothing; an in-place update of an existing instance is never touched.
// The boolean is false when nothing is injected.
func planValkeyPassword(plan InstanceResourceModel, state *InstanceResourceModel) (types.Map, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan.ServiceID.IsUnknown() || plan.ServiceID.IsNull() || plan.ServiceID.ValueString() != valkeyServiceID ||
		!isCreateOrReplace(plan, state) {
		return plan.SensitiveParameters, false, diags
	}
	// A plain Password, or a map that may hold one, is the user's choice.
	if plan.Parameters.IsUnknown() {
		return plan.SensitiveParameters, false, diags
	}
	if _, ok := plan.Parameters.Elements()[valkeyPasswordKey]; ok {
		return plan.SensitiveParameters, false, diags
	}
	if plan.SensitiveParameters.IsUnknown() {
		diags.AddAttributeError(path.Root("sensitive_parameters"), "Cannot generate a password for the Valkey instance",
			"The value of sensitive_parameters is not known until apply, so the provider cannot tell whether it sets "+
				"sensitive_parameters.Password and cannot generate one. A Valkey instance created without a password is open "+
				"to anyone who can reach it. Set sensitive_parameters.Password explicitly, with the keys of the map known at plan time.")
		return plan.SensitiveParameters, false, diags
	}
	elems := map[string]attr.Value{}
	if !plan.SensitiveParameters.IsNull() {
		for k, v := range plan.SensitiveParameters.Elements() {
			elems[k] = v
		}
	}
	if _, ok := elems[valkeyPasswordKey]; ok {
		return plan.SensitiveParameters, false, diags
	}
	elems[valkeyPasswordKey] = types.StringUnknown()
	m, d := types.MapValue(types.StringType, elems)
	diags.Append(d...)
	return m, !d.HasError(), diags
}

// resolveGeneratedPassword replaces a planned unknown sensitive Password of a Valkey
// instance with a generated value. It never logs or reports the value.
func resolveGeneratedPassword(plan *InstanceResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	if plan.ServiceID.ValueString() != valkeyServiceID || plan.SensitiveParameters.IsNull() || plan.SensitiveParameters.IsUnknown() {
		return diags
	}
	v, ok := plan.SensitiveParameters.Elements()[valkeyPasswordKey]
	if !ok {
		return diags
	}
	if s, isStr := v.(types.String); !isStr || !s.IsUnknown() {
		return diags
	}
	pw, err := generatePassword()
	if err != nil {
		diags.AddAttributeError(path.Root("sensitive_parameters"), "Failed to generate a Valkey password",
			fmt.Sprintf("Could not read random bytes for sensitive_parameters.%s: %s", valkeyPasswordKey, err.Error()))
		return diags
	}
	elems := map[string]attr.Value{}
	for k, e := range plan.SensitiveParameters.Elements() {
		elems[k] = e
	}
	elems[valkeyPasswordKey] = types.StringValue(pw)
	m, d := types.MapValue(types.StringType, elems)
	diags.Append(d...)
	if !d.HasError() {
		plan.SensitiveParameters = m
		plan.SecretNames = stringsToMap(desiredSecretNames(plan.ServiceID.ValueString(), plan.Name.ValueString(),
			mapToStrings(m), plan.AsSecrets.ValueBool()))
	}
	return diags
}
