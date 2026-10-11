data "phare_users" "on_call" {
  emails = ["alice@example.com", "bob@example.com"]
}

# Requires a Scale plan subscription, members must verify their phone number
resource "phare_integration_sms" "on_call" {
  name = "On-call phones"

  settings = {
    members = data.phare_users.on_call.users[*].id
  }
}
