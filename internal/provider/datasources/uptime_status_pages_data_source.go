package datasources

import (
	"context"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeStatusPagesDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeStatusPagesDataSource{}
)

// NewUptimeStatusPagesDataSource returns a new uptime status pages data source.
func NewUptimeStatusPagesDataSource() datasource.DataSource {
	return &uptimeStatusPagesDataSource{}
}

// uptimeStatusPagesDataSource is the data source implementation.
type uptimeStatusPagesDataSource struct {
	helpers.DataSourceBase
}

// uptimeStatusPagesDataSourceModel describes the data source data model.
type uptimeStatusPagesDataSourceModel struct {
	StatusPages  []statusPageModel `tfsdk:"status_pages"`
	ProjectScope types.Dynamic     `tfsdk:"project_scope"`
}

func (d *uptimeStatusPagesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_status_pages"
}

func (d *uptimeStatusPagesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of status pages.",
		Attributes: map[string]schema.Attribute{
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
			"status_pages": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of status pages",
				NestedObject: schema.NestedAttributeObject{
					Attributes: statusPageSchemaAttributes(),
				},
			},
		},
	}
}

func (d *uptimeStatusPagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeStatusPagesDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_status_pages", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch all status pages (first 100 for MVP)
	statusPages, err := scopedClient.ListStatusPages(ctx, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Status Pages",
			err.Error(),
		)
		return
	}

	// Map API response to state using shared helper
	for _, page := range statusPages {
		pageState := mapStatusPageToModel(ctx, page, resp)
		if resp.Diagnostics.HasError() {
			return
		}
		config.StatusPages = append(config.StatusPages, pageState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
