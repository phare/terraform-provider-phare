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

// NewIntegrationIlertResource returns a new ilert integration resource.
func NewIntegrationIlertResource() resource.Resource {
	return &integrationResource{
		app:                "ilert",
		displayName:        "ilert",
		description:        "Manages an ilert integration in Phare, routing alerts to an ilert alert source.",
		docsPage:           "ilert",
		settingsAttributes: integrationIlertSettingsAttributes,
		newModel:           func() integrationModel { return &integrationIlertModel{} },
	}
}

type integrationIlertModel struct {
	IntegrationBaseModel
	Settings *integrationIlertSettingsModel `tfsdk:"settings"`
}

type integrationIlertSettingsModel struct {
	IntegrationKey types.String `tfsdk:"integration_key"`
}

func integrationIlertSettingsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"integration_key": schema.StringAttribute{
			Required:            true,
			Sensitive:           true,
			Description:         "ilert alert source integration key (40-70 characters)." + integrationSecretNote,
			MarkdownDescription: "ilert alert source integration key (40-70 characters)." + integrationSecretNote,
			Validators: []validator.String{
				stringvalidator.LengthBetween(40, 70),
			},
		},
	}
}

func (m *integrationIlertModel) base() *IntegrationBaseModel {
	return &m.IntegrationBaseModel
}

func (m *integrationIlertModel) buildSettings(ctx context.Context, diags *diag.Diagnostics) any {
	return client.IlertIntegrationSettings{
		IntegrationKey: m.Settings.IntegrationKey.ValueString(),
	}
}

func (m *integrationIlertModel) readSettings(ctx context.Context, raw json.RawMessage, diags *diag.Diagnostics) {
	// The API returns no ilert settings, the integration key is kept from the prior model
	if m.Settings == nil {
		m.Settings = &integrationIlertSettingsModel{}
	}
}
