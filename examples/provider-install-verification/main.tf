terraform {
  required_providers {
    osc = {
      source  = "EyevinnOSC/osc"
      version = ">= 1.6.1"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.0"
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

# Letters and digits only, since the password is embedded in a URL.
resource "random_password" "valkey" {
  length  = 32
  special = false
}

# The password is set explicitly so that the Valkey is protected and the URLs
# below can carry it. Newer provider versions generate a password when none is
# given, and a URL without it would then be refused with NOAUTH.
resource "osc_instance" "valkey" {
  service_id = "valkey-io-valkey"
  name       = "ggexample"
  sensitive_parameters = {
    Password = random_password.valkey.result
  }
}

locals {
  redis_url = format(
    "redis://:%s@%s:%s",
    urlencode(random_password.valkey.result),
    osc_instance.valkey.external_ip,
    osc_instance.valkey.external_port,
  )
}

resource "osc_instance" "callback" {
  service_id = "eyevinn-encore-callback-listener"
  name       = "ggexample"
  parameters = {
    EncoreUrl  = trimsuffix(osc_instance.encore.url, "/")
    RedisQueue = "transfer"
  }
  sensitive_parameters = {
    RedisUrl = local.redis_url
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
    RedisQueue               = osc_instance.callback.parameters["RedisQueue"]
    Output                   = var.aws_output
    AwsAccessKeyIdSecret     = osc_secret.keyid.ref
    AwsSecretAccessKeySecret = osc_secret.secret.ref
  }
  sensitive_parameters = {
    OscAccessToken = var.osc_pat
    RedisUrl       = local.redis_url
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
