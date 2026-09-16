# Import by identity (Terraform 1.12+). `terraform query -generate-config-out` writes
# these blocks for every instance in the workspace; see the osc_instance list resource.
import {
  to = osc_instance.cache
  identity = {
    service_id = "valkey-io-valkey"
    name       = "mycache"
  }
}
