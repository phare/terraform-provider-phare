package testacc

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Scope describes the API key a test runs with, so that the same test body
// can verify both organization-scoped and project-scoped API keys
type Scope struct {
	// keyEnv is the environment variable holding the API key
	keyEnv string

	// Project is the project slug resources are scoped to, empty for
	// project-scoped API keys
	Project string
}

// OrgScope returns the scope of an organization-scoped API key, creating
// resources in the given project
func OrgScope(projectSlug string) Scope {
	return Scope{keyEnv: "PHARE_ORG_API_KEY", Project: projectSlug}
}

// ProjectScope is the scope of a project-scoped API key
var ProjectScope = Scope{keyEnv: "PHARE_PRJ_API_KEY"}

// Attr returns the project_scope attribute to add to resources and data
// sources, empty for project-scoped API keys
func (s Scope) Attr() string {
	if s.Project == "" {
		return ""
	}

	return `project_scope = "` + s.Project + `"`
}

// Test runs an acceptance test with the API key of the scope, and verifies
// that every resource is destroyed at the end of the test
func (s Scope) Test(t *testing.T, steps ...testingresource.TestStep) {
	t.Helper()

	testingresource.Test(t, testingresource.TestCase{
		PreCheck:                 func() { s.preCheck(t) },
		ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
		CheckDestroy:             CheckDestroy(s.Project),
		Steps:                    steps,
	})
}

// ImportStep returns a step importing the given resource and verifying that
// the imported state matches the current one, except for the ignored attributes
func (s Scope) ImportStep(resourceName string, ignore ...string) testingresource.TestStep {
	step := testingresource.TestStep{
		ResourceName:            resourceName,
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: ignore,
	}

	if s.Project != "" {
		step.ImportStateIdFunc = importStateIdWithProjectScope(resourceName, s.Project)
		step.ImportStateVerifyIgnore = append(step.ImportStateVerifyIgnore, "project_scope")
	}

	return step
}

// preCheck validates that the API key of the scope is set and copies it to
// PHARE_API_KEY for the provider to use
func (s Scope) preCheck(t *testing.T) {
	v := os.Getenv(s.keyEnv)
	if v == "" {
		t.Fatalf("%s must be set for this acceptance test", s.keyEnv)
	}

	os.Setenv("PHARE_API_KEY", v)
}

// HTTPMonitorFixture returns a basic HTTP monitor resource, for tests that
// only need a monitor to reference
func (s Scope) HTTPMonitorFixture(name string) string {
	return `
resource "phare_uptime_monitor_http" "test" {
	` + s.Attr() + `
	name         = "` + name + `"
	display_name = "Core Monitor"
	interval     = 30
	timeout      = 15000
	regions      = ["eu-fra-cdg"]

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
}

// EmailIntegrationDataSource reads the default email integration, used by
// alert rule tests
const EmailIntegrationDataSource = `
data "phare_integration" "email" {
  app  = "email"
  name = "Default"
}
`
