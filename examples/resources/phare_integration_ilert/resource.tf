variable "ilert_integration_key" {
  type      = string
  sensitive = true
}

resource "phare_integration_ilert" "on_call" {
  name = "ilert on-call"

  settings = {
    integration_key = var.ilert_integration_key
  }
}
