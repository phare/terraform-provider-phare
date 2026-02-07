package resources

import (
	"context"
	"strconv"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ColorsModel represents the color configuration for status pages
type ColorsModel struct {
	DegradedPerformance types.String `tfsdk:"degraded_performance"`
	Empty               types.String `tfsdk:"empty"`
	Maintenance         types.String `tfsdk:"maintenance"`
	MajorOutage         types.String `tfsdk:"major_outage"`
	Operational         types.String `tfsdk:"operational"`
	PartialOutage       types.String `tfsdk:"partial_outage"`
}

// ComponentModel represents a component in the status page
type ComponentModel struct {
	ComponentableType types.String `tfsdk:"componentable_type"`
	ComponentableID   types.Int64  `tfsdk:"componentable_id"`
}

// UptimeStatusPageModel defines the data model for the status page resource
type UptimeStatusPageModel struct {
	Colors               ColorsModel   `tfsdk:"colors"`
	Components           types.List    `tfsdk:"components"`
	CreatedAt            types.String  `tfsdk:"created_at"`
	Description          types.String  `tfsdk:"description"`
	Domain               types.String  `tfsdk:"domain"`
	Id                   types.Int64   `tfsdk:"id"`
	Name                 types.String  `tfsdk:"name"`
	ProjectId            types.Int64   `tfsdk:"project_id"`
	SearchEngineIndexed  types.Bool    `tfsdk:"search_engine_indexed"`
	Subdomain            types.String  `tfsdk:"subdomain"`
	SubscriptionChannels types.List    `tfsdk:"subscription_channels"`
	Timeframe            types.Int64   `tfsdk:"timeframe"`
	Title                types.String  `tfsdk:"title"`
	UpdatedAt            types.String  `tfsdk:"updated_at"`
	WebsiteUrl           types.String  `tfsdk:"website_url"`
	ProjectScope         types.Dynamic `tfsdk:"project_scope"`
}

// UptimeStatusPageResourceSchema defines the schema for the status page resource
func UptimeStatusPageResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"colors": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"degraded_performance": schema.StringAttribute{
						Required:            true,
						Description:         "Color for degraded performance status (e.g., #fbbf24)",
						MarkdownDescription: "Color for degraded performance status (e.g., #fbbf24)",
					},
					"empty": schema.StringAttribute{
						Required:            true,
						Description:         "Color for empty/no data status (e.g., #d3d3d3)",
						MarkdownDescription: "Color for empty/no data status (e.g., #d3d3d3)",
					},
					"maintenance": schema.StringAttribute{
						Required:            true,
						Description:         "Color for maintenance status (e.g., #6366f1)",
						MarkdownDescription: "Color for maintenance status (e.g., #6366f1)",
					},
					"major_outage": schema.StringAttribute{
						Required:            true,
						Description:         "Color for major outage status (e.g., #ef4444)",
						MarkdownDescription: "Color for major outage status (e.g., #ef4444)",
					},
					"operational": schema.StringAttribute{
						Required:            true,
						Description:         "Color for operational status (e.g., #16a34a)",
						MarkdownDescription: "Color for operational status (e.g., #16a34a)",
					},
					"partial_outage": schema.StringAttribute{
						Required:            true,
						Description:         "Color for partial outage status (e.g., #f59e0b)",
						MarkdownDescription: "Color for partial outage status (e.g., #f59e0b)",
					},
				},
				Required:            true,
				Description:         "Colors to customize the status page appearance",
				MarkdownDescription: "Colors to customize the status page appearance",
			},
			"components": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"componentable_id": schema.Int64Attribute{
							Required:            true,
							Description:         "ID of the monitor to display on the status page",
							MarkdownDescription: "ID of the monitor to display on the status page",
						},
						"componentable_type": schema.StringAttribute{
							Required:            true,
							Description:         "Type of component (uptime/monitor)",
							MarkdownDescription: "Type of component (uptime/monitor)",
							Validators: []validator.String{
								stringvalidator.OneOf("uptime/monitor"),
							},
						},
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "List of components (monitors) to show on the status page",
				MarkdownDescription: "List of components (monitors) to show on the status page",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				Description:         "Date of creation",
				MarkdownDescription: "Date of creation",
			},
			"description": schema.StringAttribute{
				Required:            true,
				Description:         "Status page description",
				MarkdownDescription: "Status page description",
			},
			"domain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Custom domain for the status page",
				MarkdownDescription: "Custom domain for the status page",
			},
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Status page ID",
				MarkdownDescription: "Status page ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Status page name",
				MarkdownDescription: "Status page name",
			},
			"project_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Parent project ID",
				MarkdownDescription: "Parent project ID",
			},
			"search_engine_indexed": schema.BoolAttribute{
				Required:            true,
				Description:         "Whether search engines can index the page",
				MarkdownDescription: "Whether search engines can index the page",
			},
			"subdomain": schema.StringAttribute{
				Required:            true,
				Description:         "Subdomain for the status page",
				MarkdownDescription: "Subdomain for the status page",
			},
			"subscription_channels": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				Description:         "Subscription channels available (rss, atom)",
				MarkdownDescription: "Subscription channels available (rss, atom)",
			},
			"timeframe": schema.Int64Attribute{
				Required:            true,
				Description:         "Number of days of status/incident history to display (30, 60, or 90)",
				MarkdownDescription: "Number of days of status/incident history to display (30, 60, or 90)",
			},
			"title": schema.StringAttribute{
				Required:            true,
				Description:         "Status page title",
				MarkdownDescription: "Status page title",
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				Description:         "Date of last update",
				MarkdownDescription: "Date of last update",
			},
			"website_url": schema.StringAttribute{
				Required:            true,
				Description:         "URL to redirect users from the status page",
				MarkdownDescription: "URL to redirect users from the status page",
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

// ColorsModelAttrTypes defines the attribute types for ColorsModel
var ColorsModelAttrTypes = map[string]attr.Type{
	"degraded_performance": types.StringType,
	"empty":                types.StringType,
	"maintenance":          types.StringType,
	"major_outage":         types.StringType,
	"operational":          types.StringType,
	"partial_outage":       types.StringType,
}

// ComponentModelAttrTypes defines the attribute types for ComponentModel
var ComponentModelAttrTypes = map[string]attr.Type{
	"componentable_type": types.StringType,
	"componentable_id":   types.Int64Type,
}

var (
	_ resource.Resource                = &uptimeStatusPageResource{}
	_ resource.ResourceWithConfigure   = &uptimeStatusPageResource{}
	_ resource.ResourceWithImportState = &uptimeStatusPageResource{}
	_ resource.ResourceWithModifyPlan  = &uptimeStatusPageResource{}
)

// NewUptimeStatusPageResource returns a new status page resource.
func NewUptimeStatusPageResource() resource.Resource {
	return &uptimeStatusPageResource{}
}

// uptimeStatusPageResource is the resource implementation.
type uptimeStatusPageResource struct {
	helpers.ResourceBase
}

func (r *uptimeStatusPageResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_status_page"
}

func (r *uptimeStatusPageResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	generatedSchema := UptimeStatusPageResourceSchema(ctx)

	// Add description to the resource
	generatedSchema.Description = "Manages a status page in Phare. Status pages provide visibility into system status and incidents."
	generatedSchema.MarkdownDescription = "Manages a status page in Phare. Status pages provide visibility into system status and incidents."

	// Make timeframe required with default value and add validator
	generatedSchema.Attributes["timeframe"] = schema.Int64Attribute{
		Required:    true,
		Description: "Number of days of status/incident history to display (30, 60, or 90)",
		Validators: []validator.Int64{
			int64validator.OneOf(30, 60, 90),
		},
	}

	// Add validator for subscription_channels
	if attr, ok := generatedSchema.Attributes["subscription_channels"].(schema.ListAttribute); ok {
		attr.Validators = []validator.List{
			listvalidator.ValueStringsAre(stringvalidator.OneOf("rss", "atom")),
		}
		generatedSchema.Attributes["subscription_channels"] = attr
	}

	resp.Schema = generatedSchema
}

// Configure is provided by helpers.ResourceBase

func (r *uptimeStatusPageResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip validation if resource is being destroyed
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan UptimeStatusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project scope configuration at plan time
	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
}

// getScopedClient is provided by helpers.ResourceBase

// Helper function to convert Terraform colors model to client colors config
func colorsModelToClientConfig(colors ColorsModel) *client.StatusPageColors {
	return &client.StatusPageColors{
		Operational:         colors.Operational.ValueString(),
		DegradedPerformance: colors.DegradedPerformance.ValueString(),
		PartialOutage:       colors.PartialOutage.ValueString(),
		MajorOutage:         colors.MajorOutage.ValueString(),
		Maintenance:         colors.Maintenance.ValueString(),
		Empty:               colors.Empty.ValueString(),
	}
}

// Helper function to convert client colors config to Terraform colors model
func clientConfigToColorsModel(config *client.StatusPageColors) ColorsModel {
	return ColorsModel{
		Operational:         types.StringValue(config.Operational),
		DegradedPerformance: types.StringValue(config.DegradedPerformance),
		PartialOutage:       types.StringValue(config.PartialOutage),
		MajorOutage:         types.StringValue(config.MajorOutage),
		Maintenance:         types.StringValue(config.Maintenance),
		Empty:               types.StringValue(config.Empty),
	}
}

func (r *uptimeStatusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UptimeStatusPageModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build components list from plan
	var components []client.StatusPageComponent
	if !plan.Components.IsNull() && !plan.Components.IsUnknown() {
		var planComponents []ComponentModel
		resp.Diagnostics.Append(plan.Components.ElementsAs(ctx, &planComponents, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		components = make([]client.StatusPageComponent, len(planComponents))
		for i, comp := range planComponents {
			components[i] = client.StatusPageComponent{
				ComponentableType: comp.ComponentableType.ValueString(),
				ComponentableID:   comp.ComponentableID.ValueInt64(),
			}
		}
	}

	// Convert colors from plan
	colors := colorsModelToClientConfig(plan.Colors)

	// Convert subscription_channels List to []string
	var subscriptionChannels []string
	if !plan.SubscriptionChannels.IsNull() && !plan.SubscriptionChannels.IsUnknown() {
		resp.Diagnostics.Append(plan.SubscriptionChannels.ElementsAs(ctx, &subscriptionChannels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Build API request from plan
	subdomain := plan.Subdomain.ValueString()
	apiReq := &client.StatusPageRequest{
		Name:                 plan.Name.ValueString(),
		Subdomain:            &subdomain,
		Title:                plan.Title.ValueString(),
		Description:          plan.Description.ValueString(),
		SearchEngineIndexed:  plan.SearchEngineIndexed.ValueBool(),
		WebsiteURL:           plan.WebsiteUrl.ValueString(),
		Colors:               colors,
		Components:           components,
		SubscriptionChannels: subscriptionChannels,
	}

	// Add timeframe (now required)
	timeframe := plan.Timeframe.ValueInt64()
	apiReq.Timeframe = &timeframe

	// Add optional fields
	if !plan.Domain.IsNull() {
		domain := plan.Domain.ValueString()
		apiReq.Domain = &domain
	}

	// Call API to create status page
	apiResp, err := scopedClient.CreateStatusPage(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Status Page",
			"Could not create status page: "+err.Error(),
		)
		return
	}

	// Map API response to state
	plan.Id = types.Int64Value(apiResp.ID)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)
	plan.Name = types.StringValue(apiResp.Name)
	plan.Subdomain = types.StringValue(*apiResp.Subdomain)
	plan.Title = types.StringValue(apiResp.Title)
	plan.Description = types.StringValue(apiResp.Description)
	plan.SearchEngineIndexed = types.BoolValue(apiResp.SearchEngineIndexed)
	plan.WebsiteUrl = types.StringValue(apiResp.WebsiteURL)

	if apiResp.Domain != nil {
		plan.Domain = types.StringValue(*apiResp.Domain)
	} else {
		plan.Domain = types.StringNull()
	}

	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Map timeframe from API response
	if apiResp.Timeframe != nil {
		plan.Timeframe = types.Int64Value(*apiResp.Timeframe)
	}

	// Map colors from API response
	if apiResp.Colors != nil {
		plan.Colors = clientConfigToColorsModel(apiResp.Colors)
	}

	// Map components from API response
	if len(apiResp.Components) > 0 {
		componentModels := make([]ComponentModel, len(apiResp.Components))
		for i, comp := range apiResp.Components {
			componentModels[i] = ComponentModel{
				ComponentableType: types.StringValue(comp.ComponentableType),
				ComponentableID:   types.Int64Value(comp.ComponentableID),
			}
		}

		componentType := types.ObjectType{
			AttrTypes: ComponentModelAttrTypes,
		}

		componentsList, diags := types.ListValueFrom(ctx, componentType, componentModels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Components = componentsList
	} else {
		plan.Components = types.ListNull(types.ObjectType{AttrTypes: ComponentModelAttrTypes})
	}

	// Map subscription_channels from API response
	if len(apiResp.SubscriptionChannels) > 0 {
		channelsElements := make([]attr.Value, len(apiResp.SubscriptionChannels))
		for i, channel := range apiResp.SubscriptionChannels {
			channelsElements[i] = types.StringValue(channel)
		}
		channelsList, diags := types.ListValue(types.StringType, channelsElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.SubscriptionChannels = channelsList
	} else {
		plan.SubscriptionChannels = types.ListNull(types.StringType)
	}

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeStatusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UptimeStatusPageModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read status page
	apiResp, err := scopedClient.GetStatusPage(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Status Page",
			"Could not read status page ID "+state.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	state.ProjectId = types.Int64Value(apiResp.ProjectID)
	state.Name = types.StringValue(apiResp.Name)
	state.Subdomain = types.StringValue(*apiResp.Subdomain)
	state.Title = types.StringValue(apiResp.Title)
	state.Description = types.StringValue(apiResp.Description)
	state.SearchEngineIndexed = types.BoolValue(apiResp.SearchEngineIndexed)
	state.WebsiteUrl = types.StringValue(apiResp.WebsiteURL)

	if apiResp.Domain != nil {
		state.Domain = types.StringValue(*apiResp.Domain)
	} else {
		state.Domain = types.StringNull()
	}

	state.CreatedAt = types.StringValue(apiResp.CreatedAt)
	state.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Map timeframe from API response
	if apiResp.Timeframe != nil {
		state.Timeframe = types.Int64Value(*apiResp.Timeframe)
	}

	// Map colors from API response
	if apiResp.Colors != nil {
		state.Colors = clientConfigToColorsModel(apiResp.Colors)
	}

	// Map components from API response
	if len(apiResp.Components) > 0 {
		componentModels := make([]ComponentModel, len(apiResp.Components))
		for i, comp := range apiResp.Components {
			componentModels[i] = ComponentModel{
				ComponentableType: types.StringValue(comp.ComponentableType),
				ComponentableID:   types.Int64Value(comp.ComponentableID),
			}
		}

		componentType := types.ObjectType{
			AttrTypes: ComponentModelAttrTypes,
		}

		componentsList, diags := types.ListValueFrom(ctx, componentType, componentModels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.Components = componentsList
	} else {
		state.Components = types.ListNull(types.ObjectType{AttrTypes: ComponentModelAttrTypes})
	}

	// Map subscription_channels from API response
	if len(apiResp.SubscriptionChannels) > 0 {
		channelsElements := make([]attr.Value, len(apiResp.SubscriptionChannels))
		for i, channel := range apiResp.SubscriptionChannels {
			channelsElements[i] = types.StringValue(channel)
		}
		channelsList, diags := types.ListValue(types.StringType, channelsElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		state.SubscriptionChannels = channelsList
	} else {
		state.SubscriptionChannels = types.ListNull(types.StringType)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *uptimeStatusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan UptimeStatusPageModel
	var state UptimeStatusPageModel

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
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build components list from plan
	var components []client.StatusPageComponent
	if !plan.Components.IsNull() && !plan.Components.IsUnknown() {
		var planComponents []ComponentModel
		resp.Diagnostics.Append(plan.Components.ElementsAs(ctx, &planComponents, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		components = make([]client.StatusPageComponent, len(planComponents))
		for i, comp := range planComponents {
			components[i] = client.StatusPageComponent{
				ComponentableType: comp.ComponentableType.ValueString(),
				ComponentableID:   comp.ComponentableID.ValueInt64(),
			}
		}
	}

	// Convert colors from plan
	colors := colorsModelToClientConfig(plan.Colors)

	// Convert subscription_channels List to []string
	var subscriptionChannels []string
	if !plan.SubscriptionChannels.IsNull() && !plan.SubscriptionChannels.IsUnknown() {
		resp.Diagnostics.Append(plan.SubscriptionChannels.ElementsAs(ctx, &subscriptionChannels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Build API request
	subdomain := plan.Subdomain.ValueString()
	apiReq := &client.StatusPageRequest{
		Name:                 plan.Name.ValueString(),
		Subdomain:            &subdomain,
		Title:                plan.Title.ValueString(),
		Description:          plan.Description.ValueString(),
		SearchEngineIndexed:  plan.SearchEngineIndexed.ValueBool(),
		WebsiteURL:           plan.WebsiteUrl.ValueString(),
		Colors:               colors,
		Components:           components,
		SubscriptionChannels: subscriptionChannels,
	}

	// Add timeframe (now required)
	timeframe := plan.Timeframe.ValueInt64()
	apiReq.Timeframe = &timeframe

	// Add optional fields
	if !plan.Domain.IsNull() {
		domain := plan.Domain.ValueString()
		apiReq.Domain = &domain
	}

	// Call API to update status page using ID from current state (note: API uses POST not PUT)
	apiResp, err := scopedClient.UpdateStatusPage(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Status Page",
			"Could not update status page ID "+state.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	plan.Id = types.Int64Value(apiResp.ID)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)
	plan.Name = types.StringValue(apiResp.Name)
	plan.Subdomain = types.StringValue(*apiResp.Subdomain)
	plan.Title = types.StringValue(apiResp.Title)
	plan.Description = types.StringValue(apiResp.Description)
	plan.SearchEngineIndexed = types.BoolValue(apiResp.SearchEngineIndexed)
	plan.WebsiteUrl = types.StringValue(apiResp.WebsiteURL)

	if apiResp.Domain != nil {
		plan.Domain = types.StringValue(*apiResp.Domain)
	} else {
		plan.Domain = types.StringNull()
	}

	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Map timeframe from API response
	if apiResp.Timeframe != nil {
		plan.Timeframe = types.Int64Value(*apiResp.Timeframe)
	}

	// Map colors from API response
	if apiResp.Colors != nil {
		plan.Colors = clientConfigToColorsModel(apiResp.Colors)
	}

	// Map components from API response
	if len(apiResp.Components) > 0 {
		componentModels := make([]ComponentModel, len(apiResp.Components))
		for i, comp := range apiResp.Components {
			componentModels[i] = ComponentModel{
				ComponentableType: types.StringValue(comp.ComponentableType),
				ComponentableID:   types.Int64Value(comp.ComponentableID),
			}
		}

		componentType := types.ObjectType{
			AttrTypes: ComponentModelAttrTypes,
		}

		componentsList, diags := types.ListValueFrom(ctx, componentType, componentModels)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.Components = componentsList
	} else {
		plan.Components = types.ListNull(types.ObjectType{AttrTypes: ComponentModelAttrTypes})
	}

	// Map subscription_channels from API response
	if len(apiResp.SubscriptionChannels) > 0 {
		channelsElements := make([]attr.Value, len(apiResp.SubscriptionChannels))
		for i, channel := range apiResp.SubscriptionChannels {
			channelsElements[i] = types.StringValue(channel)
		}
		channelsList, diags := types.ListValue(types.StringType, channelsElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		plan.SubscriptionChannels = channelsList
	} else {
		plan.SubscriptionChannels = types.ListNull(types.StringType)
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeStatusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UptimeStatusPageModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_status_page", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete status page
	err := scopedClient.DeleteStatusPage(ctx, state.Id.ValueInt64())
	if err != nil {
		// Ignore 404 errors - resource already deleted
		if !client.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Error Deleting Status Page",
				"Could not delete status page ID "+state.Id.String()+": "+err.Error(),
			)
			return
		}
	}
}

func (r *uptimeStatusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse ID from import string
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Status page ID must be a valid integer: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
