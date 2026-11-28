package datasources

import (
	"context"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeHeartbeatsDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeHeartbeatsDataSource{}
)

// NewUptimeHeartbeatsDataSource returns a new uptime heartbeats data source.
func NewUptimeHeartbeatsDataSource() datasource.DataSource {
	return &uptimeHeartbeatsDataSource{}
}

// uptimeHeartbeatsDataSource is the data source implementation.
type uptimeHeartbeatsDataSource struct {
	helpers.DataSourceBase
}

// uptimeHeartbeatsDataSourceModel describes the data source data model.
type uptimeHeartbeatsDataSourceModel struct {
	Tags         types.List       `tfsdk:"tags"`
	Heartbeats   []heartbeatModel `tfsdk:"heartbeats"`
	ProjectScope types.Dynamic    `tfsdk:"project_scope"`
}

func (d *uptimeHeartbeatsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_heartbeats"
}

func (d *uptimeHeartbeatsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of uptime heartbeats, optionally filtered by tags.",
		Attributes: map[string]schema.Attribute{
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
			"tags": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Description:         "List of tags to filter heartbeats, only the heartbeats having all the tags are returned. If not specified, returns all heartbeats (first page, up to 100 heartbeats).",
				MarkdownDescription: "List of tags to filter heartbeats, only the heartbeats having all the tags are returned. If not specified, returns all heartbeats (first page, up to 100 heartbeats).",
				Validators:          helpers.TagListValidators(),
			},
			"heartbeats": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of uptime heartbeats",
				NestedObject: schema.NestedAttributeObject{
					Attributes: heartbeatSchemaAttributes(),
				},
			},
		},
	}
}

func (d *uptimeHeartbeatsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeHeartbeatsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_heartbeats", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract tags filter (nil when not set)
	var tags []string
	if !config.Tags.IsNull() && len(config.Tags.Elements()) > 0 {
		resp.Diagnostics.Append(config.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	heartbeats, err := scopedClient.ListHeartbeats(ctx, 1, 100, tags)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Heartbeats",
			err.Error(),
		)
		return
	}

	config.Heartbeats = make([]heartbeatModel, 0, len(heartbeats))
	for _, heartbeat := range heartbeats {
		config.Heartbeats = append(config.Heartbeats, mapHeartbeatToModel(heartbeat, &resp.Diagnostics))
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
