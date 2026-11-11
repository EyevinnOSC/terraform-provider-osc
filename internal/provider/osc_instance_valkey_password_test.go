package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func valkeyPlan(serviceID, name string, params, sensitive types.Map) InstanceResourceModel {
	return InstanceResourceModel{
		ServiceID:           types.StringValue(serviceID),
		Name:                types.StringValue(name),
		Parameters:          params,
		SensitiveParameters: sensitive,
		AsSecrets:           types.BoolValue(true),
		SecretNames:         types.MapUnknown(types.StringType),
	}
}

func TestPlanValkeyPassword(t *testing.T) {
	null := types.MapNull(types.StringType)
	existing := valkeyPlan(valkeyServiceID, "db", null, stringsToMap(map[string]string{"Password": "old"}))
	passwordless := valkeyPlan(valkeyServiceID, "db", null, null)

	t.Run("create injects an unknown Password", func(t *testing.T) {
		got, injected, diags := planValkeyPassword(passwordless, nil)
		if diags.HasError() || !injected {
			t.Fatalf("injected %v, %v", injected, diags)
		}
		pw := got.Elements()[valkeyPasswordKey].(types.String)
		if !pw.IsUnknown() {
			t.Errorf("Password = %v, want unknown", pw)
		}
	})
	t.Run("create keeps other sensitive keys", func(t *testing.T) {
		p := valkeyPlan(valkeyServiceID, "db", null, stringsToMap(map[string]string{"Other": "x"}))
		got, injected, _ := planValkeyPassword(p, nil)
		if !injected || got.Elements()["Other"].(types.String).ValueString() != "x" {
			t.Errorf("got %v injected %v", got, injected)
		}
	})
	t.Run("empty map is filled", func(t *testing.T) {
		p := valkeyPlan(valkeyServiceID, "db", null, stringsToMap(nil))
		if _, injected, _ := planValkeyPassword(p, nil); !injected {
			t.Error("not injected for {}")
		}
	})
	t.Run("supplied sensitive Password is untouched", func(t *testing.T) {
		got, injected, _ := planValkeyPassword(existing, nil)
		if injected || !got.Equal(existing.SensitiveParameters) {
			t.Errorf("injected %v, got %v", injected, got)
		}
	})
	t.Run("supplied unknown sensitive Password is untouched", func(t *testing.T) {
		p := valkeyPlan(valkeyServiceID, "db", null, types.MapValueMust(types.StringType, map[string]attr.Value{"Password": types.StringUnknown()}))
		if _, injected, _ := planValkeyPassword(p, nil); injected {
			t.Error("injected over a Password that is only unknown")
		}
	})
	t.Run("supplied plain Password is untouched", func(t *testing.T) {
		p := valkeyPlan(valkeyServiceID, "db", stringsToMap(map[string]string{"Password": "plain"}), null)
		if _, injected, _ := planValkeyPassword(p, nil); injected {
			t.Error("injected despite parameters.Password")
		}
	})
	t.Run("non Valkey service is untouched", func(t *testing.T) {
		p := valkeyPlan("eyevinn-other", "db", null, null)
		if _, injected, _ := planValkeyPassword(p, nil); injected {
			t.Error("injected for another service")
		}
	})
	t.Run("in-place update of a password-less store injects nothing", func(t *testing.T) {
		state := passwordless
		plan := valkeyPlan(valkeyServiceID, "db", stringsToMap(map[string]string{"x": "y"}), null)
		if _, injected, _ := planValkeyPassword(plan, &state); injected {
			t.Error("injected on in-place update")
		}
	})
	t.Run("replace by name injects", func(t *testing.T) {
		state := valkeyPlan(valkeyServiceID, "old", null, null)
		if _, injected, _ := planValkeyPassword(passwordless, &state); !injected {
			t.Error("not injected on replacement")
		}
	})
	t.Run("replace from another service injects", func(t *testing.T) {
		state := valkeyPlan("eyevinn-other", "db", null, null)
		if _, injected, _ := planValkeyPassword(passwordless, &state); !injected {
			t.Error("not injected on service change")
		}
	})
	t.Run("later plan sees state password and injects nothing", func(t *testing.T) {
		state := existing
		if _, injected, _ := planValkeyPassword(existing, &state); injected {
			t.Error("injected on a later plan")
		}
	})
	t.Run("unknown sensitive_parameters fails loud naming the Password", func(t *testing.T) {
		p := valkeyPlan(valkeyServiceID, "db", null, types.MapUnknown(types.StringType))
		_, injected, diags := planValkeyPassword(p, nil)
		if injected || !diags.HasError() || !strings.Contains(diags[0].Detail(), "sensitive_parameters.Password") {
			t.Errorf("injected %v, diags %v", injected, diags)
		}
	})
}

// TestValkeyPlanIsDeterministic runs the injection against the real schema twice, as
// Terraform does at plan and at apply, and checks both plans are identical, so core
// does not reject the apply with "inconsistent final plan".
func TestValkeyPlanIsDeterministic(t *testing.T) {
	ctx := context.Background()
	r := &InstanceResource{}
	var sr resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &sr)

	planOnce := func() tftypes.Value {
		model := valkeyPlan(valkeyServiceID, "db", types.MapNull(types.StringType), types.MapNull(types.StringType))
		pl := tfsdk.Plan{Schema: sr.Schema}
		model.ID, model.URL = types.StringUnknown(), types.StringUnknown()
		model.InternalHost = types.StringUnknown()
		model.ExternalIP, model.ExternalPort = types.StringUnknown(), types.Int64Unknown()
		model.Instance = types.MapUnknown(types.StringType)
		model.Suspended = types.BoolUnknown()
		model.PublicAccess = types.BoolUnknown()
		model.WaitForReady, model.AllowSuspend, model.UseLatest = types.BoolValue(true), types.BoolValue(false), types.BoolValue(false)
		model.Tags = types.SetNull(types.StringType)
		if d := pl.Set(ctx, &model); d.HasError() {
			t.Fatalf("plan: %v", d)
		}
		sensitive, injected, d := planValkeyPassword(model, nil)
		if d.HasError() || !injected {
			t.Fatalf("injected %v: %v", injected, d)
		}
		if d := pl.SetAttribute(ctx, path.Root("sensitive_parameters"), sensitive); d.HasError() {
			t.Fatalf("set: %v", d)
		}
		return pl.Raw
	}
	a, b := planOnce(), planOnce()
	if !a.Equal(b) {
		t.Errorf("plans differ:\n%v\n%v", a, b)
	}
}

func TestResolveGeneratedPasswordStoresSecretReference(t *testing.T) {
	null := types.MapNull(types.StringType)
	plan := valkeyPlan(valkeyServiceID, "db", null, null)
	m, _, _ := planValkeyPassword(plan, nil)
	plan.SensitiveParameters = m

	if d := resolveGeneratedPassword(&plan); d.HasError() {
		t.Fatalf("resolve: %v", d)
	}
	sensitive := mapToStrings(plan.SensitiveParameters)
	pw := sensitive["Password"]
	if len(pw) < 40 || !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(pw) {
		t.Fatalf("generated password %q is not a long URL safe value", pw)
	}
	names := mapToStrings(plan.SecretNames)
	if names["Password"] == "" {
		t.Fatalf("secret_names = %v", names)
	}

	var mu sync.Mutex
	stored := map[string]string{}
	ctx := deployManager(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		var body struct {
			Secrets []struct {
				SecretName string `json:"secretName"`
				SecretData string `json:"secretData"`
			} `json:"secrets"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if r.Method != http.MethodPut || r.URL.Path != "/mysecrets/"+valkeyServiceID+"/batch" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		for _, s := range body.Secrets {
			stored[s.SecretName] = s.SecretData
		}
	}))
	if err := putSecrets(ctx, valkeyServiceID, secretValues(sensitive, names)); err != nil {
		t.Fatalf("putSecrets: %v", err)
	}
	if stored[names["Password"]] != pw {
		t.Errorf("secret %q not stored with the generated value", names["Password"])
	}
	if got := withSecretRefs(sensitive, names)["Password"]; got != "{{secrets."+names["Password"]+"}}" {
		t.Errorf("instance is sent %q, want the secret reference", got)
	}
}

func TestResolveGeneratedPasswordLeavesOthers(t *testing.T) {
	null := types.MapNull(types.StringType)
	given := valkeyPlan(valkeyServiceID, "db", null, stringsToMap(map[string]string{"Password": "mine"}))
	if d := resolveGeneratedPassword(&given); d.HasError() || mapToStrings(given.SensitiveParameters)["Password"] != "mine" {
		t.Errorf("supplied password changed: %v", given.SensitiveParameters)
	}
	other := valkeyPlan("eyevinn-other", "db", null, null)
	resolveGeneratedPassword(&other)
	if !other.SensitiveParameters.IsNull() {
		t.Error("non Valkey plan changed")
	}
	a, _ := generatePassword()
	b, _ := generatePassword()
	if a == b {
		t.Error("passwords repeat")
	}
}
