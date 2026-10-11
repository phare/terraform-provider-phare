package testacc

import (
	"strings"
	"testing"

	testingresource "github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// Fake credentials, the API does not verify them with the integrated services
var (
	testIlertIntegrationKeys = []string{"il1api" + strings.Repeat("0", 40), "il1api" + strings.Repeat("1", 40)}
	testNtfyAccessToken      = "tk_" + strings.Repeat("0", 29)
	testPushoverUserKeys     = []string{"u" + strings.Repeat("0", 29), "u" + strings.Repeat("1", 29)}
	testPushoverAPIKey       = "a" + strings.Repeat("0", 29)
	testWebhookSecrets       = []string{"a-very-secure-secret", "another-very-secure-secret"}
)

// integrationImportStep returns a step importing the given integration and
// verifying that the imported state matches the current one, except for the
// ignored secret settings the API never returns. Integrations belong to the
// organization, so the import ID is the integration ID with every API key.
func integrationImportStep(resourceName string, ignore ...string) testingresource.TestStep {
	return testingresource.TestStep{
		ResourceName:            resourceName,
		ImportState:             true,
		ImportStateVerify:       true,
		ImportStateVerifyIgnore: ignore,
	}
}

// integrationEmailConfig returns an email integration alerting the test user
func integrationEmailConfig(name string) string {
	return `
data "phare_users" "team" {
  emails = ["` + TestUserEmail + `"]
}

resource "phare_integration_email" "test" {
  name = "` + name + `"

  settings = {
    members = data.phare_users.team.users[*].id
  }
}
`
}

// RunIntegrationEmailResource creates an email integration, verifies CRUD
// operations and its use by an alert rule
func RunIntegrationEmailResource(t *testing.T, s Scope) {
	name := RandomName("email")
	updatedName := RandomName("email")

	s.Test(t,
		testingresource.TestStep{
			Config: integrationEmailConfig(name),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_email.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_integration_email.test", "settings.members.#", "1"),
				testingresource.TestCheckTypeSetElemAttrPair("phare_integration_email.test", "settings.members.*", "data.phare_users.team", "users.0.id"),
				testingresource.TestCheckResourceAttr("phare_integration_email.test", "paused", "false"),
				testingresource.TestCheckResourceAttrSet("phare_integration_email.test", "id"),
				testingresource.TestCheckResourceAttrSet("phare_integration_email.test", "created_at"),
				testingresource.TestCheckResourceAttrSet("phare_integration_email.test", "updated_at"),
			),
		},
		integrationImportStep("phare_integration_email.test"),
		// The alert rule is destroyed before the integration it uses
		testingresource.TestStep{
			Config: integrationEmailConfig(updatedName) + `
resource "phare_alert_rule" "test" {
  ` + s.Attr() + `
  event          = "uptime.incident.created"
  scope          = "project"
  integration_id = phare_integration_email.test.id
  rate_limit     = 5

  event_settings = jsonencode({
    type = "all"
  })
}
`,
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_email.test", "name", updatedName),
				testingresource.TestCheckResourceAttrPair("phare_alert_rule.test", "integration_id", "phare_integration_email.test", "id"),
			),
		},
	)
}

// integrationIlertConfig returns an ilert integration
func integrationIlertConfig(name, integrationKey string) string {
	return `
resource "phare_integration_ilert" "test" {
  name = "` + name + `"

  settings = {
    integration_key = "` + integrationKey + `"
  }
}
`
}

// RunIntegrationIlertResource creates an ilert integration and verifies CRUD operations
func RunIntegrationIlertResource(t *testing.T, s Scope) {
	name := RandomName("ilert")
	updatedName := RandomName("ilert")

	s.Test(t,
		testingresource.TestStep{
			Config: integrationIlertConfig(name, testIlertIntegrationKeys[0]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_ilert.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_integration_ilert.test", "settings.integration_key", testIlertIntegrationKeys[0]),
				testingresource.TestCheckResourceAttrSet("phare_integration_ilert.test", "id"),
			),
		},
		integrationImportStep("phare_integration_ilert.test", "settings.integration_key"),
		testingresource.TestStep{
			Config: integrationIlertConfig(updatedName, testIlertIntegrationKeys[1]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_ilert.test", "name", updatedName),
				testingresource.TestCheckResourceAttr("phare_integration_ilert.test", "settings.integration_key", testIlertIntegrationKeys[1]),
			),
		},
	)
}

// integrationNtfyConfig returns an ntfy integration, with an access token
// when accessToken is not empty
func integrationNtfyConfig(name, topic, accessToken string) string {
	accessTokenAttr := ""
	if accessToken != "" {
		accessTokenAttr = `access_token = "` + accessToken + `"`
	}

	return `
resource "phare_integration_ntfy" "test" {
  name = "` + name + `"

  settings = {
    hostname = "ntfy.sh"
    topic    = "` + topic + `"
    ` + accessTokenAttr + `
  }
}
`
}

// RunIntegrationNtfyResource creates an ntfy integration, verifies CRUD
// operations and the removal of its access token
func RunIntegrationNtfyResource(t *testing.T, s Scope) {
	name := RandomName("ntfy")
	topic := RandomName("topic")
	updatedTopic := RandomName("topic")

	s.Test(t,
		testingresource.TestStep{
			Config: integrationNtfyConfig(name, topic, testNtfyAccessToken),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_ntfy.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_integration_ntfy.test", "settings.hostname", "ntfy.sh"),
				testingresource.TestCheckResourceAttr("phare_integration_ntfy.test", "settings.topic", topic),
				testingresource.TestCheckResourceAttr("phare_integration_ntfy.test", "settings.access_token", testNtfyAccessToken),
				testingresource.TestCheckResourceAttrSet("phare_integration_ntfy.test", "id"),
			),
		},
		integrationImportStep("phare_integration_ntfy.test", "settings.access_token"),
		testingresource.TestStep{
			Config: integrationNtfyConfig(name, updatedTopic, ""),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_ntfy.test", "settings.topic", updatedTopic),
				testingresource.TestCheckNoResourceAttr("phare_integration_ntfy.test", "settings.access_token"),
			),
		},
	)
}

// integrationOutgoingWebhookConfig returns an outgoing webhook integration
func integrationOutgoingWebhookConfig(name, url, secret string) string {
	return `
resource "phare_integration_outgoing_webhook" "test" {
  name = "` + name + `"

  settings = {
    url    = "` + url + `"
    secret = "` + secret + `"
  }
}
`
}

// RunIntegrationOutgoingWebhookResource creates an outgoing webhook integration and verifies CRUD operations
func RunIntegrationOutgoingWebhookResource(t *testing.T, s Scope) {
	name := RandomName("webhook")

	s.Test(t,
		testingresource.TestStep{
			Config: integrationOutgoingWebhookConfig(name, "https://example.com/webhooks/phare", testWebhookSecrets[0]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_outgoing_webhook.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_integration_outgoing_webhook.test", "settings.url", "https://example.com/webhooks/phare"),
				testingresource.TestCheckResourceAttr("phare_integration_outgoing_webhook.test", "settings.secret", testWebhookSecrets[0]),
				testingresource.TestCheckResourceAttrSet("phare_integration_outgoing_webhook.test", "id"),
			),
		},
		integrationImportStep("phare_integration_outgoing_webhook.test", "settings.secret"),
		testingresource.TestStep{
			Config: integrationOutgoingWebhookConfig(name, "https://example.com/webhooks/phare-updated", testWebhookSecrets[1]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_outgoing_webhook.test", "settings.url", "https://example.com/webhooks/phare-updated"),
				testingresource.TestCheckResourceAttr("phare_integration_outgoing_webhook.test", "settings.secret", testWebhookSecrets[1]),
			),
		},
	)
}

// integrationPushoverConfig returns a pushover integration
func integrationPushoverConfig(name, userKey string) string {
	return `
resource "phare_integration_pushover" "test" {
  name = "` + name + `"

  settings = {
    user_key = "` + userKey + `"
    api_key  = "` + testPushoverAPIKey + `"
  }
}
`
}

// RunIntegrationPushoverResource creates a pushover integration and verifies CRUD operations
func RunIntegrationPushoverResource(t *testing.T, s Scope) {
	name := RandomName("pushover")

	s.Test(t,
		testingresource.TestStep{
			Config: integrationPushoverConfig(name, testPushoverUserKeys[0]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_pushover.test", "name", name),
				testingresource.TestCheckResourceAttr("phare_integration_pushover.test", "settings.user_key", testPushoverUserKeys[0]),
				testingresource.TestCheckResourceAttr("phare_integration_pushover.test", "settings.api_key", testPushoverAPIKey),
				testingresource.TestCheckResourceAttrSet("phare_integration_pushover.test", "id"),
			),
		},
		integrationImportStep("phare_integration_pushover.test", "settings.user_key", "settings.api_key"),
		testingresource.TestStep{
			Config: integrationPushoverConfig(name, testPushoverUserKeys[1]),
			Check: testingresource.ComposeAggregateTestCheckFunc(
				testingresource.TestCheckResourceAttr("phare_integration_pushover.test", "settings.user_key", testPushoverUserKeys[1]),
			),
		},
	)
}
