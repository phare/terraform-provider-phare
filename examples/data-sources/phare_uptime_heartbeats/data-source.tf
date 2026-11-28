data "phare_uptime_heartbeats" "all" {}

data "phare_uptime_heartbeats" "production" {
  tags = ["environment:production"]
}
