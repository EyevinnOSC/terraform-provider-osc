package provider

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

// paramValues holds the parameter values known at validation time. A key present in
// unknown means the key is configured but its value is not yet known (plan time).
type paramValues struct {
	values  map[string]string
	unknown map[string]bool
}

func newParamValues() paramValues {
	return paramValues{values: map[string]string{}, unknown: map[string]bool{}}
}

func (p paramValues) keys() []string {
	keys := make([]string, 0, len(p.values)+len(p.unknown))
	for k := range p.values {
		keys = append(keys, k)
	}
	for k := range p.unknown {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (p paramValues) has(key string) bool {
	_, ok := p.values[key]
	return ok || p.unknown[key]
}

func findOption(service *catalogService, name string) *serviceOption {
	for i := range service.ServiceInstanceOptions {
		if service.ServiceInstanceOptions[i].Name == name {
			return &service.ServiceInstanceOptions[i]
		}
	}
	return nil
}

// describeOptions renders the accepted parameter schema of a service so that error
// messages carry everything needed to fix the configuration.
func describeOptions(service *catalogService) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Accepted parameters for service %q:\n", service.ServiceId)
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Name == "name" {
			continue
		}
		req := "optional"
		if opt.Mandatory {
			req = "required"
		}
		fmt.Fprintf(&b, "  - %s (%s, %s)", opt.Name, opt.Type, req)
		if len(opt.Enum) > 0 {
			fmt.Fprintf(&b, " one of [%s]", strings.Join(opt.Enum, ", "))
		}
		if opt.Description != "" {
			fmt.Fprintf(&b, ": %s", strings.TrimSpace(opt.Description))
		}
		b.WriteString("\n")
	}
	if len(service.ServiceInstanceOptions) <= 1 {
		b.WriteString("  (this service takes no parameters besides name)\n")
	}
	return b.String()
}

// validateInstanceName checks the instance name against the service's name option.
func validateInstanceName(service *catalogService, name string) diag.Diagnostics {
	var diags diag.Diagnostics
	opt := findOption(service, "name")
	if opt == nil {
		return diags
	}
	if msg := validateOptionValue(opt, name); msg != "" {
		diags.AddAttributeError(path.Root("name"), "Invalid instance name", msg)
	}
	return diags
}

func validateOptionValue(opt *serviceOption, value string) string {
	switch opt.Type {
	case "boolean":
		if value != "true" && value != "false" {
			return fmt.Sprintf("Parameter %q is a boolean and must be \"true\" or \"false\", got %q.", opt.Name, value)
		}
	case "enum":
		if len(opt.Enum) > 0 {
			ok := false
			for _, e := range opt.Enum {
				if e == value {
					ok = true
					break
				}
			}
			if !ok {
				return fmt.Sprintf("Parameter %q must be one of [%s], got %q.", opt.Name, strings.Join(opt.Enum, ", "), value)
			}
		}
	}
	if opt.RegexValidator != "" {
		re, err := regexp.Compile(opt.RegexValidator)
		if err == nil && !re.MatchString(value) {
			return fmt.Sprintf("Parameter %q value %q does not match the required pattern %s.", opt.Name, value, opt.RegexValidator)
		}
	}
	return ""
}

// validateParameters checks configured parameters against the service's option schema.
// Both the regular and sensitive parameter maps are checked; the combined key set is
// checked for unknown keys, duplicates and missing required options.
func validateParameters(service *catalogService, params, sensitive paramValues) diag.Diagnostics {
	var diags diag.Diagnostics
	schema := describeOptions(service)

	// Duplicate keys across the two maps.
	for _, k := range params.keys() {
		if sensitive.has(k) {
			diags.AddAttributeError(path.Root("sensitive_parameters"), "Parameter defined twice",
				fmt.Sprintf("Parameter %q is set in both parameters and sensitive_parameters. Set it in exactly one of them.", k))
		}
	}

	// name is a top level attribute, never a parameter.
	for attr, pv := range map[string]paramValues{"parameters": params, "sensitive_parameters": sensitive} {
		if pv.has("name") {
			diags.AddAttributeError(path.Root(attr), "Instance name must not be a parameter",
				"Set the instance name with the top level name attribute instead of parameters[\"name\"].")
		}
	}

	// Unknown keys, with nearest-match suggestions.
	for attr, pv := range map[string]paramValues{"parameters": params, "sensitive_parameters": sensitive} {
		for _, k := range pv.keys() {
			if k == "name" {
				continue
			}
			if findOption(service, k) == nil {
				msg := fmt.Sprintf("Service %q has no parameter named %q.", service.ServiceId, k)
				if s := suggestOption(service, k); s != "" {
					msg += fmt.Sprintf(" Did you mean %q? Parameter names are case sensitive.", s)
				}
				diags.AddAttributeError(path.Root(attr), "Unknown parameter", msg+"\n\n"+schema)
			}
		}
	}

	// Required options.
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Name == "name" || !opt.Mandatory {
			continue
		}
		if !params.has(opt.Name) && !sensitive.has(opt.Name) {
			diags.AddAttributeError(path.Root("parameters"), "Missing required parameter",
				fmt.Sprintf("Service %q requires parameter %q.\n\n%s", service.ServiceId, opt.Name, schema))
		}
	}

	// Values that are known now.
	for attr, pv := range map[string]paramValues{"parameters": params, "sensitive_parameters": sensitive} {
		for k, v := range pv.values {
			opt := findOption(service, k)
			if opt == nil {
				continue
			}
			if msg := validateOptionValue(opt, v); msg != "" {
				diags.AddAttributeError(path.Root(attr), "Invalid parameter value", msg)
			}
		}
	}

	// Sensitive options placed in the non-sensitive map. A {{secrets.*}} reference is
	// not itself sensitive, so it is fine in plain parameters.
	for _, k := range params.keys() {
		if v, known := params.values[k]; known && isSecretRef(v) {
			continue
		}
		if opt := findOption(service, k); opt != nil && opt.Sensitive {
			diags.AddAttributeWarning(path.Root("parameters"), "Sensitive parameter in plain parameters",
				fmt.Sprintf("Parameter %q is marked sensitive by the service. Move it to sensitive_parameters to keep it out of plan output.", k))
		}
	}

	return diags
}

// buildInstanceBody creates the request body for create and update calls, coercing
// values to the types the service declares.
func buildInstanceBody(service *catalogService, name string, params, sensitive map[string]string) map[string]interface{} {
	body := map[string]interface{}{"name": name}
	add := func(m map[string]string) {
		for k, v := range m {
			opt := findOption(service, k)
			if opt != nil && opt.Type == "boolean" {
				body[k] = v == "true"
				continue
			}
			body[k] = v
		}
	}
	add(params)
	add(sensitive)
	return body
}

// suggestOption returns the closest option name to the given key, or "" if nothing is close.
func suggestOption(service *catalogService, key string) string {
	best := ""
	bestDist := -1
	lower := strings.ToLower(key)
	for _, opt := range service.ServiceInstanceOptions {
		if opt.Name == "name" {
			continue
		}
		if strings.ToLower(opt.Name) == lower {
			return opt.Name
		}
		d := levenshtein(lower, strings.ToLower(opt.Name))
		if bestDist == -1 || d < bestDist {
			best, bestDist = opt.Name, d
		}
	}
	if best == "" {
		return ""
	}
	limit := len(key) / 2
	if limit < 2 {
		limit = 2
	}
	if bestDist <= limit {
		return best
	}
	return ""
}

func levenshtein(a, b string) int {
	ra, rb := []rune(a), []rune(b)
	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(rb)]
}

// isSecretRef reports whether a value is an OSC secret reference such as {{secrets.name}}.
func isSecretRef(v string) bool {
	v = strings.TrimSpace(v)
	return strings.HasPrefix(v, "{{secrets.") && strings.HasSuffix(v, "}}")
}
