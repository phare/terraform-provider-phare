package resources

import (
	"context"
	"encoding/json"
	"regexp"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIntegrationOutgoingWebhookResource returns a new outgoing webhook integration resource.
func NewIntegrationOutgoingWebhookResource() resource.Resource {
	return &integrationResource{
		app:                "outgoing_webhook",
		displayName:        "outgoing webhook",
		description:        "Manages an outgoing webhook integration in Phare, sending alerts as signed HTTP requests to a URL.",
		docsPage:           "outgoing-webhook",
		settingsAttributes: integrationOutgoingWebhookSettingsAttributes,
		newModel:           func() integrationModel { return &integrationOutgoingWebhookModel{} },
	}
}

type integrationOutgoingWebhookModel struct {
	IntegrationBaseModel
	Settings *integrationOutgoingWebhookSettingsModel `tfsdk:"settings"`
}

type integrationOutgoingWebhookSettingsModel struct {
	Secret types.String `tfsdk:"secret"`
	Url    types.String `tfsdk:"url"`
}

func integrationOutgoingWebhookSettingsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"secret": schema.StringAttribute{
			Required:            true,
			Sensitive:           true,
			Description:         "Secret used to sign the webhook requests (10-250 characters)." + integrationSecretNote,
			MarkdownDescription: "Secret used to sign the webhook requests (10-250 characters), [see docs](https://docs.phare.io/integrations/outgoing-webhook#delivery-details)." + integrationSecretNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(10, 250),
			},
		},
		"url": schema.StringAttribute{
			Required:            true,
			Description:         "URL receiving the webhook requests (up to 250 characters)",
			MarkdownDescription: "URL receiving the webhook requests (up to 250 characters)",
			Validators: []validator.String{
				stringvalidator.LengthAtMost(250),
				stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must be a valid http or https URL"),
			},
		},
	}
}

func (m *integrationOutgoingWebhookModel) base() *IntegrationBaseModel {
	return &m.IntegrationBaseModel
}

func (m *integrationOutgoingWebhookModel) buildSettings(ctx context.Context, prior integrationModel, diags *diag.Diagnostics) any {
	return client.OutgoingWebhookIntegrationSettings{
		URL:    m.Settings.Url.ValueString(),
		Secret: m.Settings.Secret.ValueString(),
	}
}

func (m *integrationOutgoingWebhookModel) readSettings(ctx context.Context, raw json.RawMessage, diags *diag.Diagnostics) {
	var settings client.OutgoingWebhookIntegrationSettings
	if !decodeIntegrationSettings(raw, &settings, diags) {
		return
	}

	// The secret is never returned by the API and is kept from the prior model
	if m.Settings == nil {
		m.Settings = &integrationOutgoingWebhookSettingsModel{}
	}
	m.Settings.Url = types.StringValue(settings.URL)
}
