package testacc_project

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeMonitorTcpResourceTags verifies the tags parameter on TCP uptime monitors
// and the tags filter on the phare_uptime_monitors data source with a project-scoped API key.
func TestAccUptimeMonitorTcpResourceTags(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testacc.TestAccProjectPreCheck(t)

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: testacc.TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			{
				Config: `
resource "phare_uptime_monitor_tcp" "test" {
  name = "TCP Tags Service"

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
  tags                   = ["tfacc:tcp-tags-test"]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "1"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.0", "tfacc:tcp-tags-test"),
				),
			},
			{
				Config: `
resource "phare_uptime_monitor_tcp" "test" {
  name = "TCP Tags Service"

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
  tags                   = ["tfacc:tcp-tags-test", "environment:production"]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "2"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.0", "tfacc:tcp-tags-test"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.1", "environment:production"),
				),
			},
			{
				Config: `
resource "phare_uptime_monitor_tcp" "test" {
  name = "TCP Tags Service"

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
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "0"),
				),
			},
			{
				Config: `
resource "phare_uptime_monitor_tcp" "test" {
  name = "TCP Tags Service"

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
  tags                   = ["tfacc:tcp-tags-test", "environment:production"]
}

data "phare_uptime_monitors" "filtered" {
  tags = ["tfacc:tcp-tags-test", "environment:production"]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("data.phare_uptime_monitors.filtered", "monitors.#", "1"),
					testingresource.TestCheckResourceAttrPair("data.phare_uptime_monitors.filtered", "monitors.0.id", "phare_uptime_monitor_tcp.test", "id"),
				),
			},
		},
	})
}
