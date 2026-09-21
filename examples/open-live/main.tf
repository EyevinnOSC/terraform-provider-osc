# Open Live on OSC: CouchDB for persistence, the Open Live API server, and the
# Open Live Studio frontend, with credentials kept in OSC secrets.
#
#   export TF_VAR_osc_workspace=<workspace>
#   export TF_VAR_strom_api_key=<api key for the managed Strom>
#   export TF_VAR_studio_osc_pat=<OSC personal access token for this workspace>
#   terraform apply
#
# The CouchDB admin password is generated and only ever handed to OSC as a
# secret. It still lives in the Terraform state, so protect the state.

terraform {
  required_providers {
    osc = {
      source = "EyevinnOSC/osc"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

variable "osc_workspace" {
  type        = string
  description = "OSC workspace (tenant) the token belongs to and where everything is created."
}

variable "name" {
  type        = string
  default     = "openlive"
  description = "Base instance name. Lowercase letters and digits only, at most 14 characters."
  validation {
    condition     = can(regex("^[a-z0-9]{1,14}$", var.name))
    error_message = "Use lowercase letters and digits only, at most 14 characters."
  }
}

variable "open_live_use_latest" {
  type        = bool
  default     = false
  description = "Run Open Live and Open Live Studio from their latest built images instead of the stable releases."
}

variable "strom_url" {
  type        = string
  default     = "https://strom-dev-de-fra.osaas.io:8080"
  description = "Base URL of the managed Strom pipeline engine."
}

variable "strom_api_key" {
  type        = string
  sensitive   = true
  description = "API key for the managed Strom instance."
}

variable "studio_osc_pat" {
  type        = string
  sensitive   = true
  description = <<-EOT
    OSC personal access token for this workspace, given to Open Live Studio. The
    browser exchanges it for a service access token so it can reach the Open Live
    backend behind OSC's ingress auth. Use a dedicated token so it can be rotated
    independently of the token Terraform runs with.
  EOT
}

provider "osc" {
  environment = "prod"
  workspace   = var.osc_workspace
}

# Subscribing here lets us predict the studio URL before the studio exists.
# Open Live needs it as CORS origin at creation time and does not support updates.
data "osc_service" "open_live_studio" {
  service_id = "eyevinn-open-live-studio"
  subscribe  = true
}

locals {
  couchdb_name = "${var.name}db"
  studio_name  = "${var.name}studio"

  # Instance URLs are https://{workspace}-{name}.{service host}, where the
  # service host is the service API host without its "api-" prefix.
  studio_host = replace(regex("^https://([^/]+)/", data.osc_service.open_live_studio.api_url)[0], "/^api-/", "")
  studio_url  = "https://${var.osc_workspace}-${local.studio_name}.${local.studio_host}"
}

# Letters and digits only, since the password is embedded in a URL.
resource "random_password" "couchdb_admin" {
  length  = 32
  special = false
}

resource "osc_secret" "couchdb_admin_password" {
  service_ids  = ["apache-couchdb"]
  secret_name  = "${local.couchdb_name}adminpw"
  secret_value = random_password.couchdb_admin.result
}

resource "osc_instance" "couchdb" {
  service_id = "apache-couchdb"
  name       = local.couchdb_name

  parameters = {
    AdminPassword = osc_secret.couchdb_admin_password.ref
  }

  lifecycle {
    replace_triggered_by = [osc_secret.couchdb_admin_password]
  }
}

# Open Live expects its database to exist and CouchDB does not create it, so
# create it once the instance answers. 201 = created, 412 = already there.
resource "terraform_data" "open_live_database" {
  triggers_replace = [osc_instance.couchdb.id]

  lifecycle {
    replace_triggered_by = [osc_instance.couchdb]
  }

  provisioner "local-exec" {
    environment = {
      COUCHDB_URL      = osc_instance.couchdb.url
      COUCHDB_PASSWORD = random_password.couchdb_admin.result
    }
    command = <<-EOT
      for i in $(seq 1 30); do
        code=$(curl -s -o /dev/null -w '%%{http_code}' -X PUT -u "admin:$COUCHDB_PASSWORD" "$COUCHDB_URL/open-live")
        case "$code" in
          201|412) exit 0 ;;
        esac
        sleep 5
      done
      echo "creating the open-live database failed, last HTTP status $code" >&2
      exit 1
    EOT
  }
}

resource "osc_secret" "open_live_db_url" {
  service_ids  = ["eyevinn-open-live"]
  secret_name  = "${var.name}dburl"
  secret_value = "https://admin:${random_password.couchdb_admin.result}@${trimprefix(osc_instance.couchdb.url, "https://")}/open-live"
}

resource "osc_secret" "strom_api_key" {
  service_ids  = ["eyevinn-open-live"]
  secret_name  = "${var.name}stromkey"
  secret_value = var.strom_api_key
}

resource "osc_instance" "open_live" {
  service_id = "eyevinn-open-live"
  name       = var.name
  use_latest = var.open_live_use_latest

  parameters = {
    DatabaseUrl      = osc_secret.open_live_db_url.ref
    StromUrl         = var.strom_url
    StromAuthMode    = "direct"
    StromAccessToken = osc_secret.strom_api_key.ref
    CorsOrigin       = local.studio_url
  }

  depends_on = [terraform_data.open_live_database]

  # Secrets are read when the instance starts, so a new secret needs a new instance.
  lifecycle {
    replace_triggered_by = [osc_secret.open_live_db_url, osc_secret.strom_api_key]
  }
}

resource "osc_secret" "studio_osc_pat" {
  service_ids  = ["eyevinn-open-live-studio"]
  secret_name  = "${local.studio_name}oscpat"
  secret_value = var.studio_osc_pat
}

resource "osc_instance" "open_live_studio" {
  service_id = "eyevinn-open-live-studio"
  name       = local.studio_name
  use_latest = var.open_live_use_latest

  parameters = {
    OpenLiveUrl    = osc_instance.open_live.url
    OscAccessToken = osc_secret.studio_osc_pat.ref
  }

  lifecycle {
    replace_triggered_by = [osc_secret.studio_osc_pat]

    postcondition {
      condition     = self.url == local.studio_url
      error_message = "Studio URL ${self.url} differs from the CORS origin ${local.studio_url} configured on Open Live."
    }
  }
}

output "couchdb_url" {
  value = osc_instance.couchdb.url
}

output "open_live_url" {
  value = osc_instance.open_live.url
}

output "open_live_studio_url" {
  value = osc_instance.open_live_studio.url
}
