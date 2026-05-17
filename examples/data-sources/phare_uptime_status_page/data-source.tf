data "phare_uptime_status_page" "public" {
  id = 123
}

output "is_password_protected" {
  value = data.phare_uptime_status_page.public.access_password_enabled
}

output "allowed_ips" {
  value = data.phare_uptime_status_page.public.access_ips
}
