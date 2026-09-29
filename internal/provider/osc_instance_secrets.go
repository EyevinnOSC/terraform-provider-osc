package provider

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// maxSecretBatch is the number of secrets the batch endpoint accepts per request.
const maxSecretBatch = 10

// instanceSecretName is the name of the service secret that holds a sensitive parameter
// of an instance. OSC stores secrets as Kubernetes secrets named after the tenant and the
// secret name, so it must be lowercase alphanumeric. The readable part is the instance
// and parameter name; the hash keeps names apart that read the same once lowercased and
// stripped ("ivy" + "PreviewKey" and "ivypreview" + "Key", "Prod" and "prod"), and across
// services, since the Kubernetes name does not include the service.
func instanceSecretName(serviceID, instance, key string) string {
	sum := sha256.Sum256([]byte(serviceID + "/" + instance + "/" + key))
	readable := alnumLower(instance) + alnumLower(key)
	if len(readable) > 40 {
		readable = readable[:40]
	}
	return readable + hex.EncodeToString(sum[:])[:8]
}

func alnumLower(s string) string {
	var b strings.Builder
	for _, c := range strings.ToLower(s) {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') {
			b.WriteRune(c)
		}
	}
	return b.String()
}

func secretRef(name string) string {
	return "{{secrets." + name + "}}"
}

// desiredSecretNames maps each sensitive parameter to the secret that should hold it.
// Values that already are secret references are passed to OSC as they are.
func desiredSecretNames(serviceID, instance string, sensitive map[string]string, enabled bool) map[string]string {
	names := map[string]string{}
	if !enabled {
		return names
	}
	for k, v := range sensitive {
		if isSecretRef(v) {
			continue
		}
		names[k] = instanceSecretName(serviceID, instance, k)
	}
	return names
}

// withSecretRefs returns the sensitive parameters as sent to OSC: values held in a
// secret are replaced by a reference to it.
func withSecretRefs(sensitive, secretNames map[string]string) map[string]string {
	out := make(map[string]string, len(sensitive))
	for k, v := range sensitive {
		if name, ok := secretNames[k]; ok {
			out[k] = secretRef(name)
		} else {
			out[k] = v
		}
	}
	return out
}

// secretDrift returns the parameters whose live value is not the reference to their
// secret, e.g. because the instance was changed outside Terraform or was created by a
// provider version that sent the values in plain text. Parameters the service does not
// echo back are not drift.
func secretDrift(secretNames map[string]string, instance map[string]interface{}) []string {
	flat := flattenInstance(instance)
	var drifted []string
	for k, name := range secretNames {
		if v, ok := flat[k]; ok && strings.TrimSpace(v) != secretRef(name) {
			drifted = append(drifted, k)
		}
	}
	return drifted
}

// putSecrets creates or updates service secrets, name to value.
func putSecrets(ctx *osaasclient.Context, serviceID string, secrets map[string]string) error {
	type secret struct {
		SecretName string `json:"secretName"`
		SecretData string `json:"secretData"`
	}
	var batch []secret
	flush := func() error {
		if len(batch) == 0 {
			return nil
		}
		err := deployDo(ctx, http.MethodPut, deployURL(ctx, "/mysecrets/%s/batch", serviceID), map[string]interface{}{"secrets": batch}, nil)
		batch = nil
		return err
	}
	for name, value := range secrets {
		batch = append(batch, secret{name, value})
		if len(batch) == maxSecretBatch {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	return flush()
}

// deleteSecret removes a service secret. A missing secret is not an error.
func deleteSecret(ctx *osaasclient.Context, serviceID, name string) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mysecrets/%s/%s", serviceID, name), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// restartInstance restarts the instance through the restart link in its document, so
// that it reads secrets whose values changed. It reports false when the service does
// not offer a restart.
func restartInstance(service *catalogService, token string, instance map[string]interface{}) (bool, error) {
	links, _ := instance["_links"].(map[string]interface{})
	restart, _ := links["restart"].(map[string]interface{})
	href, _ := restart["href"].(string)
	if href == "" {
		return false, nil
	}
	host, err := instanceHost(service)
	if err != nil {
		return false, err
	}
	h, v := serviceAuth(token)
	return true, doJSON(http.MethodPost, "https://"+host+href, h, v, nil, nil)
}

// plannedSecretNames is the secret_names attribute for a plan: unknown until the instance
// name and all sensitive values are known, since a value may turn out to be a reference.
func plannedSecretNames(plan InstanceResourceModel) types.Map {
	if plan.ServiceID.IsUnknown() || plan.Name.IsUnknown() || plan.AsSecrets.IsUnknown() || plan.SensitiveParameters.IsUnknown() {
		return types.MapUnknown(types.StringType)
	}
	if len(mapToParamValues(plan.SensitiveParameters).unknown) > 0 {
		return types.MapUnknown(types.StringType)
	}
	return stringsToMap(desiredSecretNames(plan.ServiceID.ValueString(), plan.Name.ValueString(),
		mapToStrings(plan.SensitiveParameters), plan.AsSecrets.ValueBool()))
}

// secretValues maps secret name to the value it holds.
func secretValues(sensitive, secretNames map[string]string) map[string]string {
	out := make(map[string]string, len(secretNames))
	for k, name := range secretNames {
		out[name] = sensitive[k]
	}
	return out
}

// movesToSecrets reports whether an update points a parameter at a secret it did not
// refer to before.
func movesToSecrets(oldNames, newNames map[string]string) bool {
	for k, name := range newNames {
		if oldNames[k] != name {
			return true
		}
	}
	return false
}
