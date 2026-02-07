resource "phare_uptime_monitor_tcp" "service" {
  name = "TCP Service"

  request = {
    host            = "phare.io"
    port            = 443
    connection      = "tls"
    tls_skip_verify = false
  }

  interval               = 30
  timeout                = 7000
  incident_confirmations = 1
  recovery_confirmations = 3
  regions                = ["eu-fra-cdg"]
}
