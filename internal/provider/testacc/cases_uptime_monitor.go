package testacc

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// RunUptimeMonitorHttpResource creates a basic HTTP uptime monitor and verifies CRUD operations
func RunUptimeMonitorHttpResource(t *testing.T, s Scope) {
	name := RandomName("http")

	s.Test(t,
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_http" "test" {
	` + s.Attr() + `
	name     = "` + name + `"
	interval = 30
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 3
	recovery_confirmations = 2
	region_threshold       = 1

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "interval", "30"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "timeout", "15000"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "incident_confirmations", "3"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "recovery_confirmations", "2"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "region_threshold", "1"),
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
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_http" "test" {
	` + s.Attr() + `
	name     = "` + name + `-updated"
	interval = 60
	timeout  = 15000
	regions  = ["eu-fra-cdg"]

	incident_confirmations = 2
	recovery_confirmations = 1
	region_threshold       = 1

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "name", name+"-updated"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "interval", "60"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "region_threshold", "1"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_http.test", "request.method", "GET"),
			),
		},
		s.ImportStep("phare_uptime_monitor_http.test"),
	)
}

// RunUptimeMonitorIcmpResource creates a basic ICMP uptime monitor and verifies CRUD operations
func RunUptimeMonitorIcmpResource(t *testing.T, s Scope) {
	name := RandomName("icmp")

	s.Test(t,
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_icmp" "test" {
  ` + s.Attr() + `
  name = "` + name + `"

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "name", name),
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
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_icmp" "test" {
  ` + s.Attr() + `
  name = "` + name + `-updated"

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "name", name+"-updated"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "interval", "60"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "timeout", "10000"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "incident_confirmations", "2"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "recovery_confirmations", "1"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "region_threshold", "1"),
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_icmp.test", "request.host", "8.8.8.8"),
			),
		},
		s.ImportStep("phare_uptime_monitor_icmp.test"),
	)
}

// RunUptimeMonitorTcpResource creates a basic TCP uptime monitor and verifies CRUD operations
func RunUptimeMonitorTcpResource(t *testing.T, s Scope) {
	name := RandomName("tcp")

	s.Test(t,
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_tcp" "test" {
  ` + s.Attr() + `
  name = "` + name + `"

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "name", name),
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
		testingresource.TestStep{
			Config: `
resource "phare_uptime_monitor_tcp" "test" {
  ` + s.Attr() + `
  name = "` + name + `-updated"

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
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "name", name+"-updated"),
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
		s.ImportStep("phare_uptime_monitor_tcp.test"),
	)
}

// RunUptimeMonitorTcpResourceTags verifies the tags parameter on uptime monitors
// and the tags filter on the phare_uptime_monitors data source
func RunUptimeMonitorTcpResourceTags(t *testing.T, s Scope) {
	name := RandomName("tcp-tags")
	tag := RandomTag()

	config := func(tags string) string {
		return `
resource "phare_uptime_monitor_tcp" "test" {
  ` + s.Attr() + `
  name = "` + name + `"

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
  ` + tags + `
}
`
	}

	s.Test(t,
		testingresource.TestStep{
			Config: config(`tags = ["` + tag + `"]`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "1"),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_monitor_tcp.test", "tags.*", tag),
			),
		},
		testingresource.TestStep{
			Config: config(`tags = ["` + tag + `", "environment:production"]`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "2"),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_monitor_tcp.test", "tags.*", tag),
				testingresource.TestCheckTypeSetElemAttr("phare_uptime_monitor_tcp.test", "tags.*", "environment:production"),
			),
		},
		testingresource.TestStep{
			Config: config(``),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckNoResourceAttr("phare_uptime_monitor_tcp.test", "tags.#"),
			),
		},
		testingresource.TestStep{
			Config: config(`tags = ["`+tag+`", "environment:production"]`) + `
data "phare_uptime_monitors" "filtered" {
  ` + s.Attr() + `
  tags = ["` + tag + `", "environment:production"]

  depends_on = [phare_uptime_monitor_tcp.test]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_monitors.filtered", "monitors.*.id", "phare_uptime_monitor_tcp.test", "id"),
			),
		},
		testingresource.TestStep{
			Config: config(`tags = []`),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_uptime_monitor_tcp.test", "tags.#", "0"),
			),
		},
	)
}

// RunUptimeMonitorDataSource verifies the phare_uptime_monitor and phare_uptime_monitors data sources
func RunUptimeMonitorDataSource(t *testing.T, s Scope) {
	name := RandomName("http")

	s.Test(t,
		testingresource.TestStep{
			Config: s.HTTPMonitorFixture(name) + `
data "phare_uptime_monitor" "test" {
	` + s.Attr() + `
	id = phare_uptime_monitor_http.test.id
}

data "phare_uptime_monitors" "all" {
	` + s.Attr() + `
	depends_on = [phare_uptime_monitor_http.test]
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_monitor.test", "id", "phare_uptime_monitor_http.test", "id"),
				testingresource.TestCheckResourceAttrPair("data.phare_uptime_monitor.test", "project_id", "phare_uptime_monitor_http.test", "project_id"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_monitor.test", "name", name),
				testingresource.TestCheckResourceAttr("data.phare_uptime_monitor.test", "protocol", "http"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_monitor.test", "interval", "30"),
				testingresource.TestCheckResourceAttr("data.phare_uptime_monitor.test", "request.url", "https://invariance.dev"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_uptime_monitors.all", "monitors.*.id", "phare_uptime_monitor_http.test", "id"),
			),
		},
	)
}
