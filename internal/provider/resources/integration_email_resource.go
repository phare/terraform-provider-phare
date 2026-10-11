package resources

import (
	"context"
	"encoding/json"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// NewIntegrationEmailResource returns a new email integration resource.
func NewIntegrationEmailResource() resource.Resource {
	return &integrationResource{
		app:                "email",
		displayName:        "email",
		description:        "Manages an email integration in Phare, sending alerts by email to organization members.",
		docsPage:           "email",
		settingsAttributes: integrationMembersSettingsAttributes,
		newModel:           func() integrationModel { return &integrationMembersModel{} },
	}
}

// integrationMembersModel is the model of the email and sms integrations, alerting organization members
type integrationMembersModel struct {
	IntegrationBaseModel
	Settings *integrationMembersSettingsModel `tfsdk:"settings"`
}

type integrationMembersSettingsModel struct {
	Members types.Set `tfsdk:"members"`
}

// integrationMembersSettingsAttributes defines the settings schema of the email and sms integrations
func integrationMembersSettingsAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"members": schema.SetAttribute{
			Required:            true,
			ElementType:         types.Int64Type,
			Description:         "IDs of the organization members receiving the alerts (1-100)",
			MarkdownDescription: "IDs of the organization members receiving the alerts (1-100), see the `phare_users` data source",
			Validators: []validator.Set{
				setvalidator.SizeBetween(1, 100),
				setvalidator.ValueInt64sAre(int64validator.AtLeast(1)),
			},
		},
	}
}

func (m *integrationMembersModel) base() *IntegrationBaseModel {
	return &m.IntegrationBaseModel
}

func (m *integrationMembersModel) buildSettings(ctx context.Context, prior integrationModel, diags *diag.Diagnostics) any {
	var settings client.MembersIntegrationSettings
	diags.Append(m.Settings.Members.ElementsAs(ctx, &settings.Members, false)...)
	return settings
}

func (m *integrationMembersModel) readSettings(ctx context.Context, raw json.RawMessage, diags *diag.Diagnostics) {
	var settings client.MembersIntegrationSettings
	if !decodeIntegrationSettings(raw, &settings, diags) {
		return
	}

	members, d := types.SetValueFrom(ctx, types.Int64Type, settings.Members)
	diags.Append(d...)
	m.Settings = &integrationMembersSettingsModel{Members: members}
}
