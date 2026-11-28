package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeHeartbeatDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeHeartbeatDataSource{}
)

// NewUptimeHeartbeatDataSource returns a new heartbeat data source.
func NewUptimeHeartbeatDataSource() datasource.DataSource {
	return &uptimeHeartbeatDataSource{}
}

// uptimeHeartbeatDataSource is the data source implementation.
type uptimeHeartbeatDataSource struct {
	helpers.DataSourceBase
}

// heartbeatModel describes the common heartbeat data model used by both
// the single heartbeat and heartbeats list data sources.
type heartbeatModel struct {
	ID                    types.Int64  `tfsdk:"id"`
	ProjectID             types.Int64  `tfsdk:"project_id"`
	Name                  types.String `tfsdk:"name"`
	DisplayName           types.String `tfsdk:"display_name"`
	Interval              types.Int64  `tfsdk:"interval"`
	Grace                 types.Int64  `tfsdk:"grace"`
	IncidentConfirmations types.Int64  `tfsdk:"incident_confirmations"`
	RecoveryConfirmations types.Int64  `tfsdk:"recovery_confirmations"`
	Tags                  types.Set    `tfsdk:"tags"`
	Status                types.String `tfsdk:"status"`
	Paused                types.Bool   `tfsdk:"paused"`
	PingURL               types.String `tfsdk:"ping_url"`
	LastPingedAt          types.String `tfsdk:"last_pinged_at"`
	CreatedAt             types.String `tfsdk:"created_at"`
	UpdatedAt             types.String `tfsdk:"updated_at"`
}

// mapHeartbeatToModel maps an API heartbeat response to the shared heartbeat model
func mapHeartbeatToModel(heartbeat *client.HeartbeatResponse, diags *diag.Diagnostics) heartbeatModel {
	return heartbeatModel{
		ID:                    types.Int64Value(heartbeat.ID),
		ProjectID:             types.Int64Value(heartbeat.ProjectID),
		Name:                  types.StringValue(heartbeat.Name),
		DisplayName:           types.StringPointerValue(heartbeat.DisplayName),
		Interval:              types.Int64Value(heartbeat.Interval),
		Grace:                 types.Int64Value(heartbeat.Grace),
		IncidentConfirmations: types.Int64Value(heartbeat.IncidentConfirmations),
		RecoveryConfirmations: types.Int64Value(heartbeat.RecoveryConfirmations),
		Tags:                  helpers.StringSliceToSet(heartbeat.Tags, types.SetNull(types.StringType), diags),
		Status:                types.StringValue(heartbeat.Status),
		Paused:                types.BoolValue(heartbeat.Paused),
		PingURL:               types.StringValue(heartbeat.PingURL),
		LastPingedAt:          types.StringPointerValue(heartbeat.LastPingedAt),
		CreatedAt:             types.StringValue(heartbeat.CreatedAt),
		UpdatedAt:             types.StringValue(heartbeat.UpdatedAt),
	}
}

// heartbeatSchemaAttributes returns the common heartbeat schema attributes
func heartbeatSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Heartbeat ID",
		},
		"project_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Parent project ID",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Heartbeat name",
		},
		"display_name": schema.StringAttribute{
			Computed:    true,
			Description: "Public name of the heartbeat on status pages, null when it defaults to the heartbeat name",
		},
		"interval": schema.Int64Attribute{
			Computed:    true,
			Description: "Expected time between two pings, in seconds",
		},
		"grace": schema.Int64Attribute{
			Computed:    true,
			Description: "Time allowed after an expected ping before it counts as missed, in seconds",
		},
		"incident_confirmations": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of uninterrupted missed or failed pings required to create an incident",
		},
		"recovery_confirmations": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of uninterrupted successful pings required to resolve an incident",
		},
		"tags": schema.SetAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "Resource tags",
		},
		"status": schema.StringAttribute{
			Computed:    true,
			Description: "Current heartbeat status (waiting, online, offline or paused)",
		},
		"paused": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the heartbeat is paused",
		},
		"ping_url": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "URL to request with GET, HEAD or POST when the job succeeds, add /start when it begins and /fail to report a failure",
		},
		"last_pinged_at": schema.StringAttribute{
			Computed:    true,
			Description: "Date of the last successful or failed ping with a 1 minute accuracy, null until the first ping",
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

// uptimeHeartbeatDataSourceModel describes the data source data model.
type uptimeHeartbeatDataSourceModel struct {
	heartbeatModel
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
}

func (d *uptimeHeartbeatDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_heartbeat"
}

func (d *uptimeHeartbeatDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific uptime heartbeat by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Heartbeat ID",
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

	for key, attr := range heartbeatSchemaAttributes() {
		if key != "id" {
			resp.Schema.Attributes[key] = attr
		}
	}
}

func (d *uptimeHeartbeatDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeHeartbeatDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	heartbeat, err := scopedClient.GetHeartbeat(ctx, config.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Heartbeat",
			fmt.Sprintf("Error reading heartbeat ID %d: %s", config.ID.ValueInt64(), err.Error()),
		)
		return
	}

	config.heartbeatModel = mapHeartbeatToModel(heartbeat, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
