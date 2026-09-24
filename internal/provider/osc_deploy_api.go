package provider

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// The deploy manager is the platform API behind My Apps, My Pages, custom domains and
// parameter stores. None of these are catalog services, so they are managed through it
// with the personal access token rather than through a service instance API.

// webRunnerServiceID is the service a My App runs on. Custom domains for a My App are
// mapped to it, with the app id as the instance name.
const webRunnerServiceID = "eyevinn-web-runner"

// appConfigServiceID is the service a parameter store runs on.
const appConfigServiceID = "eyevinn-app-config-svc"

func deployURL(ctx *osaasclient.Context, format string, args ...interface{}) string {
	escaped := make([]interface{}, len(args))
	for i, a := range args {
		escaped[i] = url.PathEscape(fmt.Sprint(a))
	}
	return fmt.Sprintf("https://deploy.svc.%s.osaas.io", ctx.GetEnvironment()) + fmt.Sprintf(format, escaped...)
}

func deployDo(ctx *osaasclient.Context, method, rawURL string, body, out interface{}) error {
	h, v := patAuth(ctx)
	return doJSON(method, rawURL, h, v, body, out)
}

// ------------------------------------------------------------------- My Apps

type myApp struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	GitURL         string `json:"gitUrl"`
	GitHubURL      string `json:"gitHubUrl"`
	URL            string `json:"url"`
	AppDNS         string `json:"appDns"`
	HAEnabled      bool   `json:"haEnabled"`
	BuildStatus    string `json:"buildStatus"`
	ConfigService  string `json:"configService"`
	SourceRef      string `json:"sourceRef"`
	SubPath        string `json:"subPath"`
	StageProd      bool   `json:"stageProdEnabled"`
	Sovereignty    string `json:"sovereignty"`
	TenantID       string `json:"tenantId"`
	AnalyticsBound string `json:"analyticsService"`
}

// sourceURL is the repository URL without the "#ref" fragment the API appends.
func (a *myApp) sourceURL() string {
	u := a.GitURL
	if u == "" {
		u = a.GitHubURL
	}
	if i := strings.Index(u, "#"); i >= 0 {
		u = u[:i]
	}
	return u
}

// boundConfigService is the parameter store the app is bound to. The API reports an
// unbound store as "" or, for some older apps, the string "undefined".
func (a *myApp) boundConfigService() string {
	if a.ConfigService == "undefined" {
		return ""
	}
	return a.ConfigService
}

func createMyApp(ctx *osaasclient.Context, body map[string]interface{}) (*myApp, error) {
	var out myApp
	if err := deployDo(ctx, http.MethodPost, deployURL(ctx, "/myapps"), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getMyApp returns the app, or nil if it does not exist.
func getMyApp(ctx *osaasclient.Context, id string) (*myApp, error) {
	var out myApp
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/myapps/%s", id), nil, &out); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if out.ID == "" {
		return nil, nil
	}
	return &out, nil
}

func deleteMyApp(ctx *osaasclient.Context, id string) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/myapps/%s", id), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// restartMyApp pulls the latest commit and restarts the app. With rebuild the instance is
// recreated from a fresh image; for HA apps that is a blue-green deployment.
func restartMyApp(ctx *osaasclient.Context, id string, rebuild bool) error {
	u := deployURL(ctx, "/myapps/%s", id)
	if rebuild {
		u += "?rebuild=true"
	}
	return deployDo(ctx, http.MethodPut, u, nil, nil)
}

// setMyAppConfig binds the app to a parameter store, or unbinds it when store is empty.
func setMyAppConfig(ctx *osaasclient.Context, id, store, apiKey string) error {
	body := map[string]interface{}{"configService": nil}
	if store != "" {
		body["configService"] = store
		if apiKey != "" {
			body["configApiKey"] = apiKey
		}
	}
	return deployDo(ctx, http.MethodPatch, deployURL(ctx, "/myapps/%s/config", id), body, nil)
}

// setMyAppSourceRef pins the app to a branch or tag, or tracks HEAD when ref is empty.
func setMyAppSourceRef(ctx *osaasclient.Context, id, ref string) error {
	body := map[string]interface{}{"sourceRef": nil}
	if ref != "" {
		body["sourceRef"] = ref
	}
	return deployDo(ctx, http.MethodPatch, deployURL(ctx, "/myapps/%s/source-ref", id), body, nil)
}

// setMyAppGitToken sets the token or credential reference used to clone the repository;
// both empty clears it.
func setMyAppGitToken(ctx *osaasclient.Context, id, token, credentialRef string) error {
	body := map[string]interface{}{"gitToken": nil}
	switch {
	case token != "":
		body["gitToken"] = token
	case credentialRef != "":
		body["gitCredentialRef"] = credentialRef
	}
	return deployDo(ctx, http.MethodPatch, deployURL(ctx, "/myapps/%s/git-token", id), body, nil)
}

func setMyAppHA(ctx *osaasclient.Context, id string, enabled bool) error {
	method := http.MethodPost
	if !enabled {
		method = http.MethodDelete
	}
	return deployDo(ctx, method, deployURL(ctx, "/myapps/%s/ha", id), nil, nil)
}

// waitForMyAppBuild polls the app until its build reports running or failed, or the
// timeout expires. It returns the last app document read.
func waitForMyAppBuild(ctx *osaasclient.Context, id string, timeout time.Duration) (*myApp, error) {
	// A restart or source change is accepted before the build status flips to building,
	// so give the platform a moment before trusting a "running" from the previous build.
	time.Sleep(5 * time.Second)
	return pollMyAppBuild(func() (*myApp, error) { return getMyApp(ctx, id) }, id, timeout, 5*time.Second, time.Minute)
}

// pollMyAppBuild is the loop behind waitForMyAppBuild. A rebuild recreates the app's
// instance, and while it does the platform answers for the app as if it did not exist,
// so an app only counts as gone once it has been missing for longer than missingFor.
func pollMyAppBuild(get func() (*myApp, error), id string, timeout, interval, missingFor time.Duration) (*myApp, error) {
	deadline := time.Now().Add(timeout)
	var last *myApp
	var missingSince time.Time
	for {
		app, err := get()
		if err != nil {
			return last, err
		}
		if app == nil {
			if missingSince.IsZero() {
				missingSince = time.Now()
			}
			if time.Since(missingSince) > missingFor {
				return last, fmt.Errorf("app %q disappeared while waiting for its build: missing for over %s", id, missingFor)
			}
		} else {
			missingSince = time.Time{}
			last = app
			switch app.BuildStatus {
			case "running":
				return app, nil
			case "failed":
				return app, fmt.Errorf("the build of app %q failed. Read its logs with the OSC CLI or dashboard, fix the repository and apply again", id)
			}
		}
		if time.Now().After(deadline) {
			status := "missing"
			if app != nil {
				status = app.BuildStatus
			}
			return last, fmt.Errorf("app %q did not report a running build within %s (last status %q)", id, timeout, status)
		}
		time.Sleep(interval)
	}
}

// -------------------------------------------------------------------- My Pages

type myPage struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	Status string `json:"status"`
	Domain string `json:"customDomain"`
	// Some responses name the custom domain "domain".
	DomainAlt string `json:"domain"`
	Auth      bool   `json:"authEnabled"`
}

func (p *myPage) customDomain() string {
	if p.Domain != "" {
		return p.Domain
	}
	return p.DomainAlt
}

// id is how the page is addressed in the API. Pages are named globally, so the name is
// accepted wherever an id is.
func (p *myPage) id() string {
	if p.ID != "" {
		return p.ID
	}
	return p.Name
}

func createMyPage(ctx *osaasclient.Context, name string) (*myPage, error) {
	var out myPage
	if err := deployDo(ctx, http.MethodPost, deployURL(ctx, "/mypages"), map[string]interface{}{"name": name}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getMyPage returns the page, or nil if it does not exist.
func getMyPage(ctx *osaasclient.Context, id string) (*myPage, error) {
	var out myPage
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mypages/%s", id), nil, &out); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if out.id() == "" {
		return nil, nil
	}
	return &out, nil
}

func deleteMyPage(ctx *osaasclient.Context, id string) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mypages/%s", id), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

func setMyPageDomain(ctx *osaasclient.Context, id, domain string) error {
	if domain == "" {
		err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mypages/%s/domain", id), nil, nil)
		if err != nil && isNotFound(err) {
			return nil
		}
		return err
	}
	return deployDo(ctx, http.MethodPost, deployURL(ctx, "/mypages/%s/domain", id), map[string]interface{}{"domain": domain}, nil)
}

func setMyPageAuth(ctx *osaasclient.Context, id, username, password string) error {
	if username == "" {
		return deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mypages/%s/auth", id), nil, nil)
	}
	return deployDo(ctx, http.MethodPut, deployURL(ctx, "/mypages/%s/auth", id),
		map[string]interface{}{"username": username, "password": password}, nil)
}

// ------------------------------------------------------------- custom domains

type domainMapping struct {
	Domain       string `json:"domain"`
	ServiceID    string `json:"serviceId"`
	InstanceName string `json:"instanceName"`
	OriginPath   string `json:"originPath"`
	IsManaged    bool   `json:"isManaged"`
}

func listDomains(ctx *osaasclient.Context) ([]domainMapping, error) {
	var out []domainMapping
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mydomains"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// findDomain returns the mapping of domain to the instance, or nil if there is none.
func findDomain(ctx *osaasclient.Context, serviceID, instanceName, domain string) (*domainMapping, error) {
	domains, err := listDomains(ctx)
	if err != nil {
		return nil, err
	}
	for i := range domains {
		d := &domains[i]
		if strings.EqualFold(d.Domain, domain) && d.ServiceID == serviceID && d.InstanceName == instanceName {
			return d, nil
		}
	}
	return nil, nil
}

// managedDomain returns the platform assigned hostname of an instance, e.g. the
// "<hash>.apps.osaas.io" name of a My App, or "" if it has none.
func managedDomain(ctx *osaasclient.Context, serviceID, instanceName string) (string, error) {
	domains, err := listDomains(ctx)
	if err != nil {
		return "", err
	}
	for _, d := range domains {
		if d.IsManaged && d.ServiceID == serviceID && d.InstanceName == instanceName {
			return d.Domain, nil
		}
	}
	return "", nil
}

func createDomain(ctx *osaasclient.Context, serviceID, instanceName, domain, originPath string) error {
	body := map[string]interface{}{"domain": domain, "serviceId": serviceID, "instanceName": instanceName}
	if originPath != "" {
		body["originPath"] = originPath
	}
	return deployDo(ctx, http.MethodPost, deployURL(ctx, "/mydomains"), body, nil)
}

func deleteDomain(ctx *osaasclient.Context, serviceID, instanceName, domain string) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mydomains/%s/%s/%s", serviceID, instanceName, domain), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// ----------------------------------------------------------- parameter stores

type parameterStoreInfo struct {
	Name         string `json:"name"`
	IsSecure     bool   `json:"isSecure"`
	ConfigAPIKey string `json:"configApiKey"`
}

func createParameterStore(ctx *osaasclient.Context, body map[string]interface{}) (*parameterStoreInfo, error) {
	var out parameterStoreInfo
	if err := deployDo(ctx, http.MethodPost, deployURL(ctx, "/parameter-stores"), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getParameterStore reports whether the store has secrets enabled and, if so, its API
// key. It answers for any name, so existence is checked on the instance instead.
func getParameterStore(ctx *osaasclient.Context, name string) (*parameterStoreInfo, error) {
	var out parameterStoreInfo
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/parameter-stores/%s", name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func deleteParameterStore(ctx *osaasclient.Context, name string) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/parameter-stores/%s", name), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// parameterStoreInstance returns the app-config-svc instance behind a parameter store
// and a token for its API, or a nil instance if the store does not exist.
func parameterStoreInstance(ctx *osaasclient.Context, name string) (map[string]interface{}, *catalogService, string, error) {
	service, err := ensureSubscribed(ctx, appConfigServiceID)
	if err != nil {
		return nil, nil, "", err
	}
	token, err := ctx.GetServiceAccessToken(appConfigServiceID)
	if err != nil {
		return nil, nil, "", err
	}
	instance, err := getInstance(service, name, token)
	if err != nil {
		return nil, nil, "", err
	}
	return instance, service, token, nil
}

// ------------------------------------------------------------------ parameters

// parameterClient talks to the app-config-svc instance of one parameter store.
type parameterClient struct {
	baseURL string
	token   string
	apiKey  string
}

type parameterObject struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Secret bool   `json:"secret"`
}

func newParameterClient(ctx *osaasclient.Context, store string) (*parameterClient, error) {
	instance, _, token, err := parameterStoreInstance(ctx, store)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("parameter store %q does not exist", store)
	}
	base, _ := instance["url"].(string)
	if base == "" {
		return nil, fmt.Errorf("parameter store %q has no URL yet; it may still be starting", store)
	}
	info, err := getParameterStore(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("could not read parameter store %q: %w", store, err)
	}
	return &parameterClient{baseURL: strings.TrimSuffix(base, "/") + "/api/v1/config", token: token, apiKey: info.ConfigAPIKey}, nil
}

// do retries what a store that has just started answers with: connection and TLS errors
// and gateway errors, for up to two minutes. Anything else is returned at once.
func (c *parameterClient) do(method, rawURL string, body, out interface{}) error {
	deadline := time.Now().Add(2 * time.Minute)
	for {
		err := doJSONWithHeaders(method, rawURL, c.headers(), body, out)
		if err == nil || !isTransient(err) || time.Now().After(deadline) {
			return err
		}
		time.Sleep(3 * time.Second)
	}
}

func isTransient(err error) bool {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae.StatusCode == http.StatusBadGateway || ae.StatusCode == http.StatusServiceUnavailable ||
			ae.StatusCode == http.StatusGatewayTimeout
	}
	return true
}

func (c *parameterClient) headers() map[string]string {
	h := map[string]string{"x-jwt": "Bearer " + c.token}
	if c.apiKey != "" {
		h["x-config-api-key"] = c.apiKey
	}
	return h
}

// get returns the parameter, or nil if it does not exist. Secret values are returned in
// plain text only when the store's API key is known; otherwise they come back masked.
func (c *parameterClient) get(key string) (*parameterObject, error) {
	var out parameterObject
	if err := c.do(http.MethodGet, c.baseURL+"/"+url.PathEscape(key), nil, &out); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// put creates the parameter, or updates it when it already exists.
func (c *parameterClient) put(key, value string, secret bool) error {
	existing, err := c.get(key)
	if err != nil {
		return err
	}
	if existing == nil {
		return c.do(http.MethodPost, c.baseURL, map[string]interface{}{"key": key, "value": value, "secret": secret}, nil)
	}
	return c.do(http.MethodPut, c.baseURL+"/"+url.PathEscape(key), map[string]interface{}{"value": value, "secret": secret}, nil)
}

func (c *parameterClient) delete(key string) error {
	err := c.do(http.MethodDelete, c.baseURL+"/"+url.PathEscape(key), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}

// waitForHTTPS polls rawURL until it answers over TLS with any status, or the timeout
// expires. A new instance's hostname can resolve before its certificate is issued, and
// until then every request fails certificate verification.
func waitForHTTPS(rawURL string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		resp, err := httpClient.Get(rawURL)
		if err == nil {
			_ = resp.Body.Close()
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%s did not answer over HTTPS within %s: %w", rawURL, timeout, err)
		}
		time.Sleep(5 * time.Second)
	}
}

// ------------------------------------------------------------------- mailbox

// mailbox is the workspace's one mailbox, {tenantId}@users.osaas.io. Mail sent through
// its SMTP server is relayed by the platform; the address is fixed.
type mailbox struct {
	TenantID  string         `json:"tenantId"`
	Email     string         `json:"email"`
	SMTP      mailServerInfo `json:"smtp"`
	IMAP      mailServerInfo `json:"imap"`
	CreatedAt string         `json:"createdAt"`
}

type mailServerInfo struct {
	Server     string  `json:"server"`
	Port       float64 `json:"port"`
	Encryption string  `json:"encryption"`
}

func createMailbox(ctx *osaasclient.Context, password string) (*mailbox, error) {
	var out mailbox
	if err := deployDo(ctx, http.MethodPost, deployURL(ctx, "/mymail"), map[string]interface{}{"password": password}, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// getMailbox returns the workspace's mailbox, or nil if it has none.
func getMailbox(ctx *osaasclient.Context) (*mailbox, error) {
	var out mailbox
	if err := deployDo(ctx, http.MethodGet, deployURL(ctx, "/mymail"), nil, &out); err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	if out.Email == "" {
		return nil, nil
	}
	return &out, nil
}

// setMailboxPassword changes the password. The platform allows five changes an hour.
func setMailboxPassword(ctx *osaasclient.Context, password string) error {
	return deployDo(ctx, http.MethodPost, deployURL(ctx, "/mymail/password"), map[string]interface{}{"password": password}, nil)
}

func deleteMailbox(ctx *osaasclient.Context) error {
	err := deployDo(ctx, http.MethodDelete, deployURL(ctx, "/mymail"), nil, nil)
	if err != nil && isNotFound(err) {
		return nil
	}
	return err
}
