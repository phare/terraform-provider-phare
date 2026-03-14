package testacc_org

import (
	"os"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccProjectResource creates a basic project and verifies CRUD operations
func TestAccProjectResource(t *testing.T) {
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
data "phare_users" "team" {
  emails = ["terraform@phare.io"]
}

resource "phare_project" "test" {
	name    = "Test Project"
	members = data.phare_users.team.users[*].id

	settings {
		use_incident_ai             = true
		use_incident_merging        = true
		incident_merging_time_window = 60
	}
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_project.test", "name", "Test Project"),
					testingresource.TestCheckResourceAttr("phare_project.test", "settings.use_incident_ai", "true"),
					testingresource.TestCheckResourceAttr("phare_project.test", "settings.use_incident_merging", "true"),
					testingresource.TestCheckResourceAttr("phare_project.test", "settings.incident_merging_time_window", "60"),
					testingresource.TestCheckResourceAttrSet("phare_project.test", "id"),
					testingresource.TestCheckResourceAttrSet("phare_project.test", "slug"),
					testingresource.TestCheckResourceAttrSet("phare_project.test", "created_at"),
					testingresource.TestCheckResourceAttrSet("phare_project.test", "updated_at"),
				),
			},
			{
				Config: `
data "phare_users" "team" {
  emails = ["terraform@phare.io"]
}

resource "phare_project" "test" {
	name    = "Test Project"
	members = data.phare_users.team.users[*].id

	settings {
		use_incident_ai             = false
		use_incident_merging        = true
		incident_merging_time_window = 30
	}
}
`,
				Check: testingresource.ComposeAggregateTestCheckFunc(
					testingresource.TestCheckResourceAttr("phare_project.test", "settings.use_incident_ai", "false"),
					testingresource.TestCheckResourceAttr("phare_project.test", "settings.incident_merging_time_window", "30"),
				),
			},
		},
	})
}
