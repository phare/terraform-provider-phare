data "phare_uptime_monitors" "all" {}

data "phare_uptime_monitors" "production" {
  tags = ["environment:production"]
}
