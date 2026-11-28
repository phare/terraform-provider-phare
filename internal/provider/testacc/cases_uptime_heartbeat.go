package testacc

import (
	"fmt"
	"strings"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// RunUptimeHeartbeatResource creates an uptime heartbeat and verifies CRUD operations
func RunUptimeHeartbeatResource(t *testing.T, s Scope) {
	name := RandomName("heartbeat")
	var pingURL string

	s.Test(t,
		testingresource.TestStep{
			Config: `
resource "phare_uptime_heartbeat" "test" {
	` + s.Attr() + `
	name         = "` + name + `"
	display_name = "Backups"
	interval     = 86400
	grace        = 300

	incident_confirmations = 1
	recovery_confirmations = 2

	tags = ["environment:production"]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "display_name", "Backups"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "interval", "86400"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "grace", "300"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "incident_confirmations", "1"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "recovery_confirmations", "2"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "tags.#", "1"),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_heartbeat.test", "tags.*", "environment:production"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "status", "waiting"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "paused", "false"),
				testingresource.TestCheckResourceAttrWith("phare_uptime_heartbeat.test", "ping_url", func(value string) error {
					if !strings.HasPrefix(value, "https://") {
						return fmt.Errorf("expected an HTTPS ping URL, got %q", value)
					}
					pingURL = value
					return nil
				}),
				testingresource.TestCheckResourceAttrSet("phare_uptime_heartbeat.test", "id"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_heartbeat.test", "project_id"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_heartbeat.test", "created_at"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_heartbeat.test", "updated_at"),
			),
		},
		testingresource.TestStep{
			Config: `
resource "phare_uptime_heartbeat" "test" {
	` + s.Attr() + `
	name     = "` + name + `/updated"
	interval = 3600
	grace    = 600

	incident_confirmations = 2
	recovery_confirmations = 1
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "name", name+"/updated"),
				testingresource.TestCheckNoResourceAttr("phare_uptime_heartbeat.test", "display_name"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "interval", "3600"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "grace", "600"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "incident_confirmations", "2"),
				testingresource.TestCheckResourceAttr("phare_uptime_heartbeat.test", "recovery_confirmations", "1"),
				testingresource.TestCheckNoResourceAttr("phare_uptime_heartbeat.test", "tags.#"),
				// The ping URL stays the same when the heartbeat is updated
				testingresource.TestCheckResourceAttrWith("phare_uptime_heartbeat.test", "ping_url", func(value string) error {
					if value != pingURL {
						return fmt.Errorf("expected the ping URL to stay %q, got %q", pingURL, value)
					}
					return nil
				}),
			),
		},
		s.ImportStep("phare_uptime_heartbeat.test", "updated_at"),
	)
}

// RunUptimeHeartbeatDataSource verifies the phare_uptime_heartbeat and
// phare_uptime_heartbeats data sources, and a heartbeat shown on a status page
func RunUptimeHeartbeatDataSource(t *testing.T, s Scope) {
	name := RandomName("heartbeat")
	pageName := RandomName("page")
	subdomain := RandomSubdomain()

	s.Test(t,
		testingresource.TestStep{
			Config: `
resource "phare_uptime_heartbeat" "test" {
	` + s.Attr() + `
	name     = "` + name + `"
	interval = 600
	grace    = 60

	incident_confirmations = 1
	recovery_confirmations = 1

	tags = ["team:` + strings.TrimPrefix(name, ResourcePrefix) + `"]
}

data "phare_uptime_heartbeat" "test" {
	` + s.Attr() + `
	id = phare_uptime_heartbeat.test.id
}

data "phare_uptime_heartbeats" "filtered" {
	` + s.Attr() + `
	tags       = ["team:` + strings.TrimPrefix(name, ResourcePrefix) + `"]
	depends_on = [phare_uptime_heartbeat.test]
}

resource "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  name                  = "` + pageName + `"
  title                 = "Example status page"
  description           = "This is an example status page showing a heartbeat."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "` + subdomain + `"
  timeframe             = 30
  color_scheme          = "all"
` + StatusPageThemeBlock + `
  components = [
    {
      componentable_type = "uptime/heartbeat"
      componentable_id   = phare_uptime_heartbeat.test.id
    }
  ]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_heartbeat.test", "id", "phare_uptime_heartbeat.test", "id"),
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_heartbeat.test", "project_id", "phare_uptime_heartbeat.test", "project_id"),
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_heartbeat.test", "ping_url", "phare_uptime_heartbeat.test", "ping_url"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_heartbeat.test", "name", name),
				testingresource.TestCheckResourceAttr("data.phare_uptime_heartbeat.test", "interval", "600"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_heartbeat.test", "grace", "60"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_heartbeat.test", "status", "waiting"),
				testingresource.TestCheckNoResourceAttr("data.phare_uptime_heartbeat.test", "last_pinged_at"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_heartbeats.filtered", "heartbeats.#", "1"),
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_heartbeats.filtered", "heartbeats.0.id", "phare_uptime_heartbeat.test", "id"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.componentable_type", "uptime/heartbeat"),
				testingresource.TestCheckResourceAttrPair("phare_uptime_status_page.test", "components.0.componentable_id", "phare_uptime_heartbeat.test", "id"),
			),
		},
	)
}
