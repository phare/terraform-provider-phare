package testacc_project

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeMonitorIcmpResource creates a basic ICMP uptime monitor and verifies CRUD operations with project-scoped API key
func TestAccUptimeMonitorIcmpResource(t *testing.T) {
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
resource "phare_uptime_monitor_icmp" "test" {
  name = "ICMP Host"

  request {
    host = "1.1.1.1"
  }

  interval               = 30
  timeout                = 7000
  incident_confirmations = 1
  recovery_confirmations = 3
  region_threshold       = 1
  regions                = ["eu-fra-cdg"]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "name", "ICMP Host"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "interval", "30"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "timeout", "7000"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "incident_confirmations", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "recovery_confirmations", "3"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "region_threshold", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "request.host", "1.1.1.1"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_icmp.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_icmp.test", "project_id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_icmp.test", "status"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_icmp.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_icmp.test", "updated_at"),
				),
			},
			{
				Config: `
resource "phare_uptime_monitor_icmp" "test" {
  name = "ICMP Host Updated"

  request {
    host = "8.8.8.8"
  }

  interval               = 60
  timeout                = 10000
  incident_confirmations = 2
  recovery_confirmations = 1
  region_threshold       = 1
  regions                = ["eu-fra-cdg"]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "name", "ICMP Host Updated"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "interval", "60"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "timeout", "10000"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "incident_confirmations", "2"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "recovery_confirmations", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "region_threshold", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "request.host", "8.8.8.8"),
				),
			},
		},
	})
}
