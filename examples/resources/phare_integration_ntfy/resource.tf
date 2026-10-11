resource "phare_integration_ntfy" "public" {
  name = "Ops notifications"

  settings = {
    hostname = "ntfy.sh"
    topic    = "phare-alerts-3f9c2a"
  }
}

variable "ntfy_access_token" {
  type      = string
  sensitive = true
}

resource "phare_integration_ntfy" "self_hosted" {
  name = "Self-hosted ntfy"

  settings = {
    hostname     = "ntfy.example.com"
    topic        = "phare-alerts"
    access_token = var.ntfy_access_token
  }
}
