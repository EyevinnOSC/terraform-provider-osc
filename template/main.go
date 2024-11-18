package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"strings"

	osaasclient "github.com/EyevinnOSC/client-go"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)


type Auth struct {
	Header string
	Value  string
}

func createFetch(url string, method string, body *bytes.Buffer, target interface{}, auth Auth) error {
	client := &http.Client{}
	if body == nil {
		body = &bytes.Buffer{}
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return err
	}
	req.Header.Set(auth.Header, auth.Value)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}


	if target != nil {
		value, ok := resp.Header["Content-Type"]
		if ok && strings.HasPrefix(value[0], "application/json") {
			if err := json.Unmarshal(responseBytes, target); err != nil {
				return err
			}
		}
	}

	return nil
}

type OscContext struct {
	PersonalAccessToken string
	Environment         string
	ApiKey				string
}

// Define the structs to represent the JSON structure
type InputParameter struct {
	Name            string `json:"name"`
	NameInteral     string `json:"Name"`
	Type            string `json:"type"`
	Flag            string `json:"flag"`
	SchemaAttribute string `json:"schemaAttribute"`
	Value           string `json:"value"`
}

type InstanceParameter struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Resource struct {
	ObjectName        string               `json:"_ObjectName"`
	ResourceName      string               `json:"resourceName"`
	InputParameters   []InputParameter     `json:"inputParameters"`
	ServiceID         string               `json:"serviceId"`
	InstanceParameters []InstanceParameter `json:"instanceParameters"`
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
func ToSnakeCase(str string) string {
    snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
    snake  = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
    return strings.ToLower(snake)
}

var caser = cases.Title(language.English)
func main() {
	ctx := &OscContext{
		Environment: "prod",
		PersonalAccessToken:  os.Getenv("OSC_ACCESS_TOKEN"),
		ApiKey: os.Getenv("OSC_API_KEY"),
	}


	serviceURL := fmt.Sprintf("https://catalog.svc.prod.osaas.io/service")
	var services []osaasclient.Service
	err := createFetch(serviceURL, "GET", nil, &services, Auth{"Authorization", fmt.Sprintf("Bearer %s", ctx.ApiKey)})
	if err != nil {
		fmt.Println(err)
		return
	}	
	for _, element := range services {
		fmt.Println(element.ServiceId)
		var inputParameters []InputParameter
		var instanceParameters []InstanceParameter
		for _, inputParameter := range element.ServiceInstanceOptions {
			var t = "types.String"
			var flag = "Required"
			if inputParameter.Name == "name" {
				continue
			}
			if inputParameter.Type != "string" {
				continue
			}
			if inputParameter.Required != true {
				flag = "Optional"
			}
			var nameInternal = caser.String(inputParameter.Name)
			var i = InputParameter{
					Name:            ToSnakeCase(inputParameter.Name),
					NameInteral:     nameInternal,
					Type:            t,
					Flag:            flag,
					SchemaAttribute: "StringAttribute",
					Value:           fmt.Sprintf("plan.%s", nameInternal),
			}
			inputParameters = append(inputParameters, i)

			var instanceParameter = InstanceParameter{
				Name: inputParameter.Name,
				Value: fmt.Sprintf("plan.%s.ValueString()", nameInternal),
			}
			instanceParameters = append(instanceParameters, instanceParameter)
		}
        resourceName := fmt.Sprintf("osc_%s_resource", strings.ReplaceAll(element.ServiceId, "-", "_"))
		resource := Resource{
			ObjectName: strings.ReplaceAll(element.ServiceId, "-", ""),
			ResourceName: resourceName,
			InputParameters: inputParameters,
			ServiceID: element.ServiceId,
			InstanceParameters: instanceParameters, 
		}

		// Marshal the struct to JSON
		jsonData, err := json.MarshalIndent(resource, "", "    ")
		if err != nil {
			fmt.Println("Error marshaling to JSON:", err)
			return
		}

		// Create or open the output file
		outputFile, err := os.Create(fmt.Sprintf("../internal/provider/%s.go", resourceName))
		if err != nil {
			fmt.Println("Error creating output file:", err)
			return
		}
		defer outputFile.Close()

		cmd := exec.Command("mustache", "-", "template/resource.tpl.go")
		cmd.Stdin = bytes.NewReader(jsonData)
		cmd.Stdout = outputFile
		if err := cmd.Run(); err != nil {
			fmt.Println("Error running mustache command:", err)
			return
		}
	}
}
