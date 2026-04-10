package testacc_org

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeMonitorTcpResource creates a basic TCP uptime monitor and verifies CRUD operations
func TestAccUptimeMonitorTcpResource(t *testing.T) {
	// Skip acceptance tests if TF_ACC environment variable is not set
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testacc.TestAccOrgPreCheck(t)

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: testacc.TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_tcp" "test" {
  project_scope = data.phare_project.test.slug

  name = "TCP Service"

  request {
    host            = "invariance.dev"
    port            = 443
    connection      = "tls"
    tls_skip_verify = false
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
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "name", "TCP Service"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "interval", "30"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "timeout", "7000"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "incident_confirmations", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "recovery_confirmations", "3"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "region_threshold", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.host", "invariance.dev"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.port", "443"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.connection", "tls"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.tls_skip_verify", "false"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_tcp.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_tcp.test", "project_id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_tcp.test", "status"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_tcp.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_tcp.test", "updated_at"),
				),
			},
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_tcp" "test" {
  project_scope = data.phare_project.test.slug

  name = "TCP Service Updated"

  request {
    host            = "invariance.dev"
    port            = 80
    connection      = "plain"
    tls_skip_verify = true
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
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "name", "TCP Service Updated"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "interval", "60"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "timeout", "10000"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "incident_confirmations", "2"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "recovery_confirmations", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "region_threshold", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.port", "80"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.connection", "plain"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "request.tls_skip_verify", "true"),
				),
			},
		},
	})
}
