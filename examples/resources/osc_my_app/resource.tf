# A Node.js API built from a private GitHub repository, with its environment from a
# parameter store. Changing rebuild_trigger (here, from CI) deploys the head of main.
resource "osc_my_app" "api" {
  name           = "myapi"
  type           = "nodejs"
  git_url        = "https://github.com/example/my-api"
  source_ref     = "main"
  git_credential = "github-example"
  config_service = osc_parameter_store.api.name

  high_availability = true
  rebuild_trigger   = var.release_sha

  # The app reads its environment at start, so create the values first.
  depends_on = [osc_parameter.node_env, osc_parameter.database_url]
}

output "api_url" {
  value = osc_my_app.api.url
}
