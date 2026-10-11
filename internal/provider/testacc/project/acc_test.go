package testacc_project

import (
	"testing"

	"terraform-provider-phare/internal/provider/testacc"
)

// The shared acceptance tests are defined in the testacc package and run
// here with a project-scoped API key.

func TestAccAlertRuleResource(t *testing.T) {
	testacc.RunAlertRuleResource(t, testacc.ProjectScope)
}

func TestAccAlertRuleDataSource(t *testing.T) {
	testacc.RunAlertRuleDataSource(t, testacc.ProjectScope)
}

func TestAccUptimeMonitorHttpResource(t *testing.T) {
	testacc.RunUptimeMonitorHttpResource(t, testacc.ProjectScope)
}

func TestAccUptimeMonitorIcmpResource(t *testing.T) {
	testacc.RunUptimeMonitorIcmpResource(t, testacc.ProjectScope)
}

func TestAccUptimeMonitorTcpResource(t *testing.T) {
	testacc.RunUptimeMonitorTcpResource(t, testacc.ProjectScope)
}

func TestAccUptimeMonitorTcpResourceTags(t *testing.T) {
	testacc.RunUptimeMonitorTcpResourceTags(t, testacc.ProjectScope)
}

func TestAccUptimeMonitorDataSource(t *testing.T) {
	testacc.RunUptimeMonitorDataSource(t, testacc.ProjectScope)
}

func TestAccUptimeStatusPageResource(t *testing.T) {
	testacc.RunUptimeStatusPageResource(t, testacc.ProjectScope)
}

func TestAccUptimeStatusPageResourceWithFiles(t *testing.T) {
	testacc.RunUptimeStatusPageResourceWithFiles(t, testacc.ProjectScope)
}

func TestAccUptimeStatusPageResourceTags(t *testing.T) {
	testacc.RunUptimeStatusPageResourceTags(t, testacc.ProjectScope)
}

func TestAccUptimeStatusPageDataSource(t *testing.T) {
	testacc.RunUptimeStatusPageDataSource(t, testacc.ProjectScope)
}
