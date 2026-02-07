package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &integrationDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationDataSource{}
)

// NewIntegrationDataSource returns a new integration data source.
func NewIntegrationDataSource() datasource.DataSource {
	return &integrationDataSource{}
}

// integrationDataSource is the data source implementation.
type integrationDataSource struct {
	helpers.DataSourceBase
}

// integrationModel describes the common integration data model used by both
// the single integration and integrations list data sources.
type integrationModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Paused    types.Bool   `tfsdk:"paused"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// integrationSchemaAttributes returns the common schema attributes for integrations
// Note: id and name are intentionally omitted as they have different configurations
// in single vs list data sources
func integrationSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"paused": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the integration is paused",
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

// integrationDataSourceModel describes the data source data model.
type integrationDataSourceModel struct {
	integrationModel
	App types.String `tfsdk:"app"`
}

func (d *integrationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (d *integrationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a single integration by app and either ID or name. Exactly one of 'id' or 'name' must be specified.",
		Attributes: map[string]schema.Attribute{
			"app": schema.StringAttribute{
				Required:    true,
				Description: "The app key to filter integrations. Valid values: sms, discord, email, ilert, ntfy, outgoing_webhook, pushover, slack, telegram",
				Validators: []validator.String{
					stringvalidator.OneOf("sms", "discord", "email", "ilert", "ntfy", "outgoing_webhook", "pushover", "slack", "telegram"),
				},
			},
			"id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "The ID of the integration to fetch. Exactly one of 'id' or 'name' must be specified.",
			},
			"name": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The name of the integration to fetch. Exactly one of 'id' or 'name' must be specified.",
			},
		},
	}

	// Add the common integration attributes
	for key, attr := range integrationSchemaAttributes() {
		resp.Schema.Attributes[key] = attr
	}
}

func (d *integrationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config integrationDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that exactly one of ID or name is provided
	hasID := !config.ID.IsNull() && !config.ID.IsUnknown()
	hasName := !config.Name.IsNull() && !config.Name.IsUnknown()

	if !helpers.ValidateExactlyOneOf(&resp.Diagnostics, hasID, hasName, "integration") {
		return
	}

	app := config.App.ValueString()
	var integration *client.IntegrationResponse
	var err error

	// Fetch integration by ID or name
	if hasID {
		integrationID := config.ID.ValueInt64()
		integration, err = d.GetClient().GetIntegration(ctx, app, integrationID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Integration",
				fmt.Sprintf("Could not read integration ID %d for app %s: %s", integrationID, app, err.Error()),
			)
			return
		}
	} else {
		// Fetch by name
		integrationName := config.Name.ValueString()
		integration, err = d.GetClient().GetIntegrationByName(ctx, app, integrationName)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Read Integration",
				fmt.Sprintf("Could not read integration with name %q for app %s: %s", integrationName, app, err.Error()),
			)
			return
		}
	}

	// Map API response to state
	config.ID = types.Int64Value(integration.ID)
	config.Name = types.StringValue(integration.Name)
	config.Paused = types.BoolValue(integration.Paused)
	config.CreatedAt = types.StringValue(integration.CreatedAt)
	config.UpdatedAt = types.StringValue(integration.UpdatedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
