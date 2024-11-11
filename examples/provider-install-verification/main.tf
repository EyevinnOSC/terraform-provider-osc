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

variable "aws_keyid" {
	type = string
}

variable "aws_secret" {
	type = string
}

variable "aws_output" {
	type = string
	default = "s3://lab-testcontent-store/tftest/"
}

provider "osc" {
  pat = var.osc_pat
  environment = var.osc_environment
}

resource "osc_encore_instance" "example" {
  name = "ggexample"
  profiles_url = "https://raw.githubusercontent.com/Eyevinn/encore-test-profiles/refs/heads/main/profiles.yml"
}

resource "osc_valkey_instance" "example" {
  name = "ggexample"
}

resource "osc_encore_callback_instance" "example" {
	name = "ggexample"
	redis_url = format("redis://%s:%s", osc_valkey_instance.example.external_ip, osc_valkey_instance.example.external_port)
	encore_url = "https://eyevinnlab-ggexample.encore.prod.osaas.io"
	redis_queue = "transfer"
}

resource "osc_encore_transfer_instance" "example" {
	name = "ggexample"
	redis_url = osc_encore_callback_instance.example.redis_url
	redis_queue = osc_encore_callback_instance.example.redis_queue
	output = var.aws_output
	aws_keyid = var.aws_keyid
	aws_secret = var.aws_secret
	osc_token = var.osc_pat
}

