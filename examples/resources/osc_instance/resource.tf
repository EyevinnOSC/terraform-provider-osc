# A Valkey (Redis compatible) key-value store. Parameter names come from the
# service's schema in the OSC catalog; see the osc_service data source.
resource "osc_instance" "cache" {
  service_id = "valkey-io-valkey"
  name       = "mycache"

  # Reachable only inside the OSC cluster.
  public_access = false

  # A project in OSC is a tag; tag every resource in the project with it.
  tags = ["my-project"]

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
    EncoreUrl  = trimsuffix(osc_instance.encore.url, "/")
    RedisQueue = "transfer"
  }

  # The Valkey requires a password, so the URL carries it. The URL holds a
  # credential and therefore goes in sensitive_parameters.
  sensitive_parameters = {
    RedisUrl = "redis://:${urlencode(var.valkey_password)}@${osc_instance.cache.internal_host}:6379"
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
