package provider

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func makeToken(t *testing.T, claims map[string]interface{}) string {
	t.Helper()
	b, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	enc := base64.RawURLEncoding.EncodeToString
	return enc([]byte(`{"alg":"HS256","typ":"JWT"}`)) + "." + enc(b) + "." + enc([]byte("sig"))
}

func TestDecodeTokenClaims(t *testing.T) {
	tok := makeToken(t, map[string]interface{}{"tenantId": "acme", "userId": "u1", "exp": 1000})
	c, err := decodeTokenClaims(tok)
	if err != nil || c.TenantID != "acme" || c.UserID != "u1" {
		t.Fatalf("unexpected %v %v", c, err)
	}
	if !c.expired(time.Unix(1000, 0)) || c.expired(time.Unix(999, 0)) {
		t.Fatal("expiry check wrong")
	}
	if _, err := decodeTokenClaims("not-a-jwt"); err == nil {
		t.Fatal("expected error for non JWT")
	}
	if _, err := decodeTokenClaims(makeToken(t, map[string]interface{}{"userId": "u1"})); err == nil {
		t.Fatal("expected error for missing tenantId")
	}
}

func TestResolveTokenPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("OSC_ACCESS_TOKEN", "")

	if _, _, err := resolveToken("", "prod"); err == nil {
		t.Fatal("expected error with no token anywhere")
	}

	if err := os.MkdirAll(filepath.Join(home, ".osc"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".osc", "token"), []byte("cli-prod\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".osc", "token-dev"), []byte("cli-dev"), 0o600); err != nil {
		t.Fatal(err)
	}
	if tok, _, _ := resolveToken("", "prod"); tok != "cli-prod" {
		t.Fatalf("expected cli prod token, got %q", tok)
	}
	if tok, _, _ := resolveToken("", "dev"); tok != "cli-dev" {
		t.Fatalf("expected cli dev token, got %q", tok)
	}
	t.Setenv("OSC_ACCESS_TOKEN", "from-env")
	if tok, _, _ := resolveToken("", "prod"); tok != "from-env" {
		t.Fatalf("expected env token, got %q", tok)
	}
	if tok, _, _ := resolveToken("explicit", "prod"); tok != "explicit" {
		t.Fatalf("expected explicit token, got %q", tok)
	}
}
