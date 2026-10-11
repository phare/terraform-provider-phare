package resources

import (
	"context"
	"encoding/json"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIntegrationNtfyResource returns a new ntfy integration resource.
func NewIntegrationNtfyResource() resource.Resource {
	return &integrationResource{
		app:                "ntfy",
		displayName:        "ntfy",
		description:        "Manages an ntfy integration in Phare, publishing alerts to an ntfy topic.",
		docsPage:           "ntfy",
		settingsAttributes: integrationNtfySettingsAttributes,
		newModel:           func() integrationModel { return &integrationNtfyModel{} },
	}
}

type integrationNtfyModel struct {
	IntegrationBaseModel
	Settings *integrationNtfySettingsModel `tfsdk:"settings"`
}

type integrationNtfySettingsModel struct {
	AccessToken types.String `tfsdk:"access_token"`
	Hostname    types.String `tfsdk:"hostname"`
	Topic       types.String `tfsdk:"topic"`
}

func integrationNtfySettingsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"access_token": schema.StringAttribute{
			Optional:            true,
			Sensitive:           true,
			Description:         "ntfy access token used for authenticated topics (32 characters starting with tk_)." + integrationSecretNote,
			MarkdownDescription: "ntfy access token used for authenticated topics (32 characters starting with `tk_`)." + integrationSecretNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(32, 32),
				helpers.HasPrefix("tk_"),
			},
		},
		"hostname": schema.StringAttribute{
			Required:            true,
			Description:         "Hostname of the ntfy server, without scheme (e.g. ntfy.sh)",
			MarkdownDescription: "Hostname of the ntfy server, without scheme (e.g. `ntfy.sh`)",
			Validators: []validator.String{
				stringvalidator.LengthBetween(4, 100),
				helpers.Hostname(),
			},
		},
		"topic": schema.StringAttribute{
			Required:            true,
			Description:         "ntfy topic receiving the alerts (1-255 characters)",
			MarkdownDescription: "ntfy topic receiving the alerts (1-255 characters)",
			Validators: []validator.String{
				stringvalidator.LengthBetween(1, 255),
			},
		},
	}
}

func (m *integrationNtfyModel) base() *IntegrationBaseModel {
	return &m.IntegrationBaseModel
}

func (m *integrationNtfyModel) buildSettings(ctx context.Context, diags *diag.Diagnostics) any {
	return client.NtfyIntegrationSettings{
		Hostname: m.Settings.Hostname.ValueString(),
		Topic:    m.Settings.Topic.ValueString(),
		// A null access token removes the token from the integration
		AccessToken: m.Settings.AccessToken.ValueStringPointer(),
	}
}

func (m *integrationNtfyModel) readSettings(ctx context.Context, raw json.RawMessage, diags *diag.Diagnostics) {
	var settings client.NtfyIntegrationSettings
	if !decodeIntegrationSettings(raw, &settings, diags) {
		return
	}

	// The access token is never returned by the API and is kept from the prior model
	if m.Settings == nil {
		m.Settings = &integrationNtfySettingsModel{}
	}
	m.Settings.Hostname = types.StringValue(settings.Hostname)
	m.Settings.Topic = types.StringValue(settings.Topic)
}
