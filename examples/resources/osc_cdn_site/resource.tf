# Serve the public-read buckets of a MinIO instance through the CDN.
resource "osc_cdn_site" "media" {
  service_id    = "minio-minio"
  instance_name = osc_instance.storage.name
}

# Put the CDN in front of a My App. It caches what the app's headers allow.
resource "osc_cdn_site" "web" {
  app_id = osc_my_app.web.id
}

output "media_cdn_url" {
  value = osc_cdn_site.media.cdn_url
}
