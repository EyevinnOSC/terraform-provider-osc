# A Valkey (Redis compatible) key-value store. Parameter names come from the
# service's schema in the OSC catalog; see the osc_service data source.
resource "osc_instance" "cache" {
  service_id = "valkey-io-valkey"
  name       = "mycache"

  sensitive_parameters = {
    Password = var.valkey_password
  }
}

# An Encore callback listener wired to the Valkey instance above. Outputs of one
# instance feed the parameters of the next, and Terraform orders the creation.
resource "osc_instance" "callback" {
  service_id = "eyevinn-encore-callback-listener"
  name       = "mycallback"

  parameters = {
    RedisUrl   = "redis://${osc_instance.cache.external_ip}:${osc_instance.cache.external_port}"
    EncoreUrl  = trimsuffix(osc_instance.encore.url, "/")
    RedisQueue = "transfer"
  }
}

resource "osc_instance" "encore" {
  service_id = "encore"
  name       = "myencore"

  parameters = {
    profilesUrl = "https://raw.githubusercontent.com/Eyevinn/encore-test-profiles/refs/heads/main/profiles.yml"
  }
}

output "cache_url" {
  value = osc_instance.cache.url
}
