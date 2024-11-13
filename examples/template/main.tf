terraform {
  required_providers {
    osc = {
      source = "eyevinn.se/terraform/osc"
    }
  }
}

variable "osc_pat" {
  type = string
}

variable "osc_environment" {
  type = string
  default = "prod"
}

provider "osc" {
  pat = var.osc_pat
  environment = var.osc_environment
}

resource "osc_qr_generator_resource" "example" {
  name = "qr_template"
  goto_url = "https://google.se"
}

output "generate" {
	value = "${osc_qr_generator_resource.example.url}/generate"
}
