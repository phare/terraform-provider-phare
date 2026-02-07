data "phare_integration" "email" {
  app  = "email"
  name = "Default"
}

data "phare_integration" "email" {
  app  = "outgoing_webhook"
  name = "n8n"
}

resource "phare_alert_rule" "incident_created" {
  event          = "uptime.incident.created"
  scope          = "project"
  integration_id = data.phare_integration.email.id
  rate_limit     = 5

  event_settings = jsonencode({
    type = "all"
  })
}

resource "phare_alert_rule" "monitor_certificate_expiring" {
  event          = "uptime.monitor_certificate.expiring"
  scope          = "organization"
  integration_id = data.phare_integration.email.id
  rate_limit     = 5

  event_settings = jsonencode({
    days_before_expiry = 25
  })

  integration_settings = jsonencode({
    schema = jsonencode({
      "event" : "uptime.monitor_certificate.expiring",
      "certificate" : {
        "serial_number" : "{{ certificate.serial_number }}",
        "subject_common_name" : "{{ certificate.subject_common_name }}",
        "subject_alternative_names" : "{{ certificate.subject_alternative_names }}",
        "issuer_common_name" : "{{ certificate.issuer_common_name }}",
        "issuer_organization" : "{{ certificate.issuer_organization }}",
        "not_before" : "{{ certificate.not_before }}",
        "not_after" : "{{ certificate.not_after }}"
      },
      "monitor" : {
        "id" : "{{ monitor.id }}",
        "name" : "{{ monitor.name }}",
        "status" : "{{ monitor.status }}",
        "protocol" : "{{ monitor.protocol }}",
        "request" : "{{ monitor.request }}",
        "regions" : "{{ monitor.regions }}",
        "interval" : "{{ monitor.interval }}",
        "incident_confirmations" : "{{ monitor.incident_confirmations }}",
        "recovery_confirmations" : "{{ monitor.recovery_confirmations }}"
      },
      "project" : {
        "id" : "{{ project.id }}",
        "name" : "{{ project.name }}",
        "slug" : "{{ project.slug }}"
      }
    })
  })
}
