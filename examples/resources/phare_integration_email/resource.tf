data "phare_users" "on_call" {
  emails = ["alice@example.com", "bob@example.com"]
}

resource "phare_integration_email" "on_call" {
  name = "On-call team"

  settings = {
    members = data.phare_users.on_call.users[*].id
  }
}

resource "phare_alert_rule" "incident_created" {
  event          = "uptime.incident.created"
  scope          = "organization"
  integration_id = phare_integration_email.on_call.id
  rate_limit     = 5

  event_settings = jsonencode({
    type = "all"
  })
}
