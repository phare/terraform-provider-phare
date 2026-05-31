data "phare_uptime_monitor" "web" {
  id = 123
}

resource "phare_uptime_status_page" "main" {
  name                  = "Status page"
  title                 = "Example status page"
  description           = "This is an example status page description created from Terraform."
  website_url           = "https://example.com"
  search_engine_indexed = false
  subdomain             = "example" # example.status.phare.io
  domain                = "status.example.com"
  timeframe             = 30
  logo_light            = "${path.module}/assets/logo-light.png"
  logo_dark             = "${path.module}/assets/logo-dark.png"
  favicon_light         = "${path.module}/assets/favicon.png"
  favicon_dark          = "${path.module}/assets/favicon.png"
  color_scheme          = "all"

  theme {
    rounded      = true
    border_width = 1

    light {
      empty                = "#e5e5e5"
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      background           = "#ffffff"
      border               = "#e5e5e5"
      foreground           = "#000000"
      foreground_muted     = "#737373"
      background_card      = "#ffffff"
    }

    dark {
      empty                = "#3f3f46"
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      background           = "#18181b"
      border               = "#3f3f46"
      foreground           = "#f5f5f5"
      foreground_muted     = "#a3a3a3"
      background_card      = "#18181b"
    }
  }

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor.web.id
    }
  ]

  # Requires an active Scale plan subscription
  access_password = "supersecret"
  access_token    = "mytoken"
  access_ips      = ["192.168.1.0/24", "10.0.0.1"]
}
