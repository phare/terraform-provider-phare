package testacc

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUptimeStatusPageResource creates a basic uptime status page and verifies CRUD operations
func TestAccUptimeStatusPageResource(t *testing.T) {
	// Skip acceptance tests if TF_ACC environment variable is not set
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	TestAccPreCheck(t)

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_http" "test" {
	project_scope = data.phare_project.test.slug

	name     = "Website"
	interval = 30
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 3
	recovery_confirmations = 2

	request = {
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

resource "phare_uptime_status_page" "test" {
  project_scope = data.phare_project.test.slug

  name                  = "Status page"
  title                 = "Example status page"
  description           = "This is an example status page description created from Terraform."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "invariance"
  timeframe             = 30

  colors = {
    operational          = "#16a34a"
    degraded_performance = "#fbbf24"
    partial_outage       = "#f59e0b"
    major_outage         = "#ef4444"
    maintenance          = "#6366f1"
    empty                = "#d3d3d3"
  }

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test.id
    }
  ]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "name", "Status page"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "title", "Example status page"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "description", "This is an example status page description created from Terraform."),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "website_url", "https://invariance.dev"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "search_engine_indexed", "false"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "subdomain", "invariance"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "timeframe", "30"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.operational", "#16a34a"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.degraded_performance", "#fbbf24"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.partial_outage", "#f59e0b"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.major_outage", "#ef4444"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.maintenance", "#6366f1"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "colors.empty", "#d3d3d3"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.componentable_type", "uptime/monitor"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "components.0.componentable_id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "project_id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "updated_at"),
				),
			},
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_http" "test" {
	project_scope = data.phare_project.test.slug

	name     = "Website"
	interval = 30
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 3
	recovery_confirmations = 2

	request = {
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

resource "phare_uptime_status_page" "test" {
  project_scope = data.phare_project.test.slug

  name                  = "Status page updated"
  title                 = "Updated status page"
  description           = "This is an updated status page description."
  website_url           = "https://invariance.dev"
  search_engine_indexed = true
  subdomain             = "invariance"
  timeframe             = 60

  colors = {
    operational          = "#16a34a"
    degraded_performance = "#fbbf24"
    partial_outage       = "#f59e0b"
    major_outage         = "#ef4444"
    maintenance          = "#6366f1"
    empty                = "#d3d3d3"
  }

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test.id
    }
  ]
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "name", "Status page updated"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "title", "Updated status page"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "description", "This is an updated status page description."),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "search_engine_indexed", "true"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "timeframe", "60"),
				),
			},
		},
	})
}
