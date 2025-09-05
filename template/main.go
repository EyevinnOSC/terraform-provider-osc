package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"unicode"

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
	Description		string `json:"description"`
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
	Description			string				`json:"description"`
}

var matchFirstCap = regexp.MustCompile("(.)([A-Z][a-z]+)")
var matchAllCap   = regexp.MustCompile("([a-z0-9])([A-Z])")
func ToSnakeCase(str string) string {
    snake := matchFirstCap.ReplaceAllString(str, "${1}_${2}")
    snake  = matchAllCap.ReplaceAllString(snake, "${1}_${2}")
    return strings.ToLower(snake)
}

func typeMap(t string) string {
	switch (t) {
		case "string":	return "types.String"
		case "boolean": return "bool"
		case "enum":	return "types.String"
		case "list":	return "string"
		default:		return "types.String"
	}
}


func attributeMap(t string) string {
	switch (t) {
		case "boolean":			return "BoolAttribute"
		case "enum":			return "StringAttribute"
		case "string", "list":	return "StringAttribute"
		default:				return "StringAttribute"
	}
}

func flagMap(f bool) string {
	if f == true {
		return "Required"
	}
	return "Optional"
}

func sanitizeToVariableName(input string) string {
	var builder strings.Builder
	first := true

	for _, r := range input {
		if first {
			// Ensure the first character is a letter or underscore
			if unicode.IsLetter(r) || r == '_' {
				builder.WriteRune(r)
				first = false
			}
		} else {
			// Allow letters, digits, and underscores
			if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
				builder.WriteRune(r)
			}
		}
	}

	// Remove any invalid starting character by prepending an underscore if the builder is empty
	if builder.Len() == 0 || !unicode.IsLetter(rune(builder.String()[0])) {
		return "_" + builder.String()
	}

	return builder.String()
}

type Config struct {
	ServiceIgnore []string `json:"serviceIgnore"`
	ServiceIgnoreMap map[string]struct{}
}

func readSeviceIgnoreList() (*Config, error) {
	jsonFile := "config.json"

	data, err := os.ReadFile(jsonFile)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	ignoreMap := make(map[string]struct{}, len(config.ServiceIgnore))
	for _, item := range config.ServiceIgnore {
		ignoreMap[item] = struct{}{}
	}
	config.ServiceIgnoreMap = ignoreMap

	return &config, nil
}

func main() {
	config, err := readSeviceIgnoreList()
	if err != nil {
		fmt.Println(err)
		return
	}

	ctx := &OscContext{
		Environment: "prod",
		PersonalAccessToken:  os.Getenv("OSC_ACCESS_TOKEN"),
		ApiKey: os.Getenv("OSC_API_KEY"),
	}

	serviceURL := fmt.Sprintf("https://catalog.svc.%s.osaas.io/service", ctx.Environment)
	var services []osaasclient.Service
	err = createFetch(serviceURL, "GET", nil, &services, Auth{"Authorization", fmt.Sprintf("Bearer %s", ctx.ApiKey)})
	if err != nil {
		fmt.Println(err)
		return
	}	
	var caser = cases.Title(language.English)
	counter := 1
	for _, element := range services {
		if _, shouldSkip := config.ServiceIgnoreMap[element.ServiceId]; shouldSkip {
			fmt.Println("Skipping:", element.ServiceId)
			continue
		}

		if element.Status != "PUBLISHED" {
			continue
		}
		fmt.Println(counter, element.ServiceId)
		counter++
		var inputParameters []InputParameter
		var instanceParameters []InstanceParameter
		for _, inputParameter := range element.ServiceInstanceOptions {
			var sanitizedName = sanitizeToVariableName(inputParameter.Name)
			var nameInternal = caser.String(sanitizedName)
			var i = InputParameter{
					Name:            ToSnakeCase(sanitizedName),
					NameInteral:     nameInternal,
					Type:            typeMap(inputParameter.Type),
					Flag:            flagMap(inputParameter.Mandatory),
					SchemaAttribute: attributeMap(inputParameter.Type),
					Value:           fmt.Sprintf("plan.%s", nameInternal),
					Description:	 inputParameter.Description,
			}
			inputParameters = append(inputParameters, i)

			var suffix = ""
			if inputParameter.Type == "string" {
				suffix = ".ValueString()"
			}
			var instanceParameter = InstanceParameter{
				Name: inputParameter.Name,
				Value: fmt.Sprintf("plan.%s%s", nameInternal, suffix),
			}
			instanceParameters = append(instanceParameters, instanceParameter)
		}
        resourceName := fmt.Sprintf("osc_%s", strings.ReplaceAll(element.ServiceId, "-", "_"))
		resource := Resource{
			ObjectName: strings.ReplaceAll(element.ServiceId, "-", ""),
			ResourceName: resourceName,
			Description: element.Metadata.Description,
			InputParameters: inputParameters,
			ServiceID: element.ServiceId,
			InstanceParameters: instanceParameters, 
		}

		// Create or open the output file
		outputFile, err := os.Create(fmt.Sprintf("../internal/provider/%s.go", resourceName))
		if err != nil {
			fmt.Println("Error creating output file:", err)
			return
		}
		defer outputFile.Close()

		// Parse and execute the template
		tmpl, err := template.ParseFiles("template/resource.tpl")
		if err != nil {
			fmt.Println("Error parsing template:", err)
			return
		}

		if err := tmpl.Execute(outputFile, resource); err != nil {
			fmt.Println("Error executing template:", err)
			return
		}
	}
}
