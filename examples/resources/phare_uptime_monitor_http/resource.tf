resource "phare_uptime_monitor_http" "website" {
  name         = "Website"
  display_name = "Marketing Website" # Public name on status pages

  request {
    method            = "HEAD"
    url               = "https://docs.phare.io/introduction"
    tls_skip_verify   = false
    follow_redirects  = true
    user_agent_secret = "definitely-not-a-bot" # Never returned by the API

    headers {
      name  = "X-Phare-Says"
      value = "Hello world!"
    }

    headers {
      name   = "Authorization"
      value  = "Bearer my-api-token"
      secret = true # Encrypted by Phare, its value is never returned by the API
    }
  }

  interval               = 60
  timeout                = 10000
  incident_confirmations = 3
  recovery_confirmations = 3
  region_threshold       = 1
  regions                = ["as-jpn-hnd"]

  success_assertions {
    status_code {
      operator = "in"
      value    = "2xx"
    }

    response_header {
      selector = "Content-Type"
      operator = "equals"
      value    = "application/json"
    }

    response_header {
      selector = "Cache-Control"
      operator = "not_equals"
      value    = "no-cache"
    }

    response_body {
      operator = "contains"
      value    = "Hello"
    }

    response_body {
      operator = "not_contains"
      value    = "Error"
    }
  }

  tags = ["environment:production", "team:backend"]
}
