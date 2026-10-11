package testacc_org

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccIntegrationDataSource verifies the phare_integration data source lookups by name
// and by ID, and the phare_integrations data source
func TestAccIntegrationDataSource(t *testing.T) {
	scope().Test(t,
		testingresource.TestStep{
			Config: `
data "phare_integration" "by_name" {
  app  = "email"
  name = "Default"
}

data "phare_integration" "by_id" {
  app = "email"
  id  = data.phare_integration.by_name.id
}

data "phare_integrations" "email" {
  app = "email"
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttrSet("data.phare_integration.by_name", "id"),
				testingresource.TestCheckResourceAttr("data.phare_integration.by_name", "name", "Default"),
				testingresource.TestCheckResourceAttr("data.phare_integration.by_id", "name", "Default"),
				testingresource.TestCheckResourceAttrPair("data.phare_integration.by_id", "id", "data.phare_integration.by_name", "id"),
				testingresource.TestCheckTypeSetElemAttrPair("data.phare_integrations.email", "integrations.*.id", "data.phare_integration.by_name", "id"),
			),
		},
	)
}
