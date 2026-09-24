# A static site at https://example-docs.pages.osaas.io/. Publish its files from CI.
resource "osc_my_page" "docs" {
  name = "example-docs"
}

# A preview site behind basic auth, also served at a custom domain.
resource "osc_my_page" "preview" {
  name                = "example-preview"
  custom_domain       = "preview.example.com"
  basic_auth_username = "review"
  basic_auth_password = var.preview_password
}
