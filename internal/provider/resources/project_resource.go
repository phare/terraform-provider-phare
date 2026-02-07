package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                = &projectResource{}
	_ resource.ResourceWithConfigure   = &projectResource{}
	_ resource.ResourceWithImportState = &projectResource{}
)

// ProjectResourceSchema defines the schema for the project resource
func ProjectResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Date of creation",
				MarkdownDescription: "Date of creation",
			},
			"id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "Project ID",
				MarkdownDescription: "Project ID",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"members": schema.ListAttribute{
				ElementType:         types.Int64Type,
				Required:            true,
				Description:         "List of team member IDs (1-100 members)",
				MarkdownDescription: "List of team member IDs (1-100 members)",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Project name (1-25 characters)",
				MarkdownDescription: "Project name (1-25 characters)",
			},
			"settings": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"incident_merging_time_window": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Time window for incident merging in minutes (5, 10, 30, 60, 120, or 180). Defaults to 60.",
						MarkdownDescription: "Time window for incident merging in minutes (5, 10, 30, 60, 120, or 180). Defaults to 60.",
						Default:             int64default.StaticInt64(60),
						Validators: []validator.Int64{
							int64validator.OneOf(5, 10, 30, 60, 120, 180),
						},
					},
					"use_incident_ai": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Whether to generate AI summaries for incidents. Defaults to true.",
						MarkdownDescription: "Whether to generate AI summaries for incidents. Defaults to true.",
						Default:             booldefault.StaticBool(true),
					},
					"use_incident_merging": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Whether to use incident merging. Defaults to true.",
						MarkdownDescription: "Whether to use incident merging. Defaults to true.",
						Default:             booldefault.StaticBool(true),
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "Project settings",
				MarkdownDescription: "Project settings",
			},
			"slug": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Project slug (auto-generated from name)",
				MarkdownDescription: "Project slug (auto-generated from name)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"updated_at": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Date of last update",
				MarkdownDescription: "Date of last update",
			},
		},
	}
}

// ProjectModel represents the project resource model
type ProjectModel struct {
	CreatedAt types.String `tfsdk:"created_at"`
	Id        types.Int64  `tfsdk:"id"`
	Members   types.List   `tfsdk:"members"`
	Name      types.String `tfsdk:"name"`
	Settings  types.Object `tfsdk:"settings"`
	Slug      types.String `tfsdk:"slug"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

// SettingsAttributeTypes defines the attribute types for the settings object
type SettingsAttributeTypes struct {
	IncidentMergingTimeWindow types.Int64 `tfsdk:"incident_merging_time_window"`
	UseIncidentAi             types.Bool  `tfsdk:"use_incident_ai"`
	UseIncidentMerging        types.Bool  `tfsdk:"use_incident_merging"`
}

// NewProjectResource returns a new project resource.
func NewProjectResource() resource.Resource {
	return &projectResource{}
}

// projectResource is the resource implementation.
type projectResource struct {
	client *client.Client
}

func (r *projectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_project"
}

func (r *projectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceSchema := ProjectResourceSchema(ctx)

	// Add description to the resource
	resourceSchema.Description = "Manages a Phare project. Projects are used to organize and group monitoring resources."
	resourceSchema.MarkdownDescription = "Manages a Phare project. Projects are used to organize and group monitoring resources."

	resp.Schema = resourceSchema
}

func (r *projectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.client = apiClient
}

// validateAPIKeyScope checks if the API key is organization-scoped.
// Project resources can only be managed with organization-scoped API keys (starting with pha_org_).
func (r *projectResource) validateAPIKeyScope(diagnostics *diag.Diagnostics, operation string) {
	_, apiToken, _ := r.client.GetConfig()

	// Project-scoped keys do NOT start with "pha_org_"
	if !strings.HasPrefix(apiToken, "pha_org_") {
		diagnostics.AddError(
			"Invalid API Key for Project Resource",
			fmt.Sprintf("Cannot %s project resources with a project-scoped API key. "+
				"Use an organization-scoped API key instead.", operation),
		)
	}
}

func (r *projectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate API key scope - only org-scoped keys can create projects
	r.validateAPIKeyScope(&resp.Diagnostics, "create")
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert members List to []int64
	var members []int64
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract settings from nested object, using defaults if not specified
	// Defaults: use_incident_ai=true, use_incident_merging=true, incident_merging_time_window=60
	useAI := true
	useMerging := true
	timeWindow := int64(60)

	if !plan.Settings.IsNull() && !plan.Settings.IsUnknown() {
		// Create a temporary struct to unmarshal the settings object
		settingsStruct := struct {
			IncidentMergingTimeWindow types.Int64 `tfsdk:"incident_merging_time_window"`
			UseIncidentAi             types.Bool  `tfsdk:"use_incident_ai"`
			UseIncidentMerging        types.Bool  `tfsdk:"use_incident_merging"`
		}{}

		diags := plan.Settings.As(ctx, &settingsStruct, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Extract individual settings values
		if !settingsStruct.IncidentMergingTimeWindow.IsNull() {
			timeWindow = settingsStruct.IncidentMergingTimeWindow.ValueInt64()
		}
		if !settingsStruct.UseIncidentAi.IsNull() {
			useAI = settingsStruct.UseIncidentAi.ValueBool()
		}
		if !settingsStruct.UseIncidentMerging.IsNull() {
			useMerging = settingsStruct.UseIncidentMerging.ValueBool()
		}
	}

	settings := &client.ProjectSettings{
		UseIncidentAI:             &useAI,
		UseIncidentMerging:        &useMerging,
		IncidentMergingTimeWindow: &timeWindow,
	}

	// Build API request from plan
	apiReq := &client.ProjectRequest{
		Name:     plan.Name.ValueString(),
		Members:  members,
		Settings: settings,
	}

	// Call API to create project
	apiResp, err := r.client.CreateProject(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Project",
			"Could not create project: "+err.Error(),
		)
		return
	}

	// Map API response to state
	r.updateStateFromResponse(ctx, &plan, apiResp, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read project
	apiResp, err := r.client.GetProject(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Project",
			"Could not read project ID "+state.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	r.updateStateFromResponse(ctx, &state, apiResp, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *projectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ProjectModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert members List to []int64
	var members []int64
	resp.Diagnostics.Append(plan.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract settings from nested object, using defaults if not specified
	// Defaults: use_incident_ai=true, use_incident_merging=true, incident_merging_time_window=60
	useAI := true
	useMerging := true
	timeWindow := int64(60)

	if !plan.Settings.IsNull() && !plan.Settings.IsUnknown() {
		// Create a temporary struct to unmarshal the settings object
		settingsStruct := struct {
			IncidentMergingTimeWindow types.Int64 `tfsdk:"incident_merging_time_window"`
			UseIncidentAi             types.Bool  `tfsdk:"use_incident_ai"`
			UseIncidentMerging        types.Bool  `tfsdk:"use_incident_merging"`
		}{}

		diags := plan.Settings.As(ctx, &settingsStruct, basetypes.ObjectAsOptions{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Extract individual settings values
		if !settingsStruct.IncidentMergingTimeWindow.IsNull() {
			timeWindow = settingsStruct.IncidentMergingTimeWindow.ValueInt64()
		}
		if !settingsStruct.UseIncidentAi.IsNull() {
			useAI = settingsStruct.UseIncidentAi.ValueBool()
		}
		if !settingsStruct.UseIncidentMerging.IsNull() {
			useMerging = settingsStruct.UseIncidentMerging.ValueBool()
		}
	}

	settings := &client.ProjectSettings{
		UseIncidentAI:             &useAI,
		UseIncidentMerging:        &useMerging,
		IncidentMergingTimeWindow: &timeWindow,
	}

	// Build API request
	apiReq := &client.ProjectRequest{
		Name:     plan.Name.ValueString(),
		Members:  members,
		Settings: settings,
	}

	// Call API to update project
	apiResp, err := r.client.UpdateProject(ctx, plan.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Project",
			"Could not update project ID "+plan.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	r.updateStateFromResponse(ctx, &plan, apiResp, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *projectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ProjectModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate API key scope - only org-scoped keys can delete projects
	r.validateAPIKeyScope(&resp.Diagnostics, "delete")
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete project
	err := r.client.DeleteProject(ctx, state.Id.ValueInt64())
	if err != nil {
		// Ignore 404 errors - resource already deleted
		if !client.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Error Deleting Project",
				"Could not delete project ID "+state.Id.String()+": "+err.Error(),
			)
			return
		}
	}
}

func (r *projectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse ID from import string
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Project ID must be a valid integer: "+err.Error(),
		)
		return
	}

	// Validate API key scope - only org-scoped keys can import projects
	r.validateAPIKeyScope(&resp.Diagnostics, "import")
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}

// Helper function to update state from API response
func (r *projectResource) updateStateFromResponse(ctx context.Context, state *ProjectModel, apiResp *client.ProjectResponse, diags *diag.Diagnostics) {
	state.Id = types.Int64Value(apiResp.ID)
	state.Slug = types.StringValue(apiResp.Slug)
	state.Name = types.StringValue(apiResp.Name)
	state.CreatedAt = types.StringValue(apiResp.CreatedAt)
	state.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Convert members to List
	membersElements := make([]attr.Value, len(apiResp.Members))
	for i, member := range apiResp.Members {
		membersElements[i] = types.Int64Value(member)
	}
	membersList, listDiags := types.ListValue(types.Int64Type, membersElements)
	diags.Append(listDiags...)
	if diags.HasError() {
		return
	}
	state.Members = membersList

	// Map settings as nested object
	if apiResp.Settings != nil {
		settingsAttrs := map[string]attr.Value{
			"use_incident_ai":              types.BoolNull(),
			"use_incident_merging":         types.BoolNull(),
			"incident_merging_time_window": types.Int64Null(),
		}

		if apiResp.Settings.UseIncidentAI != nil {
			settingsAttrs["use_incident_ai"] = types.BoolValue(*apiResp.Settings.UseIncidentAI)
		}
		if apiResp.Settings.UseIncidentMerging != nil {
			settingsAttrs["use_incident_merging"] = types.BoolValue(*apiResp.Settings.UseIncidentMerging)
		}
		if apiResp.Settings.IncidentMergingTimeWindow != nil {
			settingsAttrs["incident_merging_time_window"] = types.Int64Value(*apiResp.Settings.IncidentMergingTimeWindow)
		}

		settingsObj, objDiags := types.ObjectValue(
			map[string]attr.Type{
				"use_incident_ai":              types.BoolType,
				"use_incident_merging":         types.BoolType,
				"incident_merging_time_window": types.Int64Type,
			},
			settingsAttrs,
		)
		diags.Append(objDiags...)
		if diags.HasError() {
			return
		}
		state.Settings = settingsObj
	} else {
		state.Settings = types.ObjectNull(map[string]attr.Type{
			"use_incident_ai":              types.BoolType,
			"use_incident_merging":         types.BoolType,
			"incident_merging_time_window": types.Int64Type,
		})
	}
}
