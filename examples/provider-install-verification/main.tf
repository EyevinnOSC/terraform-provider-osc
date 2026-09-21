terraform {
  required_providers {
    osc = {
      source = "EyevinnOSC/osc"
    }
  }
}

variable "osc_pat" {
  type      = string
  sensitive = true
}

variable "osc_environment" {
  type    = string
  default = "prod"
}

variable "osc_workspace" {
  type        = string
  description = "OSC workspace (tenant) id the token belongs to"
}

variable "aws_keyid" {
  type      = string
  sensitive = true
}

variable "aws_secret" {
  type      = string
  sensitive = true
}

variable "aws_output" {
  type = string
}

provider "osc" {
  pat         = var.osc_pat
  environment = var.osc_environment
  workspace   = var.osc_workspace
}

resource "osc_instance" "encore" {
  service_id = "encore"
  name       = "ggexample"
  parameters = {
    profilesUrl = "https://raw.githubusercontent.com/Eyevinn/encore-test-profiles/refs/heads/main/profiles.yml"
  }
}

resource "osc_instance" "valkey" {
  service_id = "valkey-io-valkey"
  name       = "ggexample"
}

resource "osc_instance" "callback" {
  service_id = "eyevinn-encore-callback-listener"
  name       = "ggexample"
  parameters = {
    RedisUrl   = format("redis://%s:%s", osc_instance.valkey.external_ip, osc_instance.valkey.external_port)
    EncoreUrl  = trimsuffix(osc_instance.encore.url, "/")
    RedisQueue = "transfer"
  }
}

resource "osc_secret" "keyid" {
  service_ids  = ["eyevinn-encore-transfer"]
  secret_name  = "awsaccesskeyid"
  secret_value = var.aws_keyid
}

resource "osc_secret" "secret" {
  service_ids  = ["eyevinn-encore-transfer"]
  secret_name  = "awssecretaccesskey"
  secret_value = var.aws_secret
}

resource "osc_instance" "transfer" {
  service_id = "eyevinn-encore-transfer"
  name       = "ggexample"
  parameters = {
    RedisUrl                 = osc_instance.callback.parameters["RedisUrl"]
    RedisQueue               = osc_instance.callback.parameters["RedisQueue"]
    Output                   = var.aws_output
    AwsAccessKeyIdSecret     = osc_secret.keyid.ref
    AwsSecretAccessKeySecret = osc_secret.secret.ref
  }
  sensitive_parameters = {
    OscAccessToken = var.osc_pat
  }
}

output "encore_url" {
  value = trimsuffix(osc_instance.encore.url, "/")
}

output "encore_name" {
  value = osc_instance.encore.name
}

output "callback_url" {
  value = trimsuffix(osc_instance.callback.url, "/")
}
