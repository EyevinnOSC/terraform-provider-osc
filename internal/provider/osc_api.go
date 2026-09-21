package provider

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	osaasclient "github.com/EyevinnOSC/client-go"

	"terraform-provider-osc/internal/catalog"
)

// serviceOption is one entry of a service's instance option schema, shared with the
// catalog mirror.
type serviceOption = catalog.Option

// catalogService is a service as returned by the OSC catalog subscriptions endpoint.
type catalogService struct {
	ServiceId              string                      `json:"serviceId"`
	Metadata               osaasclient.ServiceMetadata `json:"serviceMetadata"`
	ApiUrl                 string                      `json:"apiUrl"`
	ServiceInstanceOptions []serviceOption             `json:"serviceInstanceOptions"`
	ServiceType            string                      `json:"serviceType"`
	Status                 string                      `json:"status"`
}

// apiError is returned for non-2xx responses from the OSC APIs.
type apiError struct {
	StatusCode int
	Body       string
}

func (e *apiError) Error() string {
	msg := strings.TrimSpace(e.Body)
	var parsed map[string]interface{}
	if json.Unmarshal([]byte(e.Body), &parsed) == nil {
		for _, key := range []string{"reason", "message", "error"} {
			if v, ok := parsed[key].(string); ok && v != "" {
				msg = v
				break
			}
		}
	}
	if msg == "" {
		msg = http.StatusText(e.StatusCode)
	}
	return fmt.Sprintf("HTTP %d: %s", e.StatusCode, msg)
}

func isNotFound(err error) bool {
	var ae *apiError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusNotFound
}

var httpClient = &http.Client{Timeout: 60 * time.Second}

// doJSON performs a JSON request against an OSC API. authHeader/authValue carry the
// credential (x-pat-jwt for platform APIs, x-jwt for service instance APIs).
func doJSON(method, rawURL, authHeader, authValue string, body interface{}, out interface{}) error {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, rawURL, reader)
	if err != nil {
		return err
	}
	req.Header.Set(authHeader, authValue)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &apiError{StatusCode: resp.StatusCode, Body: string(respBytes)}
	}
	if out != nil && len(bytes.TrimSpace(respBytes)) > 0 {
		if err := json.Unmarshal(respBytes, out); err != nil {
			return fmt.Errorf("failed to decode response from %s: %w", rawURL, err)
		}
	}
	return nil
}

func patAuth(ctx *osaasclient.Context) (string, string) {
	return "x-pat-jwt", "Bearer " + ctx.GetPersonalAccessToken()
}

func serviceAuth(token string) (string, string) {
	return "x-jwt", "Bearer " + token
}

// listSubscriptions returns the services the tenant is subscribed to. This is the only
// catalog listing available with a personal access token, and it is the source of truth
// for a service's API URL and instance option schema.
func listSubscriptions(ctx *osaasclient.Context) ([]catalogService, error) {
	u := fmt.Sprintf("https://catalog.svc.%s.osaas.io/mysubscriptions", ctx.GetEnvironment())
	var services []catalogService
	h, v := patAuth(ctx)
	if err := doJSON(http.MethodGet, u, h, v, nil, &services); err != nil {
		return nil, err
	}
	return services, nil
}

// findSubscribedService returns the subscribed service with the given id, or nil if the
// tenant is not subscribed to it.
func findSubscribedService(ctx *osaasclient.Context, serviceId string) (*catalogService, error) {
	services, err := listSubscriptions(ctx)
	if err != nil {
		return nil, err
	}
	for i := range services {
		if services[i].ServiceId == serviceId {
			return &services[i], nil
		}
	}
	return nil, nil
}

// ensureSubscribed returns the service, subscribing the tenant to it first if needed.
func ensureSubscribed(ctx *osaasclient.Context, serviceId string) (*catalogService, error) {
	service, err := findSubscribedService(ctx, serviceId)
	if err != nil {
		return nil, err
	}
	if service != nil {
		return service, nil
	}
	if err := ctx.ActivateService(serviceId); err != nil {
		return nil, fmt.Errorf("could not subscribe to service %q: %w. Check that the service id is correct (ids look like {contributor}-{name} and cannot be guessed; look them up in the OSC catalog)", serviceId, err)
	}
	service, err = findSubscribedService(ctx, serviceId)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, fmt.Errorf("service %q not found in subscriptions after subscribing; check that the service id is correct", serviceId)
	}
	return service, nil
}

func instanceHost(service *catalogService) (string, error) {
	u, err := url.Parse(service.ApiUrl)
	if err != nil {
		return "", err
	}
	return u.Host, nil
}

// createInstance creates a new instance of the service. With useLatest the instance is
// provisioned from the latest built image instead of the pinned stable release.
func createInstance(service *catalogService, token string, body map[string]interface{}, useLatest bool) (map[string]interface{}, error) {
	u := service.ApiUrl
	if useLatest {
		u += "?beta=true"
	}
	var out map[string]interface{}
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodPost, u, h, v, body, &out); err != nil {
		return nil, err
	}
	if reason, ok := out["reason"].(string); ok && reason != "" {
		return nil, errors.New(reason)
	}
	return out, nil
}

// getInstance returns the instance document, or nil if the instance does not exist.
func getInstance(service *catalogService, name, token string) (map[string]interface{}, error) {
	u := fmt.Sprintf("%s/%s", service.ApiUrl, name)
	var out map[string]interface{}
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodGet, u, h, v, nil, &out); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return out, nil
}

// removeInstance deletes the instance. A missing instance is not an error.
func removeInstance(service *catalogService, name, token string) error {
	u := fmt.Sprintf("%s/%s", service.ApiUrl, name)
	h, v := serviceAuth(token)
	err := doJSON(http.MethodDelete, u, h, v, nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// getPorts returns the TCP/UDP ports exposed by the instance, if any.
func getPorts(service *catalogService, name, token string) ([]osaasclient.Port, error) {
	host, err := instanceHost(service)
	if err != nil {
		return nil, err
	}
	u := fmt.Sprintf("https://%s/ports/%s", host, name)
	var ports []osaasclient.Port
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodGet, u, h, v, nil, &ports); err != nil {
		return nil, err
	}
	return ports, nil
}

// patchInstance updates the configuration of a running instance in place.
func patchInstance(service *catalogService, name, token string, body map[string]interface{}) (map[string]interface{}, error) {
	u := fmt.Sprintf("%s/%s", service.ApiUrl, name)
	var out map[string]interface{}
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodPatch, u, h, v, body, &out); err != nil {
		return nil, err
	}
	if reason, ok := out["reason"].(string); ok && reason != "" {
		return nil, errors.New(reason)
	}
	return out, nil
}

// getInstanceHealth returns the health status string reported by the service
// (e.g. "running"). Not all services implement the health endpoint.
func getInstanceHealth(service *catalogService, name, token string) (string, error) {
	host, err := instanceHost(service)
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf("https://%s/health/%s", host, name)
	var out struct {
		Status string `json:"status"`
	}
	h, v := serviceAuth(token)
	if err := doJSON(http.MethodGet, u, h, v, nil, &out); err != nil {
		return "", err
	}
	return out.Status, nil
}

// waitForInstanceReady polls the health endpoint until the instance reports running,
// the timeout expires, or the service turns out not to support health checks.
// It returns whether the instance was confirmed running.
func waitForInstanceReady(service *catalogService, name, token string, timeout time.Duration) (bool, error) {
	deadline := time.Now().Add(timeout)
	for {
		status, err := getInstanceHealth(service, name, token)
		if err != nil {
			if isNotFound(err) {
				// Either the health endpoint is not implemented or the instance is not
				// registered yet. Keep polling for a little while, then give up quietly.
				if time.Now().After(deadline) {
					return false, nil
				}
			} else {
				return false, err
			}
		} else if strings.EqualFold(status, "running") {
			return true, nil
		}
		if time.Now().After(deadline) {
			return false, nil
		}
		time.Sleep(3 * time.Second)
	}
}

// flattenInstance converts the raw instance document into a map of strings suitable
// for a Terraform map(string) attribute. Nested values are JSON encoded.
func flattenInstance(instance map[string]interface{}) map[string]string {
	out := make(map[string]string, len(instance))
	for k, v := range instance {
		switch t := v.(type) {
		case nil:
			out[k] = ""
		case string:
			out[k] = t
		case bool:
			if t {
				out[k] = "true"
			} else {
				out[k] = "false"
			}
		case float64:
			out[k] = strconv.FormatFloat(t, 'f', -1, 64)
		default:
			b, err := json.Marshal(t)
			if err != nil {
				out[k] = fmt.Sprintf("%v", t)
			} else {
				out[k] = string(b)
			}
		}
	}
	return out
}

func asAPIError(err error, target **apiError) bool {
	return errors.As(err, target)
}
