package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &integrationsDataSource{}
	_ datasource.DataSourceWithConfigure = &integrationsDataSource{}
)

// NewIntegrationsDataSource returns a new integrations data source.
func NewIntegrationsDataSource() datasource.DataSource {
	return &integrationsDataSource{}
}

// integrationsDataSource is the data source implementation.
type integrationsDataSource struct {
	helpers.DataSourceBase
}

// integrationsDataSourceModel describes the data source data model.
type integrationsDataSourceModel struct {
	App          types.String       `tfsdk:"app"`
	Integrations []integrationModel `tfsdk:"integrations"`
}

func (d *integrationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

func (d *integrationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of integrations for a specific app.",
		Attributes: map[string]schema.Attribute{
			"app": schema.StringAttribute{
				Required:    true,
				Description: "The app key to filter integrations. Valid values: sms, discord, email, ilert, ntfy, outgoing_webhook, pushover, slack, telegram",
				Validators: []validator.String{
					stringvalidator.OneOf("sms", "discord", "email", "ilert", "ntfy", "outgoing_webhook", "pushover", "slack", "telegram"),
				},
			},
			"integrations": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of integrations for the specified app",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Computed:    true,
							Description: "Integration ID",
						},
						"name": schema.StringAttribute{
							Computed:    true,
							Description: "Integration name",
						},
					},
				},
			},
		},
	}

	// Add the common integration attributes from the shared function
	for key, attr := range integrationSchemaAttributes() {
		resp.Schema.Attributes["integrations"].(schema.ListNestedAttribute).NestedObject.Attributes[key] = attr
	}
}

func (d *integrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config integrationsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	app := config.App.ValueString()

	// Fetch all integrations for the app (first 100 for MVP)
	integrations, err := d.GetClient().ListIntegrations(ctx, app, "", 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Integrations",
			fmt.Sprintf("Error reading integrations for app %s: %s", app, err.Error()),
		)
		return
	}

	// Map API response to state
	for _, integration := range integrations {
		integrationState := integrationModel{
			ID:        types.Int64Value(integration.ID),
			Name:      types.StringValue(integration.Name),
			Paused:    types.BoolValue(integration.Paused),
			CreatedAt: types.StringValue(integration.CreatedAt),
			UpdatedAt: types.StringValue(integration.UpdatedAt),
		}

		config.Integrations = append(config.Integrations, integrationState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
