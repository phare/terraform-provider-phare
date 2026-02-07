package datasources

import (
	"context"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &alertRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &alertRulesDataSource{}
)

// NewAlertRulesDataSource returns a new alert rules data source.
func NewAlertRulesDataSource() datasource.DataSource {
	return &alertRulesDataSource{}
}

// alertRulesDataSource is the data source implementation.
type alertRulesDataSource struct {
	helpers.DataSourceBase
}

// alertRulesDataSourceModel describes the data source data model.
type alertRulesDataSourceModel struct {
	AlertRules   []alertRuleModel `tfsdk:"alert_rules"`
	ProjectScope types.Dynamic    `tfsdk:"project_scope"`
}

func (d *alertRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rules"
}

func (d *alertRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of alert rules.",
		Attributes: map[string]schema.Attribute{
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
			"alert_rules": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of alert rules",
				NestedObject: schema.NestedAttributeObject{
					Attributes: alertRuleSchemaAttributes(),
				},
			},
		},
	}
}

func (d *alertRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config alertRulesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_alert_rules", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all alert rules (first 100 for MVP)
	alertRules, err := scopedClient.ListAlertRules(ctx, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Alert Rules",
			err.Error(),
		)
		return
	}

	// Map API response to state
	for _, rule := range alertRules {
		ruleState := alertRuleModel{
			ID:                  types.Int64Value(rule.ID),
			Event:               types.StringValue(rule.Event),
			IntegrationID:       types.Int64Value(rule.IntegrationID),
			RateLimit:           types.Int64Value(rule.RateLimit),
			EventSettings:       types.StringValue(string(rule.EventSettings)),
			IntegrationSettings: types.StringValue(string(rule.IntegrationSettings)),
			CreatedAt:           types.StringValue(rule.CreatedAt),
			UpdatedAt:           types.StringValue(rule.UpdatedAt),
		}
		if rule.ProjectID != nil {
			ruleState.ProjectID = types.Int64Value(*rule.ProjectID)
		} else {
			ruleState.ProjectID = types.Int64Null()
		}
		config.AlertRules = append(config.AlertRules, ruleState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
