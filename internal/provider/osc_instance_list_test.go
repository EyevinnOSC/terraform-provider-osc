package provider

import (
	"reflect"
	"testing"
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
