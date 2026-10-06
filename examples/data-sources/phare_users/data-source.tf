data "phare_users" "all" {}

data "phare_users" "specific" {
  emails = ["admin@example.com", "ops@example.com"]
}
