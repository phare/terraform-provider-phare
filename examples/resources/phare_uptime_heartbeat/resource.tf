resource "phare_uptime_heartbeat" "backup" {
  name         = "Nightly backup"
  display_name = "Backups" # Public name on status pages

  interval               = 86400 # Expect a ping every day
  grace                  = 1800  # Count a ping as missed 30 minutes after it was expected
  incident_confirmations = 1
  recovery_confirmations = 1
  tags                   = ["environment:production", "team:backend"]
}

# The ping URL is sensitive, anyone knowing it can report the job as successful
output "backup_ping_url" {
  value     = phare_uptime_heartbeat.backup.ping_url
  sensitive = true
}
