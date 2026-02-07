package datasources

import (
	"context"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeMonitorsDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeMonitorsDataSource{}
)

// NewUptimeMonitorsDataSource returns a new uptime monitors data source.
func NewUptimeMonitorsDataSource() datasource.DataSource {
	return &uptimeMonitorsDataSource{}
}

// uptimeMonitorsDataSource is the data source implementation.
type uptimeMonitorsDataSource struct {
	helpers.DataSourceBase
}

// uptimeMonitorsDataSourceModel describes the data source data model.
type uptimeMonitorsDataSourceModel struct {
	Monitors     []monitorModel `tfsdk:"monitors"`
	ProjectScope types.Dynamic  `tfsdk:"project_scope"`
}

func (d *uptimeMonitorsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitors"
}

func (d *uptimeMonitorsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of uptime monitors.",
		Attributes: map[string]schema.Attribute{
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
			"monitors": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of uptime monitors",
				NestedObject: schema.NestedAttributeObject{
					Attributes: monitorSchemaAttributes(),
				},
			},
		},
	}
}

func (d *uptimeMonitorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeMonitorsDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_monitors", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all monitors (first 100 for MVP)
	monitors, err := scopedClient.ListMonitors(ctx, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Monitors",
			err.Error(),
		)
		return
	}

	// Map API response to state using shared helper
	for _, monitor := range monitors {
		monitorState := mapMonitorToModel(ctx, monitor, resp)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Monitors = append(config.Monitors, monitorState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
