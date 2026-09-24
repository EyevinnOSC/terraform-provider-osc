# The workspace mailbox, and an app sending through it. The mailbox can send
# only as its own address, so the app sends as `email` and sets Reply-To to an
# inbox that is read.
resource "osc_mailbox" "main" {
  password = var.mailbox_password
}

resource "osc_parameter" "smtp_host" {
  parameter_store = osc_parameter_store.api.name
  key             = "SMTP_HOST"
  value           = osc_mailbox.main.smtp_host
}

resource "osc_parameter" "smtp_port" {
  parameter_store = osc_parameter_store.api.name
  key             = "SMTP_PORT"
  value           = tostring(osc_mailbox.main.smtp_port)
}

resource "osc_parameter" "smtp_user" {
  parameter_store = osc_parameter_store.api.name
  key             = "SMTP_USER"
  value           = osc_mailbox.main.email
}

resource "osc_parameter" "smtp_pass" {
  parameter_store = osc_parameter_store.api.name
  key             = "SMTP_PASS"
  secret_value    = var.mailbox_password
}
