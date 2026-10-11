variable "pushover_user_key" {
  type      = string
  sensitive = true
}

variable "pushover_api_key" {
  type      = string
  sensitive = true
}

resource "phare_integration_pushover" "phones" {
  name = "Pushover"

  settings = {
    user_key = var.pushover_user_key
    api_key  = var.pushover_api_key
  }
}
