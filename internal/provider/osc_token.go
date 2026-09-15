package provider

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// tokenClaims are the claims of an OSC access token that the provider cares about.
// The signature is not verified; the token is only decoded to learn which workspace
// and user it belongs to and whether it has expired. The OSC APIs do the real check.
type tokenClaims struct {
	TenantID string `json:"tenantId"`
	UserID   string `json:"userId"`
	PatID    string `json:"patId"`
	Type     string `json:"type"`
	IssuedAt int64  `json:"iat"`
	Expires  int64  `json:"exp"`
}

func decodeTokenClaims(token string) (*tokenClaims, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return nil, errors.New("token is not a JWT")
	}
	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, fmt.Errorf("token payload is not valid base64: %w", err)
	}
	var claims tokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, fmt.Errorf("token payload is not valid JSON: %w", err)
	}
	if claims.TenantID == "" {
		return nil, errors.New("token has no tenantId claim")
	}
	return &claims, nil
}

func (c *tokenClaims) expired(now time.Time) bool {
	return c.Expires != 0 && now.Unix() >= c.Expires
}

// cliTokenPath returns where the OSC CLI stores the token from `osc login` for the
// given environment: ~/.osc/token for prod, ~/.osc/token-<env> otherwise.
func cliTokenPath(environment string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	suffix := ""
	if environment != "" && environment != "prod" {
		suffix = "-" + environment
	}
	return filepath.Join(home, ".osc", "token"+suffix), nil
}

func readCLIToken(environment string) (string, string, error) {
	path, err := cliTokenPath(environment)
	if err != nil {
		return "", path, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", path, err
	}
	return strings.TrimSpace(string(data)), path, nil
}

// resolveToken picks the access token to use, in order of precedence: the explicit
// provider attribute, the OSC_ACCESS_TOKEN environment variable, then the token saved
// by `osc login`. The returned source describes where the token came from.
func resolveToken(configured, environment string) (token string, source string, err error) {
	if configured != "" {
		return configured, "the provider pat attribute", nil
	}
	if env := os.Getenv("OSC_ACCESS_TOKEN"); env != "" {
		return env, "the OSC_ACCESS_TOKEN environment variable", nil
	}
	cli, path, readErr := readCLIToken(environment)
	if readErr == nil && cli != "" {
		return cli, fmt.Sprintf("the OSC CLI login at %s", path), nil
	}
	return "", "", fmt.Errorf("no access token found. Set the pat attribute, set the OSC_ACCESS_TOKEN environment variable, or run `osc login` (looked for %s)", path)
}
