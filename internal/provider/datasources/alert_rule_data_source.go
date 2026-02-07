package datasources

import (
	"context"
	"fmt"
	"maps"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &alertRuleDataSource{}
	_ datasource.DataSourceWithConfigure = &alertRuleDataSource{}
)

// NewAlertRuleDataSource returns a new alert rule data source.
func NewAlertRuleDataSource() datasource.DataSource {
	return &alertRuleDataSource{}
}

// alertRuleDataSource is the data source implementation.
type alertRuleDataSource struct {
	helpers.DataSourceBase
}

// alertRuleModel describes the common alert rule data model used by both
// the single alert rule and alert rules list data sources.
type alertRuleModel struct {
	ID                  types.Int64  `tfsdk:"id"`
	Event               types.String `tfsdk:"event"`
	ProjectID           types.Int64  `tfsdk:"project_id"`
	IntegrationID       types.Int64  `tfsdk:"integration_id"`
	RateLimit           types.Int64  `tfsdk:"rate_limit"`
	EventSettings       types.String `tfsdk:"event_settings"`
	IntegrationSettings types.String `tfsdk:"integration_settings"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// alertRuleSchemaAttributes returns the common schema attributes for alert rules
func alertRuleSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Alert rule ID",
		},
		"event": schema.StringAttribute{
			Computed:    true,
			Description: "Event name that triggers the alert rule",
		},
		"project_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Project ID (optional scope)",
		},
		"integration_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Integration ID for notifications",
		},
		"rate_limit": schema.Int64Attribute{
			Computed:    true,
			Description: "Rate limit in minutes",
		},
		"event_settings": schema.StringAttribute{
			Computed:    true,
			Description: "Event-specific settings as JSON",
		},
		"integration_settings": schema.StringAttribute{
			Computed:    true,
			Description: "Integration-specific settings as JSON",
		},
		"created_at": schema.StringAttribute{
			Computed:    true,
			Description: "Creation timestamp",
		},
		"updated_at": schema.StringAttribute{
			Computed:    true,
			Description: "Last update timestamp",
		},
	}
}

// scopedAlertRuleModel describes the data source data model.
type scopedAlertRuleModel struct {
	alertRuleModel
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
}

func (d *alertRuleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (d *alertRuleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific alert rule by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Alert rule ID",
			},
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
		},
	}

	// Add the common alert rule attributes
	maps.Copy(resp.Schema.Attributes, alertRuleSchemaAttributes())
}

func (d *alertRuleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scopedAlertRuleModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the alert rule by ID
	alertRule, err := scopedClient.GetAlertRule(ctx, config.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Alert Rule",
			fmt.Sprintf("Error reading alert rule ID %d: %s", config.ID.ValueInt64(), err.Error()),
		)
		return
	}

	// Map API response to state
	config.Event = types.StringValue(alertRule.Event)
	config.IntegrationID = types.Int64Value(alertRule.IntegrationID)
	config.RateLimit = types.Int64Value(alertRule.RateLimit)
	if alertRule.ProjectID != nil {
		config.ProjectID = types.Int64Value(*alertRule.ProjectID)
	} else {
		config.ProjectID = types.Int64Null()
	}
	config.EventSettings = types.StringValue(string(alertRule.EventSettings))
	config.IntegrationSettings = types.StringValue(string(alertRule.IntegrationSettings))
	config.CreatedAt = types.StringValue(alertRule.CreatedAt)
	config.UpdatedAt = types.StringValue(alertRule.UpdatedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
