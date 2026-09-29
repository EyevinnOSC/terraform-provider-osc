package provider

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// suspendedInstance is a service instance that has been suspended, by the tenant or by
// the platform. OSC removes it from the service's instance list and keeps its
// parameters, so that resuming it creates the same instance again.
type suspendedInstance struct {
	ServiceID    string                 `json:"serviceId"`
	InstanceName string                 `json:"instanceName"`
	SuspendedAt  string                 `json:"suspendedAt"`
	Reason       string                 `json:"reason"`
	Parameters   map[string]interface{} `json:"parameters"`
}

// findSuspended returns the suspended instance, or nil when the instance is not suspended.
func findSuspended(ctx *osaasclient.Context, serviceID, name string) (*suspendedInstance, error) {
	var list []suspendedInstance
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mysuspended"), nil, &list); err != nil {
		return nil, err
	}
	for i := range list {
		if list[i].ServiceID == serviceID && list[i].InstanceName == name {
			return &list[i], nil
		}
	}
	return nil, nil
}

// suspendedMu serialises the calls that change the tenant's list of suspended instances.
// OSC loses records when they overlap, and Terraform applies several resources at once.
var suspendedMu sync.Mutex

// resumeSuspended asks OSC to create the suspended instance again.
func resumeSuspended(ctx *osaasclient.Context, serviceID, name string) error {
	suspendedMu.Lock()
	defer suspendedMu.Unlock()
	return deployDo(ctx, http.MethodPost, deployURL(ctx, "/mysuspended/%s/%s/resume", serviceID, name), nil, nil)
}

// discardSuspended removes a suspended instance without resuming it. An instance that is
// not suspended is not an error.
func discardSuspended(ctx *osaasclient.Context, serviceID, name string) error {
	suspendedMu.Lock()
	defer suspendedMu.Unlock()
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mysuspended/%s/%s", serviceID, name), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// resumePollInterval is how often waitForResumed polls; tests shorten it.
var resumePollInterval = 5 * time.Second

// waitForResumed polls the service API until the resumed instance is listed again. For
// up to a minute after the resume call returns, the API answers 401 for it, so failed
// reads are retried until the timeout.
func waitForResumed(service *catalogService, name, token string, timeout time.Duration) (map[string]interface{}, error) {
	deadline := time.Now().Add(timeout)
	for {
		instance, err := getInstance(service, name, token)
		if err == nil && instance != nil {
			return instance, nil
		}
		if time.Now().After(deadline) {
			return nil, err
		}
		time.Sleep(resumePollInterval)
	}
}

// resumeSettle is how long after a resume OSC may still apply the parameters it kept for
// the instance, overwriting an update made in the meantime.
var resumeSettle = 90 * time.Second

// bodyApplied reports whether the live instance has the values of an update body.
// Values the service does not echo back are not compared.
func bodyApplied(body, instance map[string]interface{}) bool {
	want, live := flattenInstance(body), flattenInstance(instance)
	for k, v := range want {
		if got, ok := live[k]; ok && got != v {
			return false
		}
	}
	return true
}

// patchAfterResume updates a just resumed instance, and applies the update again for as
// long as OSC may still overwrite it with the parameters it kept while suspended.
func patchAfterResume(service *catalogService, name, token string, body map[string]interface{}, resumedAt time.Time) (map[string]interface{}, error) {
	out, err := patchInstance(service, name, token, body)
	if err != nil {
		return nil, err
	}
	for time.Since(resumedAt) < resumeSettle {
		time.Sleep(resumePollInterval)
		instance, err := getInstance(service, name, token)
		if err != nil || instance == nil || bodyApplied(body, instance) {
			continue
		}
		if out, err = patchInstance(service, name, token, body); err != nil {
			return nil, err
		}
	}
	instance, err := getInstance(service, name, token)
	if err == nil && instance != nil && !bodyApplied(body, instance) {
		return nil, fmt.Errorf("the resumed instance does not have the updated parameters")
	}
	return out, nil
}

// statefulServices keep data on a volume that suspending an instance deletes.
var statefulServices = map[string]bool{
	"minio-minio":            true,
	"valkey-io-valkey":       true,
	"birme-osc-postgresql":   true,
	"apache-couchdb":         true,
	"eyevinn-app-config-svc": true,
}

// isStateful reports whether suspending an instance of the service probably loses data.
func isStateful(service *catalogService) bool {
	if statefulServices[service.ServiceId] {
		return true
	}
	switch strings.ToLower(service.Metadata.Category) {
	case "database", "storage":
		return true
	}
	return false
}

// modelFromSuspended builds the state of an imported suspended instance from the
// parameters OSC keeps for it. Its url and external address are empty until it is
// resumed, since OSC only knows them for a running instance.
func modelFromSuspended(service *catalogService, name string, suspended *suspendedInstance) InstanceResourceModel {
	params, sensitive := parametersFromInstance(service, suspended.Parameters)
	model := InstanceResourceModel{
		ID:                  types.StringValue(service.ServiceId + "/" + name),
		ServiceID:           types.StringValue(service.ServiceId),
		Name:                types.StringValue(name),
		Parameters:          types.MapNull(types.StringType),
		SensitiveParameters: types.MapNull(types.StringType),
		AsSecrets:           types.BoolValue(true),
		SecretNames:         stringsToMap(nil),
		WaitForReady:        types.BoolValue(true),
		AllowSuspend:        types.BoolValue(false),
		Suspended:           types.BoolValue(true),
		UseLatest:           types.BoolValue(false),
		URL:                 types.StringValue(""),
		ExternalIP:          types.StringValue(""),
		ExternalPort:        types.Int64Value(0),
		Instance:            stringsToMap(nil),
	}
	if len(params) > 0 {
		model.Parameters = stringsToMap(params)
	}
	if len(sensitive) > 0 {
		model.SensitiveParameters = stringsToMap(sensitive)
	}
	return model
}
