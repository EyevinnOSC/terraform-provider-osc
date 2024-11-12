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

resource "osc_testsrc_hls_live_instance" "example" {
  name = "sgai"
}

resource "osc_test_adserver_instance" "example" {
  name = "sgai"
}

resource "osc_sgai_ad_proxy_instance" "example" {
  name = "sgai"
  vast_endpoint = "${osc_test_adserver_instance.example.url}/api/v1/vast?dur=[template.duration]&uid=[template.sessionId]&ps=[template.pod]"
  origin_url = "${osc_testsrc_hls_live_instance.example.url}/loop/master.m3u8"
  insertion_mode = "dynamic" 

}

output "manifest_url" {
	value = "${osc_sgai_ad_proxy_instance.example.url}/loop/master.m3u8"
}
output "proxy_url" {
	value = "${osc_sgai_ad_proxy_instance.example.url}"
}
