data "phare_uptime_status_pages" "all" {}

data "phare_uptime_status_pages" "production" {
  tags = ["environment:production"]
}
