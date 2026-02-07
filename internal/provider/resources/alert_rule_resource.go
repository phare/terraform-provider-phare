package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UptimeAlertRuleResourceSchema defines the schema for the alert rule resource
func UptimeAlertRuleResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Date of creation",
				MarkdownDescription: "Date of creation",
			},
			"event": schema.StringAttribute{
				Required:            true,
				Description:         "Name of the event that will trigger the alert rule",
				MarkdownDescription: "Name of the event that will trigger the alert rule",
			},
			"event_settings": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Additional settings for the event (JSON object as string)",
				MarkdownDescription: "Additional settings for the event (JSON object as string)",
			},
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Alert rule ID",
				MarkdownDescription: "Alert rule ID",
			},
			"integration_id": schema.Int64Attribute{
				Required:            true,
				Description:         "The ID of the integration used to send notifications",
				MarkdownDescription: "The ID of the integration used to send notifications",
			},
			"integration_settings": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Additional settings for the integration (JSON object as string)",
				MarkdownDescription: "Additional settings for the integration (JSON object as string)",
			},
			"project_id": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Description:         "The ID of a project to use as a scope",
				MarkdownDescription: "The ID of a project to use as a scope",
			},
			"rate_limit": schema.Int64Attribute{
				Required:            true,
				Description:         "Minimum time in minutes between alert executions",
				MarkdownDescription: "Minimum time in minutes between alert executions",
			},
			"updated_at": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Date of last update",
				MarkdownDescription: "Date of last update",
			},
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this resource. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
		},
	}
}

// UptimeAlertRuleModel defines the data model for the alert rule resource
type UptimeAlertRuleModel struct {
	CreatedAt           types.String `tfsdk:"created_at"`
	Event               types.String `tfsdk:"event"`
	EventSettings       types.String `tfsdk:"event_settings"`
	Id                  types.Int64  `tfsdk:"id"`
	IntegrationId       types.Int64  `tfsdk:"integration_id"`
	IntegrationSettings types.String `tfsdk:"integration_settings"`
	ProjectId           types.Int64  `tfsdk:"project_id"`
	RateLimit           types.Int64  `tfsdk:"rate_limit"`
	UpdatedAt           types.String `tfsdk:"updated_at"`
}

var (
	_ resource.Resource                = &alertRuleResource{}
	_ resource.ResourceWithConfigure   = &alertRuleResource{}
	_ resource.ResourceWithImportState = &alertRuleResource{}
	_ resource.ResourceWithModifyPlan  = &alertRuleResource{}
)

// NewAlertRuleResource returns a new alert rule resource.
func NewAlertRuleResource() resource.Resource {
	return &alertRuleResource{}
}

// alertRuleResource is the resource implementation.
type alertRuleResource struct {
	helpers.ResourceBase
}

// alertRuleModel extends the generated model with project_scope and scope support.
type alertRuleModel struct {
	UptimeAlertRuleModel
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
	Scope        types.String  `tfsdk:"scope"`
}

func (r *alertRuleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rule"
}

func (r *alertRuleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := UptimeAlertRuleResourceSchema(ctx)

	// Add description to the resource
	generatedSchema.Description = "Manages an alert rule in Phare. Alert rules define conditions that trigger notifications when events occur."
	generatedSchema.MarkdownDescription = "Manages an alert rule in Phare. Alert rules define conditions that trigger notifications when events occur."

	// Add scope attribute
	if generatedSchema.Attributes == nil {
		generatedSchema.Attributes = make(map[string]schema.Attribute)
	}
	generatedSchema.Attributes["scope"] = schema.StringAttribute{
		Description: "Scope of the alert rule. Must be either 'organization' or 'project'. " +
			"When set to 'project', the alert rule is scoped to the project specified by project_scope or the provider-level project. " +
			"When set to 'organization', the alert rule applies to the entire organization.",
		Required: true,
		Validators: []validator.String{
			stringvalidator.OneOf("organization", "project"),
		},
	}

	// Add validator for rate_limit enum
	if attr, ok := generatedSchema.Attributes["rate_limit"].(schema.Int64Attribute); ok {
		attr.Validators = []validator.Int64{
			int64validator.OneOf(0, 5, 10, 30, 60, 180, 720, 1440),
		}
		generatedSchema.Attributes["rate_limit"] = attr
	}

	resp.Schema = generatedSchema
}

func (r *alertRuleResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip validation if resource is being destroyed
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan alertRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project scope configuration at plan time
	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
}

// mapAPIResponseToModel maps API response data to the resource model
func (r *alertRuleResource) mapAPIResponseToModel(apiResp *client.AlertRuleResponse, model *alertRuleModel) {
	model.Id = types.Int64Value(apiResp.ID)
	model.Event = types.StringValue(apiResp.Event)
	model.IntegrationId = types.Int64Value(apiResp.IntegrationID)
	model.RateLimit = types.Int64Value(apiResp.RateLimit)

	// Handle optional project_id
	if apiResp.ProjectID != nil {
		model.ProjectId = types.Int64Value(*apiResp.ProjectID)
	} else {
		model.ProjectId = types.Int64Null()
	}

	// Handle optional event_settings
	if len(apiResp.EventSettings) > 0 {
		model.EventSettings = types.StringValue(string(apiResp.EventSettings))
	} else {
		model.EventSettings = types.StringNull()
	}

	// Handle optional integration_settings
	if len(apiResp.IntegrationSettings) > 0 {
		model.IntegrationSettings = types.StringValue(string(apiResp.IntegrationSettings))
	} else {
		model.IntegrationSettings = types.StringNull()
	}

	model.CreatedAt = types.StringValue(apiResp.CreatedAt)
	model.UpdatedAt = types.StringValue(apiResp.UpdatedAt)
}

// buildAPIRequestFromModel builds an API request from the resource model
func (r *alertRuleResource) buildAPIRequestFromModel(model alertRuleModel) (*client.AlertRuleRequest, diag.Diagnostics) {
	var diags diag.Diagnostics

	apiReq := &client.AlertRuleRequest{
		Event:         model.Event.ValueString(),
		IntegrationID: model.IntegrationId.ValueInt64(),
		RateLimit:     model.RateLimit.ValueInt64(),
	}

	// Set scope from the model
	if !model.Scope.IsNull() && !model.Scope.IsUnknown() {
		scope := model.Scope.ValueString()
		apiReq.Scope = &scope
	}

	// Handle event_settings JSON
	if !model.EventSettings.IsNull() && !model.EventSettings.IsUnknown() {
		eventSettingsStr := model.EventSettings.ValueString()
		if eventSettingsStr != "" {
			// Validate JSON format
			if !json.Valid([]byte(eventSettingsStr)) {
				diags.AddError(
					"Invalid event_settings JSON",
					"event_settings must be a valid JSON object: "+eventSettingsStr,
				)
				return nil, diags
			}
			apiReq.EventSettings = []byte(eventSettingsStr)
		}
	}

	// Handle integration_settings JSON
	if !model.IntegrationSettings.IsNull() && !model.IntegrationSettings.IsUnknown() {
		integrationSettingsStr := model.IntegrationSettings.ValueString()
		if integrationSettingsStr != "" {
			// Validate JSON format
			if !json.Valid([]byte(integrationSettingsStr)) {
				diags.AddError(
					"Invalid integration_settings JSON",
					"integration_settings must be a valid JSON object: "+integrationSettingsStr,
				)
				return nil, diags
			}
			apiReq.IntegrationSettings = []byte(integrationSettingsStr)
		}
	}

	return apiReq, diags
}

func (r *alertRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertRuleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request from plan
	apiReq, reqDiags := r.buildAPIRequestFromModel(plan)
	resp.Diagnostics.Append(reqDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to create alert rule
	apiResp, err := scopedClient.CreateAlertRule(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating alert rule",
			"Could not create alert rule: "+err.Error(),
		)
		return
	}

	// Map API response to state
	r.mapAPIResponseToModel(apiResp, &plan)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertRuleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read alert rule
	apiResp, err := scopedClient.GetAlertRule(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error reading alert rule",
			"Could not read alert rule ID "+fmt.Sprintf("%d", state.Id.ValueInt64())+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	r.mapAPIResponseToModel(apiResp, &state)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertRuleModel
	var state alertRuleModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read current state to get the ID
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build API request from plan
	apiReq, reqDiags := r.buildAPIRequestFromModel(plan)
	resp.Diagnostics.Append(reqDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to update alert rule using ID from current state
	apiResp, err := scopedClient.UpdateAlertRule(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error updating alert rule",
			"Could not update alert rule ID "+fmt.Sprintf("%d", state.Id.ValueInt64())+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	r.mapAPIResponseToModel(apiResp, &plan)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertRuleModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_alert_rule", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete alert rule
	err := scopedClient.DeleteAlertRule(ctx, state.Id.ValueInt64())
	if err != nil {
		if !client.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Error deleting alert rule",
				"Could not delete alert rule ID "+fmt.Sprintf("%d", state.Id.ValueInt64())+": "+err.Error(),
			)
			return
		}
	}
}

func (r *alertRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse ID from import string
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Alert rule ID must be a valid integer: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
