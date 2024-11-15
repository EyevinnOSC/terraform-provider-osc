terraform {
  required_providers {
    osc = {
      source = "eyevinn.se/terraform/osc"
    }
  }
}

variable "osc_pat" {
  type = string
  sensitive = true
}

variable "osc_environment" {
  type = string
  default = "prod"
}


variable "aws_keyid" {
	type = string
	sensitive = true
}

variable "aws_secret" {
	type = string
	sensitive = true
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


resource "osc_secret" "keyid" {
	service_ids = ["eyevinn-docker-retransfer"]
	secret_name = "awsaccesskeyid"
	secret_value = var.aws_keyid
}
resource "osc_secret" "secret" {
	service_ids = ["eyevinn-docker-retransfer"]
	secret_name = "awssecretaccesskey"
	secret_value = var.aws_secret
}


resource "osc_encore_transfer_instance" "example" {
	name = "ggexample"
	redis_url = osc_encore_callback_instance.example.redis_url
	redis_queue = osc_encore_callback_instance.example.redis_queue
	output = var.aws_output
	aws_keyid = osc_secret.keyid.secret_name 
	aws_secret = osc_secret.secret.secret_name 
	osc_token = var.osc_pat
}


output "encore_url" {
	value = trimsuffix(osc_encore_instance.example.url, "/")
}

output "encore_name" {
	value = osc_encore_instance.example.name
}

output "callback_url" {
	value = trimsuffix(osc_encore_callback_instance.example.url, "/")
}
