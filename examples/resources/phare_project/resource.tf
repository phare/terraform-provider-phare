data "phare_users" "team" {
  emails = ["nicolas@phare.io"]
}

resource "phare_project" "test" {
  name    = "Terraform"
  members = data.phare_users.team.users[*].id

  settings {
    use_incident_ai              = true
    use_incident_merging         = true
    incident_merging_time_window = 30
  }
}
