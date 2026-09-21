package provider

import (
	"strings"
	"testing"
)

func testService() *catalogService {
	return &catalogService{
		ServiceId:   "eyevinn-test-svc",
		ServiceType: "instance",
		ServiceInstanceOptions: []serviceOption{
			{Name: "name", Type: "string", Mandatory: true, RegexValidator: `^\w+$`},
			{Name: "RedisUrl", Type: "string", Mandatory: true, Description: "Redis endpoint"},
			{Name: "RedisQueue", Type: "string"},
			{Name: "Mode", Type: "enum", Enum: []string{"fast", "slow"}},
			{Name: "Debug", Type: "boolean"},
			{Name: "ApiKey", Type: "string", Sensitive: true},
		},
	}
}

func pv(values map[string]string, unknown ...string) paramValues {
	p := newParamValues()
	for k, v := range values {
		p.values[k] = v
	}
	for _, k := range unknown {
		p.unknown[k] = true
	}
	return p
}

func TestValidateParametersOK(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"RedisUrl": "redis://x", "Mode": "fast", "Debug": "true"}), pv(map[string]string{"ApiKey": "s3cret"}))
	if diags.HasError() {
		t.Fatalf("unexpected errors: %v", diags)
	}
	if diags.WarningsCount() != 0 {
		t.Fatalf("unexpected warnings: %v", diags)
	}
}

func TestValidateParametersUnknownKeySuggests(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"redisurl": "redis://x"}), newParamValues())
	if !diags.HasError() {
		t.Fatal("expected error for unknown key")
	}
	found := false
	for _, d := range diags.Errors() {
		if strings.Contains(d.Detail(), `Did you mean "RedisUrl"`) && strings.Contains(d.Detail(), "Accepted parameters") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected suggestion and schema in error, got %v", diags)
	}
}

func TestValidateParametersMissingRequired(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"RedisQueue": "q"}), newParamValues())
	if !diags.HasError() || !strings.Contains(diags.Errors()[0].Detail(), `requires parameter "RedisUrl"`) {
		t.Fatalf("expected missing required error, got %v", diags)
	}
}

func TestValidateParametersUnknownValueSatisfiesRequired(t *testing.T) {
	diags := validateParameters(testService(), pv(nil, "RedisUrl"), newParamValues())
	if diags.HasError() {
		t.Fatalf("unknown value should satisfy required check, got %v", diags)
	}
}

func TestValidateParametersEnumBoolRegex(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"RedisUrl": "x", "Mode": "medium", "Debug": "yes"}), newParamValues())
	if diags.ErrorsCount() != 2 {
		t.Fatalf("expected 2 errors, got %v", diags)
	}
	if d := validateInstanceName(testService(), "bad-name"); !d.HasError() {
		t.Fatal("expected regex error for name")
	}
	if d := validateInstanceName(testService(), "good_name1"); d.HasError() {
		t.Fatalf("unexpected name error: %v", d)
	}
}

func TestValidateParametersNameAndDuplicates(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"name": "x", "RedisUrl": "a"}), pv(map[string]string{"RedisUrl": "b"}))
	if diags.ErrorsCount() != 2 {
		t.Fatalf("expected name and duplicate errors, got %v", diags)
	}
}

func TestValidateParametersSensitiveWarning(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"RedisUrl": "a", "ApiKey": "k"}), newParamValues())
	if diags.HasError() || diags.WarningsCount() != 1 {
		t.Fatalf("expected one warning, got %v", diags)
	}
}

func TestValidateParametersSecretRefNoWarning(t *testing.T) {
	diags := validateParameters(testService(), pv(map[string]string{"RedisUrl": "a", "ApiKey": "{{secrets.mykey}}"}), newParamValues())
	if diags.HasError() || diags.WarningsCount() != 0 {
		t.Fatalf("expected no diagnostics for secret ref, got %v", diags)
	}
}

func TestBuildInstanceBody(t *testing.T) {
	body := buildInstanceBody(testService(), "inst", map[string]string{"Debug": "true", "RedisUrl": "r"}, map[string]string{"ApiKey": "k"})
	if body["name"] != "inst" || body["Debug"] != true || body["RedisUrl"] != "r" || body["ApiKey"] != "k" {
		t.Fatalf("unexpected body %v", body)
	}
}

func TestFlattenInstance(t *testing.T) {
	out := flattenInstance(map[string]interface{}{
		"name": "x", "n": float64(3), "b": true, "nil": nil,
		"_links": map[string]interface{}{"self": map[string]interface{}{"href": "/x"}},
	})
	if out["name"] != "x" || out["n"] != "3" || out["b"] != "true" || out["nil"] != "" || out["_links"] != `{"self":{"href":"/x"}}` {
		t.Fatalf("unexpected flatten %v", out)
	}
}

func TestSuggestOption(t *testing.T) {
	s := testService()
	if got := suggestOption(s, "redis_url"); got != "RedisUrl" {
		t.Fatalf("got %q", got)
	}
	if got := suggestOption(s, "CompletelyDifferentThing"); got != "" {
		t.Fatalf("expected no suggestion, got %q", got)
	}
}
