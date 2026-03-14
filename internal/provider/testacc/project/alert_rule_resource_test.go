package testacc_project

import (
	"os"
	"testing"

	"terraform-provider-phare/internal/provider/testacc"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAlertRuleResource creates a basic alert rule and verifies CRUD operations with project-scoped API key
func TestAccAlertRuleResource(t *testing.T) {
	// Skip acceptance tests if TF_ACC environment variable is not set
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testacc.TestAccProjectPreCheck(t)

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: testacc.TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			{
				Config: `
data "phare_integration" "email" {
  app  = "email"
  name = "Default"
}

resource "phare_alert_rule" "test" {
  event          = "uptime.incident.created"
  scope          = "organization"
  integration_id = data.phare_integration.email.id
  rate_limit     = 5

  event_settings = jsonencode({
    type = "all"
  })
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_alert_rule.test", "event", "uptime.incident.created"),
					testingresource.TestCheckResourceAttr("phare_alert_rule.test", "scope", "organization"),
					testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "integration_id"),
					testingresource.TestCheckResourceAttr("phare_alert_rule.test", "rate_limit", "5"),
					testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "updated_at"),
				),
			},
			{
				Config: `
data "phare_integration" "email" {
  app  = "email"
  name = "Default"
}

resource "phare_alert_rule" "test" {
  event          = "uptime.incident.created"
  scope          = "organization"
  integration_id = data.phare_integration.email.id
  rate_limit     = 10

  event_settings = jsonencode({
    type = "all"
  })
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_alert_rule.test", "rate_limit", "10"),
				),
			},
		},
	})
}
