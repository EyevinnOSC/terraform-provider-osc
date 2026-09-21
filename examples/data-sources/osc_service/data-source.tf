data "osc_service" "valkey" {
  service_id = "valkey-io-valkey"
}

# Parameter names, types and whether they are required, straight from the catalog.
output "valkey_parameters" {
  value = data.osc_service.valkey.parameters
}
