package testacc

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// alertRuleConfig returns a project-scoped alert rule on the default email integration
func alertRuleConfig(s Scope, rateLimit string) string {
	return EmailIntegrationDataSource + `
resource "phare_alert_rule" "test" {
  ` + s.Attr() + `
  event          = "uptime.incident.created"
  scope          = "project"
  integration_id = data.phare_integration.email.id
  rate_limit     = ` + rateLimit + `

  event_settings = jsonencode({
    type = "all"
  })
}
`
}

// RunAlertRuleResource creates a basic alert rule and verifies CRUD operations
func RunAlertRuleResource(t *testing.T, s Scope) {
	s.Test(t,
		testingresource.TestStep{
			Config: alertRuleConfig(s, "5"),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_alert_rule.test", "event", "uptime.incident.created"),
				testingresource.TestCheckResourceAttr("phare_alert_rule.test", "scope", "project"),
				testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "integration_id"),
				testingresource.TestCheckResourceAttr("phare_alert_rule.test", "rate_limit", "5"),
				testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "id"),
				testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "created_at"),
				testingresource.TestCheckResourceAttrSet("phare_alert_rule.test", "updated_at"),
			),
		},
		testingresource.TestStep{
			Config: alertRuleConfig(s, "10"),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_alert_rule.test", "rate_limit", "10"),
			),
		},
		// The scope is not returned by the API
		s.ImportStep("phare_alert_rule.test", "scope"),
	)
}

// RunAlertRuleDataSource verifies the phare_alert_rule and phare_alert_rules data sources
func RunAlertRuleDataSource(t *testing.T, s Scope) {
	s.Test(t,
		testingresource.TestStep{
			Config: alertRuleConfig(s, "30") + `
data "phare_alert_rule" "test" {
  ` + s.Attr() + `
  id = phare_alert_rule.test.id
}

data "phare_alert_rules" "all" {
  ` + s.Attr() + `
  depends_on = [phare_alert_rule.test]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttrPair("data.phare_alert_rule.test", "id", "phare_alert_rule.test", "id"),
				testingresource.TestCheckResourceAttrPair("data.phare_alert_rule.test", "integration_id", "phare_alert_rule.test", "integration_id"),
				testingresource.TestCheckResourceAttr("data.phare_alert_rule.test", "event", "uptime.incident.created"),
				testingresource.TestCheckResourceAttr("data.phare_alert_rule.test", "rate_limit", "30"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_alert_rules.all", "alert_rules.*.id", "phare_alert_rule.test", "id"),
			),
		},
	)
}
