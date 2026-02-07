package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeStatusPageDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeStatusPageDataSource{}
)

// NewUptimeStatusPageDataSource returns a new status page data source.
func NewUptimeStatusPageDataSource() datasource.DataSource {
	return &uptimeStatusPageDataSource{}
}

// uptimeStatusPageDataSource is the data source implementation.
type uptimeStatusPageDataSource struct {
	helpers.DataSourceBase
}

// statusPageModel describes the common status page data model used by both
// the single status page and status pages list data sources.
type statusPageModel struct {
	ID                  types.Int64  `tfsdk:"id"`
	ProjectID           types.Int64  `tfsdk:"project_id"`
	Name                types.String `tfsdk:"name"`
	Subdomain           types.String `tfsdk:"subdomain"`
	Domain              types.String `tfsdk:"domain"`
	Title               types.String `tfsdk:"title"`
	Description         types.String `tfsdk:"description"`
	SearchEngineIndexed types.Bool   `tfsdk:"search_engine_indexed"`
	WebsiteURL          types.String `tfsdk:"website_url"`
	Timeframe           types.Int64  `tfsdk:"timeframe"`
	Colors              types.Object `tfsdk:"colors"`
	Components          types.List   `tfsdk:"components"`
	CreatedAt           types.String `tfsdk:"created_at"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

// mapStatusPageToModel maps an API status page response to the shared status page model
func mapStatusPageToModel(ctx context.Context, page *client.StatusPageResponse, resp *datasource.ReadResponse) statusPageModel {
	model := statusPageModel{
		ID:                  types.Int64Value(page.ID),
		ProjectID:           types.Int64Value(page.ProjectID),
		Name:                types.StringValue(page.Name),
		Title:               types.StringValue(page.Title),
		Description:         types.StringValue(page.Description),
		SearchEngineIndexed: types.BoolValue(page.SearchEngineIndexed),
		WebsiteURL:          types.StringValue(page.WebsiteURL),
		CreatedAt:           types.StringValue(page.CreatedAt),
		UpdatedAt:           types.StringValue(page.UpdatedAt),
	}

	if page.Subdomain != nil {
		model.Subdomain = types.StringValue(*page.Subdomain)
	} else {
		model.Subdomain = types.StringNull()
	}
	if page.Domain != nil {
		model.Domain = types.StringValue(*page.Domain)
	} else {
		model.Domain = types.StringNull()
	}
	if page.Timeframe != nil {
		model.Timeframe = types.Int64Value(*page.Timeframe)
	} else {
		model.Timeframe = types.Int64Null()
	}

	// Map colors
	if page.Colors != nil {
		colorsObj, diags := types.ObjectValue(
			map[string]attr.Type{
				"operational":          types.StringType,
				"degraded_performance": types.StringType,
				"partial_outage":       types.StringType,
				"major_outage":         types.StringType,
				"maintenance":          types.StringType,
				"empty":                types.StringType,
			},
			map[string]attr.Value{
				"operational":          types.StringValue(page.Colors.Operational),
				"degraded_performance": types.StringValue(page.Colors.DegradedPerformance),
				"partial_outage":       types.StringValue(page.Colors.PartialOutage),
				"major_outage":         types.StringValue(page.Colors.MajorOutage),
				"maintenance":          types.StringValue(page.Colors.Maintenance),
				"empty":                types.StringValue(page.Colors.Empty),
			},
		)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return statusPageModel{}
		}
		model.Colors = colorsObj
	} else {
		model.Colors = types.ObjectNull(map[string]attr.Type{
			"operational":          types.StringType,
			"degraded_performance": types.StringType,
			"partial_outage":       types.StringType,
			"major_outage":         types.StringType,
			"maintenance":          types.StringType,
			"empty":                types.StringType,
		})
	}

	// Map components
	if len(page.Components) > 0 {
		componentType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"componentable_type": types.StringType,
				"componentable_id":   types.Int64Type,
			},
		}

		componentElements := make([]attr.Value, len(page.Components))
		for i, comp := range page.Components {
			compObj, diags := types.ObjectValue(
				componentType.AttrTypes,
				map[string]attr.Value{
					"componentable_type": types.StringValue(comp.ComponentableType),
					"componentable_id":   types.Int64Value(comp.ComponentableID),
				},
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return statusPageModel{}
			}
			componentElements[i] = compObj
		}

		componentsList, diags := types.ListValue(componentType, componentElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return statusPageModel{}
		}
		model.Components = componentsList
	} else {
		model.Components = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"componentable_type": types.StringType,
				"componentable_id":   types.Int64Type,
			},
		})
	}

	return model
}

// statusPageSchemaAttributes returns the common schema attributes for status pages
func statusPageSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Status page ID",
		},
		"project_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Parent project ID",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Status page name",
		},
		"subdomain": schema.StringAttribute{
			Computed:    true,
			Description: "Subdomain for the status page",
		},
		"domain": schema.StringAttribute{
			Computed:    true,
			Description: "Custom domain for the status page",
		},
		"title": schema.StringAttribute{
			Computed:    true,
			Description: "Status page HTML title",
		},
		"description": schema.StringAttribute{
			Computed:    true,
			Description: "Status page HTML description",
		},
		"search_engine_indexed": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether search engines can index the page",
		},
		"website_url": schema.StringAttribute{
			Computed:    true,
			Description: "URL to redirect users from the status page",
		},
		"timeframe": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of days of status/incident history to display",
		},
		"colors": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Colors to customize the status page appearance",
			Attributes: map[string]schema.Attribute{
				"operational": schema.StringAttribute{
					Computed:    true,
					Description: "Color for operational status",
				},
				"degraded_performance": schema.StringAttribute{
					Computed:    true,
					Description: "Color for degraded performance status",
				},
				"partial_outage": schema.StringAttribute{
					Computed:    true,
					Description: "Color for partial outage status",
				},
				"major_outage": schema.StringAttribute{
					Computed:    true,
					Description: "Color for major outage status",
				},
				"maintenance": schema.StringAttribute{
					Computed:    true,
					Description: "Color for maintenance status",
				},
				"empty": schema.StringAttribute{
					Computed:    true,
					Description: "Color for empty/no data status",
				},
			},
		},
		"components": schema.ListNestedAttribute{
			Computed:    true,
			Description: "List of components (monitors) shown on the status page",
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"componentable_type": schema.StringAttribute{
						Computed:    true,
						Description: "Type of component",
					},
					"componentable_id": schema.Int64Attribute{
						Computed:    true,
						Description: "ID of the component",
					},
				},
			},
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

// uptimeStatusPageDataSourceModel describes the data source data model.
type uptimeStatusPageDataSourceModel struct {
	statusPageModel
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
}

func (d *uptimeStatusPageDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_status_page"
}

func (d *uptimeStatusPageDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific status page by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Status page ID",
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

	// Add the common status page attributes
	for key, attr := range statusPageSchemaAttributes() {
		resp.Schema.Attributes[key] = attr
	}
}

func (d *uptimeStatusPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeStatusPageDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the status page by ID
	statusPage, err := scopedClient.GetStatusPage(ctx, config.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Status Page",
			fmt.Sprintf("Error reading status page ID %d: %s", config.ID.ValueInt64(), err.Error()),
		)
		return
	}

	// Map API response to state using shared helper
	mappedModel := mapStatusPageToModel(ctx, statusPage, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	config.statusPageModel = mappedModel

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
