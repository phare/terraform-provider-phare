resource "phare_uptime_monitor_http" "website" {
  name = "Website"

  request = {
    method            = "HEAD"
    url               = "https://docs.phare.io/introduction"
    tls_skip_verify   = false
    follow_redirects  = true
    user_agent_secret = "definitely-not-a-bot"
    headers = [
      {
        name  = "X-Phare-Says"
        value = "Hello world!"
      }
    ]
  }

  interval               = 60
  timeout                = 10000
  incident_confirmations = 3
  recovery_confirmations = 3
  regions                = ["as-jpn-hnd"]

  success_assertions = [
    {
      type     = "status_code"
      operator = "in"
      value    = "2xx,30x,418"
    }
  ]
}
