package testacc_org

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUptimeMonitorHttpResource creates a basic HTTP uptime monitor and verifies CRUD operations
func TestAccUptimeMonitorHttpResource(t *testing.T) {
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

resource "phare_uptime_monitor_http" "test" {
	project_scope = data.phare_project.test.slug

	name     = "HTTP Website"
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
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "name", "HTTP Website"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "interval", "30"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "timeout", "15000"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "incident_confirmations", "3"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "recovery_confirmations", "2"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.method", "HEAD"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.url", "https://invariance.dev"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.follow_redirects", "true"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.tls_skip_verify", "false"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "success_assertions.status_code.0.operator", "in"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "success_assertions.status_code.0.value", "2xx"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_http.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_http.test", "project_id"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_http.test", "status"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_http.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_uptime_monitor_http.test", "updated_at"),
				),
			},
			{
				Config: `
data "phare_project" "test" {
	slug = "test"
}

resource "phare_uptime_monitor_http" "test" {
	project_scope = data.phare_project.test.slug

	name     = "HTTP Website Updated"
	interval = 60
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 2
	recovery_confirmations = 1

	request {
		method = "GET"
		url    = "https://invariance.dev"
	}

	success_assertions {
		status_code {
			operator = "in"
			value    = "2xx"
		}
	}
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "name", "HTTP Website Updated"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "interval", "60"),
					testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.method", "GET"),
				),
			},
		},
	})
}
