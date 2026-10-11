package testacc_org

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccProjectResource creates a basic project and verifies CRUD operations
func TestAccProjectResource(t *testing.T) {
	name := testacc.RandomName("project")

	scope().Test(t,
		testingresource.TestStep{
			Config: `
data "phare_users" "team" {
  emails = ["` + testacc.TestUserEmail + `"]
}

resource "phare_project" "test" {
	name    = "` + name + `"
	members = data.phare_users.team.users[*].id

	settings {
		use_incident_ai             = true
		use_incident_merging        = true
		incident_merging_time_window = 60
	}
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_project.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_project.test", "settings.use_incident_ai", "true"),
				testingresource.TestCheckResourceAttr("phare_project.test", "settings.use_incident_merging", "true"),
				testingresource.TestCheckResourceAttr("phare_project.test", "settings.incident_merging_time_window", "60"),
				testingresource.TestCheckResourceAttrSet("phare_project.test", "id"),
				testingresource.TestCheckResourceAttrSet("phare_project.test", "slug"),
				testingresource.TestCheckResourceAttrSet("phare_project.test", "created_at"),
				testingresource.TestCheckResourceAttrSet("phare_project.test", "updated_at"),
			),
		},
		testingresource.TestStep{
			Config: `
data "phare_users" "team" {
  emails = ["` + testacc.TestUserEmail + `"]
}

resource "phare_project" "test" {
	name    = "` + name + `"
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
		// Projects are not scoped to a project, so they are imported by ID only
		testingresource.TestStep{
			ResourceName:      "phare_project.test",
			ImportState:       true,
			ImportStateVerify: true,
		},
	)
}
