package testacc_org

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccProjectDataSource verifies the phare_project and phare_projects data sources
// by reading the project of the test run
func TestAccProjectDataSource(t *testing.T) {
	scope().Test(t,
		testingresource.TestStep{
			Config: `
data "phare_project" "by_slug" {
  slug = "` + runProjectSlug + `"
}

data "phare_project" "by_id" {
  id = data.phare_project.by_slug.id
}

data "phare_projects" "all" {}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("data.phare_project.by_slug", "slug", runProjectSlug),
				testingresource.TestCheckResourceAttrSet("data.phare_project.by_slug", "id"),
				testingresource.TestCheckResourceAttrSet("data.phare_project.by_slug", "name"),
				testingresource.TestCheckResourceAttrPair("data.phare_project.by_id", "slug", "data.phare_project.by_slug", "slug"),
				testingresource.TestCheckResourceAttrPair("data.phare_project.by_id", "name", "data.phare_project.by_slug", "name"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_projects.all", "projects.*.id", "data.phare_project.by_slug", "id"),
			),
		},
	)
}
