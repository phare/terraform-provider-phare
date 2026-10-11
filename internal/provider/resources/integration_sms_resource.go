package resources

import (
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// NewIntegrationSmsResource returns a new sms integration resource.
func NewIntegrationSmsResource() resource.Resource {
	return &integrationResource{
		app:                "sms",
		displayName:        "SMS",
		description:        "Manages an SMS integration in Phare, sending alerts by text message to organization members with a verified phone number. Requires a Scale plan subscription.",
		docsPage:           "sms",
		settingsAttributes: integrationMembersSettingsAttributes,
		newModel:           func() integrationModel { return &integrationMembersModel{} },
	}
}
