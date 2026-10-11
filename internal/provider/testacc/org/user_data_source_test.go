package testacc_org

import (
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"

	"terraform-provider-phare/internal/provider/testacc"
)

// TestAccUserDataSource verifies the phare_users data source email filter
// and the phare_user data source lookup by ID
func TestAccUserDataSource(t *testing.T) {
	scope().Test(t,
		testingresource.TestStep{
			Config: `
data "phare_users" "team" {
  emails = ["` + testacc.TestUserEmail + `"]
}

data "phare_user" "test" {
  id = data.phare_users.team.users[0].id
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("data.phare_users.team", "users.#", "1"),
				testingresource.TestCheckResourceAttr("data.phare_users.team", "users.0.email", testacc.TestUserEmail),
				testingresource.TestCheckResourceAttrPair("data.phare_user.test", "id", "data.phare_users.team", "users.0.id"),
				testingresource.TestCheckResourceAttr("data.phare_user.test", "email", testacc.TestUserEmail),
				testingresource.TestCheckResourceAttrSet("data.phare_user.test", "role"),
			),
		},
	)
}
