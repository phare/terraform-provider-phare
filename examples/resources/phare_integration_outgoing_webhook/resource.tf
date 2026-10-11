variable "webhook_secret" {
  type      = string
  sensitive = true
}

resource "phare_integration_outgoing_webhook" "automation" {
  name = "Automation webhook"

  settings = {
    url    = "https://example.com/webhooks/phare"
    secret = var.webhook_secret
  }
}
