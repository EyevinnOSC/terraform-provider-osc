# A parameter store for an app's environment. OSC creates a Valkey instance for it
# and holds its encryption and API keys, so it can store secrets.
resource "osc_parameter_store" "api" {
  name = "myapiconfig"
}
