package resources

import (
	"context"
	"encoding/json"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIntegrationPushoverResource returns a new pushover integration resource.
func NewIntegrationPushoverResource() resource.Resource {
	return &integrationResource{
		app:                "pushover",
		displayName:        "Pushover",
		description:        "Manages a Pushover integration in Phare, sending alerts as Pushover notifications.",
		docsPage:           "pushover",
		settingsAttributes: integrationPushoverSettingsAttributes,
		newModel:           func() integrationModel { return &integrationPushoverModel{} },
	}
}

type integrationPushoverModel struct {
	IntegrationBaseModel
	Settings *integrationPushoverSettingsModel `tfsdk:"settings"`
}

type integrationPushoverSettingsModel struct {
	ApiKey  types.String `tfsdk:"api_key"`
	UserKey types.String `tfsdk:"user_key"`
}

func integrationPushoverSettingsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"api_key": schema.StringAttribute{
			Required:            true,
			Sensitive:           true,
			Description:         "Pushover application API token (30 characters)." + integrationSecretNote,
			MarkdownDescription: "Pushover application API token (30 characters)." + integrationSecretNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(30, 30),
			},
		},
		"user_key": schema.StringAttribute{
			Required:            true,
			Sensitive:           true,
			Description:         "Pushover user or group key (30 characters)." + integrationSecretNote,
			MarkdownDescription: "Pushover user or group key (30 characters)." + integrationSecretNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(30, 30),
			},
		},
	}
}

func (m *integrationPushoverModel) base() *IntegrationBaseModel {
	return &m.IntegrationBaseModel
}

func (m *integrationPushoverModel) buildSettings(ctx context.Context, diags *diag.Diagnostics) any {
	return client.PushoverIntegrationSettings{
		UserKey: m.Settings.UserKey.ValueString(),
		APIKey:  m.Settings.ApiKey.ValueString(),
	}
}

func (m *integrationPushoverModel) readSettings(ctx context.Context, raw json.RawMessage, diags *diag.Diagnostics) {
	// The API returns no pushover settings, the keys are kept from the prior model
	if m.Settings == nil {
		m.Settings = &integrationPushoverSettingsModel{}
	}
}
