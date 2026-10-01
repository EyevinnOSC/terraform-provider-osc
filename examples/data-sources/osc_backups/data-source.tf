# The backups of a database, newest first.
data "osc_backups" "db" {
  service_id    = "birme-osc-postgresql"
  instance_name = "mydb"
}

# The newest completed backup, to name in a restore.
output "latest_backup" {
  value = try([for b in data.osc_backups.db.backups : b.name if b.status == "Complete"][0], null)
}
