# Running the OSC Terraform Resource Generator
```sh
cd template
OSC_API_KEY=<SECRET> go run .
```
# PoC for generating terraform resources using OSC catalog API
The generation script will make a request to the catalog-API to get all available services.
It will attempt to create a terraform resource for each service.
It will need to be able to handle all different input parameter datatyes e.g. string, int, enum etc.

Using the Catalog response it will create a `context` which is fed into the template engine.
```json
// Example context
{
	"_ObjectName": "QrGeneratorResource",
	"resourceName": "osc_qr_generator_resource",
	"inputParameters": [
		{"name": "goto_url", "Name": "GotoUrl", "type": "types.String", "flag": "Required", "schemaAttribute": "StringAttribute", "value": "plan.GotoUrl"}
	],
	"serviceId": "eyevinn-qr-generator",
	"instanceParameters": [
		{"name": "GotoUrl", "value": "plan.GotoUrl.ValueString()"}
	]
}
```

The template engine is invoked like so:
```sh
outputFile, err := os.Create(fmt.Sprintf("../internal/provider/%s.go", resourceName))

cmd := exec.Command("mustache", "-", "template/resource.tpl.go")
cmd.Stdin = bytes.NewReader(jsonData)
cmd.Stdout = outputFile
```

In the go-file generated there will be a init-function which will add itself to the global resource list:
```go
func init() {
	RegisteredResources = append(RegisteredResources, Neweyevinncastreceiver)
}
```

Then after rebuilding / reinstalling the provider the new auto-generated resources will become available.
```tf
resource "osc_eyevinn_cast_receiver_resource" "example" {
  name = "template"
  title = "my title"
}
```
