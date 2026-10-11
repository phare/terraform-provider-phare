data "phare_integration" "email" {
  app  = "email"
  name = "Default"
}

data "phare_integration" "slack" {
  app  = "slack"
  name = "DevOps Team"
}
