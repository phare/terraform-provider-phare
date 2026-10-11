package testacc_org

import (
	"testing"

	"terraform-provider-phare/internal/provider/testacc"
)

// The shared acceptance tests are defined in the testacc package and run
// here with an organization-scoped API key.

// scope returns the organization scope, creating resources in the project of
// the test run. It must be called after TestMain created the project.
func scope() testacc.Scope {
	return testacc.OrgScope(runProjectSlug)
}

func TestAccAlertRuleResource(t *testing.T) {
	testacc.RunAlertRuleResource(t, scope())
}

func TestAccAlertRuleDataSource(t *testing.T) {
	testacc.RunAlertRuleDataSource(t, scope())
}

func TestAccIntegrationEmailResource(t *testing.T) {
	testacc.RunIntegrationEmailResource(t, scope())
}

func TestAccIntegrationIlertResource(t *testing.T) {
	testacc.RunIntegrationIlertResource(t, scope())
}

func TestAccIntegrationNtfyResource(t *testing.T) {
	testacc.RunIntegrationNtfyResource(t, scope())
}

func TestAccIntegrationOutgoingWebhookResource(t *testing.T) {
	testacc.RunIntegrationOutgoingWebhookResource(t, scope())
}

func TestAccIntegrationPushoverResource(t *testing.T) {
	testacc.RunIntegrationPushoverResource(t, scope())
}

func TestAccUptimeMonitorHttpResource(t *testing.T) {
	testacc.RunUptimeMonitorHttpResource(t, scope())
}

func TestAccUptimeMonitorIcmpResource(t *testing.T) {
	testacc.RunUptimeMonitorIcmpResource(t, scope())
}

func TestAccUptimeMonitorTcpResource(t *testing.T) {
	testacc.RunUptimeMonitorTcpResource(t, scope())
}

func TestAccUptimeMonitorTcpResourceTags(t *testing.T) {
	testacc.RunUptimeMonitorTcpResourceTags(t, scope())
}

func TestAccUptimeMonitorDataSource(t *testing.T) {
	testacc.RunUptimeMonitorDataSource(t, scope())
}

func TestAccUptimeStatusPageResource(t *testing.T) {
	testacc.RunUptimeStatusPageResource(t, scope())
}

func TestAccUptimeStatusPageResourceWithFiles(t *testing.T) {
	testacc.RunUptimeStatusPageResourceWithFiles(t, scope())
}

func TestAccUptimeStatusPageResourceTags(t *testing.T) {
	testacc.RunUptimeStatusPageResourceTags(t, scope())
}

func TestAccUptimeStatusPageDataSource(t *testing.T) {
	testacc.RunUptimeStatusPageDataSource(t, scope())
}
