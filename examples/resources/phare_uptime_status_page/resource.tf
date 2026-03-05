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
  color_scheme          = "all"

  theme {
    rounded      = true
    border_width = 2

    light {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#ffffff"
      foreground           = "#000000"
      foreground_muted     = "#737373"
      background_card      = "#fafafa"
    }

    dark {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#111111"
      foreground           = "#ffffff"
      foreground_muted     = "#959595"
      background_card      = "#1a1a1a"
    }
  }

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor.web.id
    }
  ]
}
