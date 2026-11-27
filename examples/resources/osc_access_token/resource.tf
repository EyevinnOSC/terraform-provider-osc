# A token for CI that can only read: made from a template, valid for 30 days.
resource "osc_access_token" "ci_viewer" {
  name            = "ci-viewer"
  description     = "Read-only token for the CI pipeline"
  template        = "viewer"
  expires_in_days = 30
}

# A token with an explicit policy. Changing the policy later keeps the same token
# value; it is replaced in place.
resource "osc_access_token" "deployer" {
  name = "release-deployer"
  policy = jsonencode({
    version = 1
    statements = [
      {
        actions   = ["myapps:read", "myapps:deploy"]
        resources = ["myapps/*"]
      }
    ]
  })
}

# The token value is stored in the Terraform state, so protect the state. To rotate
# the token, run: terraform apply -replace=osc_access_token.deployer
output "deployer_token" {
  value     = osc_access_token.deployer.token
  sensitive = true
}
