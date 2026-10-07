package testacc_org

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeMonitorTcpResourceTags verifies the tags parameter on TCP uptime monitors
// and the tags filter on the phare_uptime_monitors data source with an organization-scoped API key.
func TestAccUptimeMonitorTcpResourceTags(t *testing.T) {
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
  project_scope = data.phare_project.test.slug
  tags          = ["tfacc:tcp-tags-test", "environment:production"]

  depends_on = [phare_uptime_monitor_tcp.test]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "2"),
					testingresource.TestCheckTypeSetElemAttr("phare_uptime_monitor_tcp.test", "tags.*", "tfacc:tcp-tags-test"),
					testingresource.TestCheckTypeSetElemAttr("phare_uptime_monitor_tcp.test", "tags.*", "environment:production"),
					testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_monitors.filtered", "monitors.*.id", "phare_uptime_monitor_tcp.test", "id"),
				),
			},
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_tcp" "test" {
  project_scope = data.phare_project.test.slug

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
					testingresource.TestCheckNoResourceAttr("phare_uptime_monitor_tcp.test", "tags.#"),
				),
			},
		},
	})
}
