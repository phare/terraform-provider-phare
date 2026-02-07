package datasources

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &projectDataSource{}
	_ datasource.DataSourceWithConfigure = &projectDataSource{}
)

// NewProjectDataSource returns a new project data source.
func NewProjectDataSource() datasource.DataSource {
	return &projectDataSource{}
}

// projectDataSource is the data source implementation.
type projectDataSource struct {
	helpers.DataSourceBase
}

// projectModel describes the common project data model used by both
// the single project and projects list data sources.
type projectModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Slug      types.String `tfsdk:"slug"`
	Name      types.String `tfsdk:"name"`
	Members   types.List   `tfsdk:"members"`
	Settings  types.Object `tfsdk:"settings"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// projectSchemaAttributes returns the common schema attributes for projects
func projectSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Project ID",
		},
		"slug": schema.StringAttribute{
			Computed:    true,
			Description: "Project slug",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Project name",
		},
		"members": schema.ListAttribute{
			ElementType: types.Int64Type,
			Computed:    true,
			Description: "List of team member IDs",
		},
		"settings": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Project settings",
			Attributes: map[string]schema.Attribute{
				"use_incident_ai": schema.BoolAttribute{
					Computed:    true,
					Description: "Whether AI summaries are enabled for incidents",
				},
				"use_incident_merging": schema.BoolAttribute{
					Computed:    true,
					Description: "Whether incident merging is enabled",
				},
				"incident_merging_time_window": schema.Int64Attribute{
					Computed:    true,
					Description: "Time window for incident merging in minutes",
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

// projectDataSourceModel describes the data source data model.
type projectDataSourceModel struct {
	projectModel
}

func (d *projectDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (d *projectDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific project by ID or slug. You must specify either 'id' or 'slug' (not both).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Project ID. Must specify either 'id' or 'slug'.",
			},
			"slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Project slug. Must specify either 'id' or 'slug'.",
			},
		},
	}

	// Add the common project attributes (skip id and slug as they're already defined above)
	for key, attr := range projectSchemaAttributes() {
		if key != "id" && key != "slug" {
			resp.Schema.Attributes[key] = attr
		}
	}
}

func (d *projectDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config projectDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate that either id or slug is provided (not both, not neither)
	hasID := !config.ID.IsNull() && !config.ID.IsUnknown()
	hasSlug := !config.Slug.IsNull() && !config.Slug.IsUnknown()

	if !helpers.ValidateExactlyOneOf(&resp.Diagnostics, hasID, hasSlug, "project") {
		return
	}

	// Determine the project ID to fetch
	var projectID int64
	if hasID {
		projectID = config.ID.ValueInt64()
	} else {
		// If slug is provided, we need to look it up
		// List all projects and find the one with matching slug
		projects, err := d.GetClient().ListProjects(ctx, 1, 100)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to List Projects",
				"Error listing projects to find slug: "+err.Error(),
			)
			return
		}

		slugToFind := config.Slug.ValueString()
		found := false
		for _, p := range projects {
			if p.Slug == slugToFind {
				projectID = p.ID
				found = true
				break
			}
		}

		if !found {
			resp.Diagnostics.AddError(
				"Project Not Found",
				fmt.Sprintf("No project found with slug '%s'", slugToFind),
			)
			return
		}
	}

	// Fetch the project by ID
	project, err := d.GetClient().GetProject(ctx, projectID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Project",
			fmt.Sprintf("Error reading project ID %d: %s", projectID, err.Error()),
		)
		return
	}

	// Map API response to state
	config.ID = types.Int64Value(project.ID)
	config.Slug = types.StringValue(project.Slug)
	config.Name = types.StringValue(project.Name)
	config.CreatedAt = types.StringValue(project.CreatedAt)
	config.UpdatedAt = types.StringValue(project.UpdatedAt)

	// Convert members to List
	var membersList types.List
	if len(project.Members) == 0 {
		membersList = types.ListNull(types.Int64Type)
	} else {
		elements := make([]attr.Value, len(project.Members))
		for i, value := range project.Members {
			elements[i] = types.Int64Value(value)
		}
		var diags diag.Diagnostics
		membersList, diags = types.ListValue(types.Int64Type, elements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}
	config.Members = membersList

	// Map settings as nested object
	if project.Settings != nil {
		settingsAttrs := map[string]attr.Value{
			"use_incident_ai":              types.BoolNull(),
			"use_incident_merging":         types.BoolNull(),
			"incident_merging_time_window": types.Int64Null(),
		}

		if project.Settings.UseIncidentAI != nil {
			settingsAttrs["use_incident_ai"] = types.BoolValue(*project.Settings.UseIncidentAI)
		}
		if project.Settings.UseIncidentMerging != nil {
			settingsAttrs["use_incident_merging"] = types.BoolValue(*project.Settings.UseIncidentMerging)
		}
		if project.Settings.IncidentMergingTimeWindow != nil {
			settingsAttrs["incident_merging_time_window"] = types.Int64Value(*project.Settings.IncidentMergingTimeWindow)
		}

		settingsObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"use_incident_ai":              types.BoolType,
				"use_incident_merging":         types.BoolType,
				"incident_merging_time_window": types.Int64Type,
			},
			settingsAttrs,
		)
		resp.Diagnostics.Append(objDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
		config.Settings = settingsObj
	} else {
		config.Settings = types.ObjectNull(map[string]attr.Type{
			"use_incident_ai":              types.BoolType,
			"use_incident_merging":         types.BoolType,
			"incident_merging_time_window": types.Int64Type,
		})
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
