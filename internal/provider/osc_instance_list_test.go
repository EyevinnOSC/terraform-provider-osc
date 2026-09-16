package provider

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestParametersFromInstance(t *testing.T) {
	service := &catalogService{
		ServiceId: "apache-couchdb",
		ServiceInstanceOptions: []serviceOption{
			{Name: "name", Type: "string"},
			{Name: "AdminPassword", Type: "string", Sensitive: true},
			{Name: "Replicas", Type: "string"},
			{Name: "Debug", Type: "boolean"},
			{Name: "Unset", Type: "string"},
		},
	}
	instance := map[string]interface{}{
		"name":          "openlive",
		"url":           "https://ludde-openlive.apache-couchdb.auto.prod-se.osaas.io",
		"AdminPassword": "s3cret",
		"Replicas":      float64(3),
		"Debug":         true,
		"status":        "running",
	}

	params, sensitive := parametersFromInstance(service, instance)

	wantParams := map[string]string{"Replicas": "3", "Debug": "true"}
	if !reflect.DeepEqual(params, wantParams) {
		t.Errorf("parameters = %v, want %v", params, wantParams)
	}
	wantSensitive := map[string]string{"AdminPassword": "s3cret"}
	if !reflect.DeepEqual(sensitive, wantSensitive) {
		t.Errorf("sensitive_parameters = %v, want %v", sensitive, wantSensitive)
	}
}

func TestKeepStateWhenUnset(t *testing.T) {
	ctx := context.Background()
	state := stringsToMap(map[string]string{"AdminPassword": "s3cret"})
	configured := stringsToMap(map[string]string{"AdminPassword": "new"})
	null := types.MapNull(types.StringType)
	empty := types.MapValueMust(types.StringType, map[string]attr.Value{})

	cases := []struct {
		name          string
		config, state types.Map
		want          types.Map
	}{
		{"unset keeps state", null, state, state},
		{"unset with no state stays null", null, null, null},
		{"unset with unknown state stays null", null, types.MapUnknown(types.StringType), null},
		{"configured value wins", configured, state, configured},
		{"empty map clears", empty, state, empty},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.MapRequest{ConfigValue: tc.config, StateValue: tc.state, PlanValue: tc.config}
			if tc.config.IsNull() {
				// The framework plans computed attributes as unknown when they are unset and something changes.
				req.PlanValue = types.MapUnknown(types.StringType)
			}
			resp := &planmodifier.MapResponse{PlanValue: req.PlanValue}
			keepStateWhenUnset{}.PlanModifyMap(ctx, req, resp)
			if !resp.PlanValue.Equal(tc.want) {
				t.Errorf("plan = %v, want %v", resp.PlanValue, tc.want)
			}
		})
	}
}
