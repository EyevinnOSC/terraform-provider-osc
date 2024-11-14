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


variable "aws_keyid" {
	type = string
}

variable "aws_secret" {
	type = string
}

locals {
	aws_keyid_name = "awsaccesskeyid"
	aws_secret_name = "awssecretaccesskey"
}

variable "aws_output" {
	type = string
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
	encore_url = trimsuffix(osc_encore_instance.example.url, "/")
	redis_queue = "transfer"
}

resource "osc_retransfer" "example" {
	aws_keyid_name = local.aws_keyid_name
	aws_secret_name = local.aws_secret_name
	aws_keyid_value = var.aws_keyid
	aws_secret_value = var.aws_secret

	lifecycle {
		prevent_destroy = true
	}
}


resource "osc_encore_transfer_instance" "example" {
	name = "ggexample"
	redis_url = osc_encore_callback_instance.example.redis_url
	redis_queue = osc_encore_callback_instance.example.redis_queue
	output = var.aws_output
	aws_keyid = local.aws_keyid_name 
	aws_secret = local.aws_secret_name 
	osc_token = var.osc_pat
}


output "encore_url" {
	value = trimsuffix(osc_encore_instance.example.url, "/")
}

output "encore_token" {
	value = osc_encore_instance.example.token
}

output "name" {
	value = osc_encore_instance.example.name
}

output "callback_url" {
	value = trimsuffix(osc_encore_callback_instance.example.url, "/")
}
