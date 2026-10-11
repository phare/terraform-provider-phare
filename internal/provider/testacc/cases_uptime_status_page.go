package testacc

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// RunUptimeStatusPageResource creates a basic uptime status page and verifies CRUD operations
func RunUptimeStatusPageResource(t *testing.T, s Scope) {
	monitorName := RandomName("http")
	name := RandomName("page")
	subdomain := RandomSubdomain()

	s.Test(t,
		testingresource.TestStep{
			Config: s.HTTPMonitorFixture(monitorName) + `
resource "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  name                  = "` + name + `"
  title                 = "Example status page"
  description           = "This is an example status page description created from Terraform."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "` + subdomain + `"
  timeframe             = 30
  color_scheme          = "all"
` + StatusPageThemeBlock + `
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
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "title", "Example status page"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "description", "This is an example status page description created from Terraform."),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "website_url", "https://invariance.dev"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "search_engine_indexed", "false"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "subdomain", subdomain),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "timeframe", "30"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "show_response_times", "true"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "color_scheme", "all"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.rounded", "true"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.border_width", "2"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.operational", "#16a34a"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.degraded_performance", "#fbbf24"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.background", "#ffffff"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.light.border", "#e5e7eb"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.dark.operational", "#16a34a"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.dark.background", "#111111"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "theme.dark.border", "#232323"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.componentable_type", "uptime/group"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.name", "Core Services"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.is_expanded", "true"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.components.0.componentable_type", "uptime/monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "components.0.components.0.componentable_id"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.components.0.display_name", "Core Monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "id"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "project_id"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "created_at"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "updated_at"),
			),
		},
		testingresource.TestStep{
			Config: s.HTTPMonitorFixture(monitorName) + `
resource "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  name                  = "` + name + `-updated"
  title                 = "Updated status page"
  description           = "This is an updated status page description."
  website_url           = "https://invariance.dev"
  search_engine_indexed = true
  subdomain             = "` + subdomain + `"
  timeframe             = 60
  color_scheme          = "all"
  show_response_times   = false
` + StatusPageThemeBlock + `
  components = [
    {
      componentable_type = "uptime/group"
      name               = "Core Services Updated"
      is_expanded        = false
      components = [
        {
          componentable_type = "uptime/monitor"
          componentable_id   = phare_uptime_monitor_http.test.id
        }
      ]
    }
  ]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "name", name+"-updated"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "title", "Updated status page"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "description", "This is an updated status page description."),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "search_engine_indexed", "true"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "timeframe", "60"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "show_response_times", "false"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.componentable_type", "uptime/group"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.name", "Core Services Updated"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.is_expanded", "false"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "components.0.components.0.componentable_type", "uptime/monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test", "components.0.components.0.componentable_id"),
			),
		},
		s.ImportStep("phare_uptime_status_page.test"),
	)
}

// RunUptimeStatusPageResourceWithFiles tests status page creation with logo and favicon file uploads
func RunUptimeStatusPageResourceWithFiles(t *testing.T, s Scope) {
	monitorName := RandomName("http")
	name := RandomName("page")
	subdomain := RandomSubdomain()

	logoLight := TestDataFile("logo_light.png")
	logoDark := TestDataFile("logo_dark.png")
	faviconLight := TestDataFile("favicon_light.png")
	faviconDark := TestDataFile("favicon.svg")

	config := func(title, files string) string {
		return s.HTTPMonitorFixture(monitorName) + `
resource "phare_uptime_status_page" "test_files" {
  ` + s.Attr() + `
  name                  = "` + name + `"
  title                 = "` + title + `"
  description           = "Testing file uploads"
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "` + subdomain + `"
  timeframe             = 30
  color_scheme          = "all"

` + files + `

  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test.id
    }
  ]
` + StatusPageThemeBlock + `
}
`
	}

	s.Test(t,
		// Step 1: Create status page with logo files
		testingresource.TestStep{
			Config: config("Status page with logo", `
  logo_light = "`+logoLight+`"
  logo_dark  = "`+logoDark+`"`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "name", name),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "logo_light", logoLight),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "logo_dark", logoDark),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test_files", "id"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "components.0.componentable_type", "uptime/monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test_files", "components.0.componentable_id"),
			),
		},
		// Step 2: Add favicon files
		testingresource.TestStep{
			Config: config("Status page with logo and favicon", `
  logo_light    = "`+logoLight+`"
  logo_dark     = "`+logoDark+`"
  favicon_light = "`+faviconLight+`"
  favicon_dark  = "`+faviconDark+`"`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "title", "Status page with logo and favicon"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_light", faviconLight),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_dark", faviconDark),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "components.0.componentable_type", "uptime/monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test_files", "components.0.componentable_id"),
			),
		},
		// Step 3: Remove logos (keep favicons)
		testingresource.TestStep{
			Config: config("Status page with favicon only", `
  favicon_light = "`+faviconLight+`"
  favicon_dark  = "`+faviconDark+`"`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "title", "Status page with favicon only"),
				testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test_files", "logo_light"),
				testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test_files", "logo_dark"),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "favicon_light", faviconLight),
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test_files", "components.0.componentable_type", "uptime/monitor"),
				testingresource.TestCheckResourceAttrSet("phare_uptime_status_page.test_files", "components.0.componentable_id"),
			),
		},
	)
}

// RunUptimeStatusPageResourceTags verifies the tags parameter on uptime status pages
// and the tags filter on the phare_uptime_status_pages data source
func RunUptimeStatusPageResourceTags(t *testing.T, s Scope) {
	monitorName := RandomName("http")
	name := RandomName("page")
	subdomain := RandomSubdomain()
	tag := RandomTag()

	config := func(tags string) string {
		return s.HTTPMonitorFixture(monitorName) + `
resource "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  name                  = "` + name + `"
  title                 = "Tags test status page"
  description           = "Status page used to test the tags parameter."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "` + subdomain + `"
  timeframe             = 30
  color_scheme          = "all"
  ` + tags + `
` + StatusPageThemeBlock + `
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

	s.Test(t,
		testingresource.TestStep{
			Config: config(`tags = ["` + tag + `"]`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "tags.#", "1"),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", tag),
			),
		},
		testingresource.TestStep{
			Config: config(`tags = ["`+tag+`", "environment:production"]`) + `
data "phare_uptime_status_pages" "filtered" {
  ` + s.Attr() + `
  tags = ["` + tag + `", "environment:production"]

  depends_on = [phare_uptime_status_page.test]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_status_page.test", "tags.#", "2"),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", tag),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_status_page.test", "tags.*", "environment:production"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_status_pages.filtered", "status_pages.*.id", "phare_uptime_status_page.test", "id"),
			),
		},
		testingresource.TestStep{
			Config: config(``),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckNoResourceAttr("phare_uptime_status_page.test", "tags.#"),
			),
		},
	)
}

// RunUptimeStatusPageDataSource verifies the phare_uptime_status_page and phare_uptime_status_pages data sources
func RunUptimeStatusPageDataSource(t *testing.T, s Scope) {
	monitorName := RandomName("http")
	name := RandomName("page")
	subdomain := RandomSubdomain()

	s.Test(t,
		testingresource.TestStep{
			Config: s.HTTPMonitorFixture(monitorName) + `
resource "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  name                  = "` + name + `"
  title                 = "Data source status page"
  description           = "Status page used to test the data sources."
  website_url           = "https://invariance.dev"
  search_engine_indexed = false
  subdomain             = "` + subdomain + `"
  timeframe             = 30
  color_scheme          = "all"
` + StatusPageThemeBlock + `
  components = [
    {
      componentable_type = "uptime/monitor"
      componentable_id   = phare_uptime_monitor_http.test.id
    }
  ]
}

data "phare_uptime_status_page" "test" {
  ` + s.Attr() + `
  id = phare_uptime_status_page.test.id
}

data "phare_uptime_status_pages" "all" {
  ` + s.Attr() + `
  depends_on = [phare_uptime_status_page.test]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_status_page.test", "id", "phare_uptime_status_page.test", "id"),
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_status_page.test", "project_id", "phare_uptime_status_page.test", "project_id"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_status_page.test", "name", name),
				testingresource.TestCheckResourceAttr("data.phare_uptime_status_page.test", "subdomain", subdomain),
				testingresource.TestCheckResourceAttr("data.phare_uptime_status_page.test", "title", "Data source status page"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_status_page.test", "access_password_enabled", "false"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_status_pages.all", "status_pages.*.id", "phare_uptime_status_page.test", "id"),
			),
		},
	)
}
