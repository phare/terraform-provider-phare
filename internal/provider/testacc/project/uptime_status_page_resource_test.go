package testacc_project

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// getTestDataDir returns the absolute path to the testdata directory
func getTestDataDir() string {
	_, currentFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(currentFile), "..", "testdata")
}

// TestAccUptimeStatusPageResource creates a basic uptime status page and verifies CRUD operations with project-scoped API key
func TestAccUptimeStatusPageResource(t *testing.T) {
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

resource "phare_uptime_status_page" "test" {
  name                  = "Status page"
  title                 = "Example status page"
  description           = "This is an example status page description created from Terraform."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "projectscoped"
  timeframe             = 30
  color_scheme          = "all"

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
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "subdomain", "projectscoped"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "timeframe", "30"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "color_scheme", "all"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.rounded", "true"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.border_width", "2"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.operational", "#16a34a"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.degraded_performance", "#fbbf24"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.background", "#ffffff"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.dark.operational", "#16a34a"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.dark.background", "#111111"),
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

resource "phare_uptime_status_page" "test" {
  name                  = "Status page updated"
  title                 = "Updated status page"
  description           = "This is an updated status page description."
  website_url           = "https://invariance.dev"
  search_engine_indexed = true
  subdomain             = "projectscoped"
  timeframe             = 60
  color_scheme          = "all"

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

// statusPageThemeBlock is a reusable theme configuration for file upload tests
const statusPageThemeBlock = `
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
`

// TestAccUptimeStatusPageResourceWithFiles tests status page file uploads with project-scoped API key
func TestAccUptimeStatusPageResourceWithFiles(t *testing.T) {
	// Skip acceptance tests if TF_ACC environment variable is not set
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Acceptance tests skipped unless env 'TF_ACC' set")
	}

	testacc.TestAccProjectPreCheck(t)

	testDataDir := getTestDataDir()

	testingresource.Test(t, testingresource.TestCase{
		ProtoV6ProviderFactories: testacc.TestAccProtoV6ProviderFactories,
		Steps: []testingresource.TestStep{
			// Step 1: Create status page with logo files
			{
				Config: `
resource "phare_uptime_monitor_http" "test_files" {
	name     = "Website for files test"
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

resource "phare_uptime_status_page" "test_files" {
  name                  = "Status page with files"
  title                 = "Status page with logo"
  description           = "Testing file uploads"
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "prjscoped-files"
  timeframe             = 30
  color_scheme          = "all"

  logo_light = "` + filepath.Join(testDataDir, "logo_light.png") + `"
  logo_dark  = "` + filepath.Join(testDataDir, "logo_dark.png") + `"

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test_files.id
    }
  ]
` + statusPageThemeBlock + `
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "name", "Status page with files"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "logo_light", filepath.Join(testDataDir, "logo_light.png")),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "logo_dark", filepath.Join(testDataDir, "logo_dark.png")),
					testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test_files", "id"),
				),
			},
			// Step 2: Add favicon files
			{
				Config: `
resource "phare_uptime_monitor_http" "test_files" {
	name     = "Website for files test"
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

resource "phare_uptime_status_page" "test_files" {
  name                  = "Status page with files"
  title                 = "Status page with logo and favicon"
  description           = "Testing file uploads"
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "prjscoped-files"
  timeframe             = 30
  color_scheme          = "all"

  logo_light    = "` + filepath.Join(testDataDir, "logo_light.png") + `"
  logo_dark     = "` + filepath.Join(testDataDir, "logo_dark.png") + `"
  favicon_light = "` + filepath.Join(testDataDir, "favicon_light.png") + `"
  favicon_dark  = "` + filepath.Join(testDataDir, "favicon.svg") + `"

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test_files.id
    }
  ]
` + statusPageThemeBlock + `
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "title", "Status page with logo and favicon"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_light", filepath.Join(testDataDir, "favicon_light.png")),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_dark", filepath.Join(testDataDir, "favicon.svg")),
				),
			},
			// Step 3: Remove logos (keep favicons)
			{
				Config: `
resource "phare_uptime_monitor_http" "test_files" {
	name     = "Website for files test"
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

resource "phare_uptime_status_page" "test_files" {
  name                  = "Status page with files"
  title                 = "Status page with favicon only"
  description           = "Testing file uploads"
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "prjscoped-files"
  timeframe             = 30
  color_scheme          = "all"

  favicon_light = "` + filepath.Join(testDataDir, "favicon_light.png") + `"
  favicon_dark  = "` + filepath.Join(testDataDir, "favicon.svg") + `"

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test_files.id
    }
  ]
` + statusPageThemeBlock + `
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "title", "Status page with favicon only"),
					testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test_files", "logo_light"),
					testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test_files", "logo_dark"),
					testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_light", filepath.Join(testDataDir, "favicon_light.png")),
				),
			},
		},
	})
}
