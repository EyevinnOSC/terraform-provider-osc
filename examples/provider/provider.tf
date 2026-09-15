terraform {
  required_providers {
    osc = {
      source = "EyevinnOSC/osc"
    }
  }
}

# The token is resolved from the pat attribute, then the OSC_ACCESS_TOKEN
# environment variable, then the token saved by `osc login`.
# workspace (or OSC_WORKSPACE) pins the target; a token for any other workspace is refused.
provider "osc" {
  environment = "prod"
  workspace   = "mytenant"
}
