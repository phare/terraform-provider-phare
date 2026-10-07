package testacc_project

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeStatusPageResourceTags verifies the tags parameter on uptime status pages
// and the tags filter on the phare_uptime_status_pages data source with a project-scoped API key.
func TestAccUptimeStatusPageResourceTags(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testacc.TestAccProjectPreCheck(t)

	monitorConfig := `
resource "phare_uptime_monitor_http" "test" {
	name     = "Website"
	interval = 30
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 3
	recovery_confirmations = 2

	request {
		method = "HEAD"
		url    = "https://invariance.dev"
	}

	success_assertions {
		status_code {
			operator = "in"
			value    = "2xx"
		}
	}
}
`

	statusPageConfig := func(tags string) string {
		return monitorConfig + `
resource "phare_uptime_status_page" "test" {
  name                  = "Status page tags"
  title                 = "Tags test status page"
  description           = "Status page used to test the tags parameter."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "tags-test"
  timeframe             = 30
  color_scheme          = "all"
` + tags + `
  theme {
    rounded      = true
    border_width = 2

    light {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#ffffff"
      foreground           = "#000000"
      foreground_muted     = "#737373"
      background_card      = "#fafafa"
    }

    dark {
      operational          = "#16a34a"
      degraded_performance = "#fbbf24"
      partial_outage       = "#f59e0b"
      major_outage         = "#ef4444"
      maintenance          = "#6366f1"
      empty                = "#d3d3d3"
      background           = "#111111"
      foreground           = "#ffffff"
      foreground_muted     = "#959595"
      background_card      = "#1a1a1a"
    }
  }

  components = [
    {
      componentable_type = "uptime/group"
      name               = "Core Services"
      is_expanded        = true
      components = [
        {
          componentable_type = "uptime/monitor"
          componentable_id   = phare_uptime_monitor_http.test.id
          display_name       = "Core Monitor"
        }
      ]
    }
  ]
}
`
	}

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: testacc.TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			{
				Config: statusPageConfig(`  tags                  = ["tfacc:status-page-tags-test"]`),
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "tags.#", "1"),
					testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", "tfacc:status-page-tags-test"),
				),
			},
			{
				Config: statusPageConfig(`  tags                  = ["tfacc:status-page-tags-test", "environment:production"]`) + `
data "phare_uptime_status_pages" "filtered" {
  tags = ["tfacc:status-page-tags-test", "environment:production"]

  depends_on = [phare_uptime_status_page.test]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "tags.#", "2"),
					testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", "tfacc:status-page-tags-test"),
					testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", "environment:production"),
					testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_status_pages.filtered", "status_pages.*.id", "phare_uptime_status_page.test", "id"),
				),
			},
			{
				Config: statusPageConfig(``),
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test", "tags.#"),
				),
			},
		},
	})
}
