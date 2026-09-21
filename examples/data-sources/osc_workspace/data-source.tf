# Which workspace and user the provider's token belongs to.
data "osc_workspace" "current" {}

output "workspace" {
  value = data.osc_workspace.current.workspace
}
