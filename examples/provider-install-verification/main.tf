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
  default = "dev"
}

provider "osc" {
  pat = var.osc_pat
  environment = var.osc_environment
}

resource "osc_encore_instance" "example" {
  name = "ggexample"
  profiles_url = "https://raw.githubusercontent.com/Eyevinn/encore-test-profiles/refs/heads/main/profiles.yml"
}