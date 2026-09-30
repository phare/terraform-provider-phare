resource "phare_uptime_monitor_icmp" "gateway" {
  name = "Gateway Ping"

  request {
    host = "1.1.1.1"
  }

  interval               = 30
  timeout                = 7000
  incident_confirmations = 1
  recovery_confirmations = 3
  region_threshold       = 1
  regions                = ["eu-fra-cdg"]
}
