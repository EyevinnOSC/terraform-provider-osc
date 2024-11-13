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

resource "osc_eyevinn_cast_receiver_resource" "example" {
  name = "template"
  title = "my title"
}


output "url" {
	value = osc_eyevinn_cast_receiver_resource.example.url 
}
