# Each value becomes an environment variable of every app bound to the store.
resource "osc_parameter" "node_env" {
  parameter_store = osc_parameter_store.api.name
  key             = "NODE_ENV"
  value           = "production"
}

# Secret values are encrypted at rest and hidden from plan output.
resource "osc_parameter" "database_url" {
  parameter_store = osc_parameter_store.api.name
  key             = "DATABASE_URL"
  secret_value    = "postgres://api:${var.db_password}@${var.db_host}:5432/api"
}
