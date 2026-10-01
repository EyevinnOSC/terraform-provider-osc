resource "osc_instance" "db" {
  service_id = "birme-osc-postgresql"
  name       = "mydb"

  sensitive_parameters = {
    PostgresPassword = var.db_password
  }
}

# Back up the database every night at 02:00 UTC and keep the backups for 30 days.
resource "osc_backup_schedule" "db" {
  service_id    = osc_instance.db.service_id
  instance_name = osc_instance.db.name
}

# Weekly backups on Sundays at 03:00 UTC, kept for a year.
resource "osc_backup_schedule" "archive" {
  service_id     = "go-gitea-gitea"
  instance_name  = "mygitea"
  schedule       = "0 3 * * 0"
  retention_days = 365
}

output "db_last_backup" {
  value = osc_backup_schedule.db.last_backup_at
}
