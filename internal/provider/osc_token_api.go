package provider

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	osaasclient "github.com/EyevinnOSC/client-go"
)

// The token service manages fine-grained access tokens (ADR-0068). Its routes are
// authenticated with the personal access token, like the other platform APIs, and the
// environment name is used as it is in the host, as for the deploy manager and the catalog.

// tokenBase is the token service's URL in an environment; tests point it elsewhere.
var tokenBase = func(env string) string { return fmt.Sprintf("https://token.svc.%s.osaas.io", env) }

func tokenURL(ctx *osaasclient.Context, format string, args ...interface{}) string {
	escaped := make([]interface{}, len(args))
	for i, a := range args {
		escaped[i] = url.PathEscape(fmt.Sprint(a))
	}
	return tokenBase(ctx.GetEnvironment()) + fmt.Sprintf(format, escaped...)
}

// The reasons token-service gives in a 404 body.
const (
	reasonTokenNotFound     = "token not found"
	reasonRoleLookupMissing = "role lookup unavailable"
)

// accessTokenRecord is a token record as POST, GET, PUT and PATCH /tokens answer it. Exp is
// in epoch seconds. Token is only in the answer to POST /tokens, which is the only time the
// value is given.
type accessTokenRecord struct {
	PatID         string          `json:"patId"`
	TenantID      string          `json:"tenantId"`
	UserID        string          `json:"userId"`
	Name          string          `json:"name"`
	Description   string          `json:"description"`
	Status        string          `json:"status"`
	Policy        json.RawMessage `json:"policy"`
	PolicyVersion int64           `json:"policyVersion"`
	Enforcement   string          `json:"enforcement"`
	AuditLevel    string          `json:"auditLevel"`
	AuditUntil    string          `json:"auditUntil"`
	LastUsedAt    string          `json:"lastUsedAt"`
	CreatedAt     string          `json:"createdAt"`
	CreatedBy     string          `json:"createdBy"`
	Exp           int64           `json:"exp"`
	Token         string          `json:"token"`
}

// createAccessToken sends one POST /tokens. It is never repeated: a request that failed may
// have minted a token.
func createAccessToken(ctx *osaasclient.Context, body map[string]interface{}) (*accessTokenRecord, error) {
	var out accessTokenRecord
	if err := tokenDo(ctx, http.MethodPost, tokenURL(ctx, "/tokens"), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func getAccessToken(ctx *osaasclient.Context, id string) (*accessTokenRecord, error) {
	var out accessTokenRecord
	if err := tokenDo(ctx, http.MethodGet, tokenURL(ctx, "/tokens/%s", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// replaceAccessTokenPolicy replaces the policy if the stored policyVersion is still
// expectedVersion; a 409 means it is not.
func replaceAccessTokenPolicy(ctx *osaasclient.Context, id string, policy json.RawMessage, expectedVersion int64) (*accessTokenRecord, error) {
	body := map[string]interface{}{"policy": policy, "expectedVersion": expectedVersion}
	var out accessTokenRecord
	if err := tokenDo(ctx, http.MethodPut, tokenURL(ctx, "/tokens/%s/policy", id), body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// patchAccessToken changes name, description, status or enforcement. The token's expiry is
// immutable and the API refuses it, so no caller can send it through here.
func patchAccessToken(ctx *osaasclient.Context, id string, fields map[string]interface{}) (*accessTokenRecord, error) {
	var out accessTokenRecord
	if err := tokenDo(ctx, http.MethodPatch, tokenURL(ctx, "/tokens/%s", id), fields, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func deleteAccessToken(ctx *osaasclient.Context, id string) error {
	return tokenDo(ctx, http.MethodDelete, tokenURL(ctx, "/tokens/%s", id), nil, nil)
}

func tokenDo(ctx *osaasclient.Context, method, rawURL string, body, out interface{}) error {
	h, v := patAuth(ctx)
	return doJSON(method, rawURL, h, v, body, out)
}

// hasTokenReason reports whether err is a 404 from token-service with the given reason.
func hasTokenReason(err error, reason string) bool {
	var ae *apiError
	return errors.As(err, &ae) && ae.StatusCode == http.StatusNotFound && ae.message() == reason
}

// isTokenNotFound is token-service saying the token does not exist (or is not visible to the
// caller). Any other 404, such as the route being off, does not mean that.
func isTokenNotFound(err error) bool { return hasTokenReason(err, reasonTokenNotFound) }

// isRoleLookupUnavailable is token-service's degraded mode: the token exists but the
// caller's role could not be looked up, so it is not shown.
func isRoleLookupUnavailable(err error) bool { return hasTokenReason(err, reasonRoleLookupMissing) }

// isTokenOutcomeUnknown reports whether a failed POST /tokens may still have created the
// token: the request was sent and no answer came back, a plain 500 (token-service can fail after it stored the record, for example while signing), or a gateway answered for the service. A 503 is not in the list: the service answers it only before the token is kept.
func isTokenOutcomeUnknown(err error) bool {
	var ae *apiError
	if errors.As(err, &ae) {
		return ae.StatusCode == http.StatusInternalServerError || ae.StatusCode == http.StatusBadGateway ||
			ae.StatusCode == http.StatusGatewayTimeout
	}
	return true
}

// normalizeJSON re-encodes a JSON document with a stable key order and no insignificant
// whitespace.
func normalizeJSON(raw []byte) (string, error) {
	var v interface{}
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", err
	}
	out, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// sameJSON reports whether two JSON documents are the same value, whatever their formatting
// or key order.
func sameJSON(a, b []byte) bool {
	var x, y interface{}
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		return false
	}
	return reflect.DeepEqual(x, y)
}
