# Import by id, then `terraform plan -generate-config-out=generated.tf` writes the
# resource block, parameters included.
import {
  to = osc_instance.cache
  id = "valkey-io-valkey/mycache"
}
