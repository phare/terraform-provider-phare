package datasources

import (
	"context"

	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &projectsDataSource{}
	_ datasource.DataSourceWithConfigure = &projectsDataSource{}
)

// NewProjectsDataSource returns a new projects data source.
func NewProjectsDataSource() datasource.DataSource {
	return &projectsDataSource{}
}

// projectsDataSource is the data source implementation.
type projectsDataSource struct {
	helpers.DataSourceBase
}

// projectsDataSourceModel describes the data source data model.
type projectsDataSourceModel struct {
	Projects []projectModel `tfsdk:"projects"`
}

func (d *projectsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_projects"
}

func (d *projectsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a list of projects. Requires an organization-scoped API key.",
		Attributes: map[string]schema.Attribute{
			"projects": schema.ListNestedAttribute{
				Computed:    true,
				Description: "List of projects",
				NestedObject: schema.NestedAttributeObject{
					Attributes: projectSchemaAttributes(),
				},
			},
		},
	}
}

func (d *projectsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state projectsDataSourceModel

	// Fetch all projects (first 100 for MVP)
	projects, err := d.GetClient().ListProjects(ctx, 1, 100)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Projects",
			"This data source requires an organization-scoped API key. "+
				"Error: "+err.Error(),
		)
		return
	}

	// Map API response to state
	for _, project := range projects {
		projectState := projectModel{
			ID:        types.Int64Value(project.ID),
			Slug:      types.StringValue(project.Slug),
			Name:      types.StringValue(project.Name),
			CreatedAt: types.StringValue(project.CreatedAt),
			UpdatedAt: types.StringValue(project.UpdatedAt),
		}

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
		projectState.Members = membersList

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
			projectState.Settings = settingsObj
		} else {
			projectState.Settings = types.ObjectNull(map[string]attr.Type{
				"use_incident_ai":              types.BoolType,
				"use_incident_merging":         types.BoolType,
				"incident_merging_time_window": types.Int64Type,
			})
		}

		state.Projects = append(state.Projects, projectState)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
