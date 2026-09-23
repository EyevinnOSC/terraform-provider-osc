# Serve a My App at a custom domain. Create a CNAME from the domain to the
# app's managed_domain first, so OSC can issue the certificate.
resource "osc_domain" "api" {
  domain        = "api.example.com"
  service_id    = osc_my_app.api.domain_service_id
  instance_name = osc_my_app.api.id
}

# Serve one bucket of a MinIO instance at a custom domain.
resource "osc_domain" "media" {
  domain        = "media.example.com"
  service_id    = "minio-minio"
  instance_name = osc_instance.storage.name
  origin_path   = "/media"
}
