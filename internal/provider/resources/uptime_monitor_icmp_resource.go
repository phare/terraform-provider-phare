package resources

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IcmpRequestModel represents the ICMP request configuration
type IcmpRequestModel struct {
	Host types.String `tfsdk:"host"`
}

// UptimeMonitorIcmpModel represents the main model for ICMP uptime monitor
// It embeds the base model and adds ICMP-specific fields
type UptimeMonitorIcmpModel struct {
	UptimeMonitorBaseModel
	Request *IcmpRequestModel `tfsdk:"request"`
}

// UptimeMonitorIcmpResourceSchema defines the schema for the ICMP uptime monitor resource
func UptimeMonitorIcmpResourceSchema(ctx context.Context) schema.Schema {
	// Start with the base schema
	baseAttributes := UptimeMonitorBaseResourceSchema(ctx)

	return schema.Schema{
		Attributes: baseAttributes,
		Blocks: map[string]schema.Block{
			"request": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"host": schema.StringAttribute{
						Required:            true,
						Description:         "Hostname or IP address",
						MarkdownDescription: "Hostname or IP address",
						PlanModifiers: []planmodifier.String{
							helpers.TrimString(),
						},
						Validators: []validator.String{
							stringvalidator.LengthBetween(1, 255),
						},
					},
				},
				Description:         "ICMP request configuration",
				MarkdownDescription: "ICMP request configuration",
			},
		},
	}
}

var (
	_ resource.Resource                = &uptimeMonitorIcmpResource{}
	_ resource.ResourceWithConfigure   = &uptimeMonitorIcmpResource{}
	_ resource.ResourceWithImportState = &uptimeMonitorIcmpResource{}
	_ resource.ResourceWithModifyPlan  = &uptimeMonitorIcmpResource{}
)

// NewUptimeMonitorIcmpResource returns a new ICMP uptime monitor resource.
func NewUptimeMonitorIcmpResource() resource.Resource {
	return &uptimeMonitorIcmpResource{}
}

// uptimeMonitorIcmpResource is the resource implementation.
type uptimeMonitorIcmpResource struct {
	helpers.ResourceBase
}

// uptimeMonitorIcmpModel is an alias for UptimeMonitorIcmpModel
type uptimeMonitorIcmpModel = UptimeMonitorIcmpModel

func (r *uptimeMonitorIcmpResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor_icmp"
}

func (r *uptimeMonitorIcmpResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceSchema := UptimeMonitorIcmpResourceSchema(ctx)

	// Add description to the resource
	resourceSchema.Description = "Manages an ICMP uptime monitor in Phare. Monitors service availability using ICMP ping."
	resourceSchema.MarkdownDescription = "Manages an ICMP uptime monitor in Phare. Monitors service availability using ICMP ping."

	// Add common validators
	AddCommonValidators(ctx, &resourceSchema)

	resp.Schema = resourceSchema
}

// Configure is provided by helpers.ResourceBase

func (r *uptimeMonitorIcmpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip validation if resource is being destroyed
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan uptimeMonitorIcmpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project scope configuration at plan time
	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)

	// Validate name length after trimming whitespace
	if !plan.Name.IsNull() && !plan.Name.IsUnknown() {
		trimmedLen := utf8.RuneCountInString(strings.TrimSpace(plan.Name.ValueString()))
		if trimmedLen < 2 || trimmedLen > 45 {
			resp.Diagnostics.AddAttributeError(
				path.Root("name"),
				"Invalid Name Length",
				"Monitor name must be between 2 and 45 characters after trimming whitespace.",
			)
		}
	}

	// Validate request block is provided
	if plan.Request == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("request"),
			"Missing Request Configuration",
			"The 'request' block is required for ICMP uptime monitors.",
		)
	}

	// Validate region_threshold <= len(regions)
	if !plan.RegionThreshold.IsNull() && !plan.RegionThreshold.IsUnknown() && !plan.Regions.IsNull() && !plan.Regions.IsUnknown() {
		regionCount := int64(len(plan.Regions.Elements()))
		if plan.RegionThreshold.ValueInt64() > regionCount {
			resp.Diagnostics.AddError(
				"Invalid region_threshold",
				fmt.Sprintf("region_threshold (%d) must not exceed the number of regions (%d)", plan.RegionThreshold.ValueInt64(), regionCount),
			)
		}
	}
}

// Helper function to convert Terraform ICMP request model to client request config
func icmpRequestModelToClientConfig(ctx context.Context, request *IcmpRequestModel) (client.MonitorRequestConfig, error) {
	config := client.MonitorRequestConfig{}

	if request == nil {
		return config, nil
	}

	// Extract request attributes
	if !request.Host.IsNull() && !request.Host.IsUnknown() {
		h := request.Host.ValueString()
		config.Host = &h
	}

	return config, nil
}

// Helper function to convert client request config to Terraform ICMP request model
func clientConfigToIcmpRequestModel(ctx context.Context, config client.MonitorRequestConfig) (*IcmpRequestModel, error) {
	request := &IcmpRequestModel{}

	if config.Host != nil {
		request.Host = types.StringValue(*config.Host)
	} else {
		request.Host = types.StringNull()
	}

	return request, nil
}

func (r *uptimeMonitorIcmpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan uptimeMonitorIcmpModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert regions List to []string
	var regions []string
	resp.Diagnostics.Append(plan.Regions.ElementsAs(ctx, &regions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert request object
	reqConfig, err := icmpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "icmp", // Always ICMP for this resource
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		RegionThreshold:       plan.RegionThreshold.ValueInt64(),
		SuccessAssertions:     nil, // ICMP monitors don't have success assertions
	}

	// Call API to create monitor
	apiResp, err := scopedClient.CreateMonitor(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating Monitor",
			"Could not create monitor: "+err.Error(),
		)
		return
	}

	// Map API response to state
	plan.Id = types.Int64Value(apiResp.ID)
	plan.Name = types.StringValue(apiResp.Name)
	plan.Interval = types.Int64Value(apiResp.Interval)
	plan.Timeout = types.Int64Value(apiResp.Timeout)
	plan.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	plan.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	plan.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request back to model
	requestModel, err := clientConfigToIcmpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	plan.Request = requestModel

	// Convert regions back to List
	regionsElements := make([]attr.Value, len(apiResp.Regions))
	for i, region := range apiResp.Regions {
		regionsElements[i] = types.StringValue(region)
	}
	regionsList, diags := types.ListValue(types.StringType, regionsElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Regions = regionsList

	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorIcmpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state uptimeMonitorIcmpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to read monitor
	apiResp, err := scopedClient.GetMonitor(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource deleted outside Terraform
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError(
			"Error Reading Monitor",
			"Could not read monitor ID "+state.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	state.Name = types.StringValue(apiResp.Name)
	state.Interval = types.Int64Value(apiResp.Interval)
	state.Timeout = types.Int64Value(apiResp.Timeout)
	state.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	state.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	state.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	state.Status = types.StringValue(apiResp.Status)
	state.Paused = types.BoolValue(apiResp.Paused)
	state.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request
	requestModel, err := clientConfigToIcmpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	state.Request = requestModel

	// Convert regions back to List
	regionsElements := make([]attr.Value, len(apiResp.Regions))
	for i, region := range apiResp.Regions {
		regionsElements[i] = types.StringValue(region)
	}
	regionsList, diags := types.ListValue(types.StringType, regionsElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Regions = regionsList

	state.CreatedAt = types.StringValue(apiResp.CreatedAt)
	state.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *uptimeMonitorIcmpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan uptimeMonitorIcmpModel
	var state uptimeMonitorIcmpModel

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
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert regions List to []string
	var regions []string
	resp.Diagnostics.Append(plan.Regions.ElementsAs(ctx, &regions, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert request object
	reqConfig, err := icmpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "icmp", // Always ICMP for this resource
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		RegionThreshold:       plan.RegionThreshold.ValueInt64(),
		SuccessAssertions:     nil, // ICMP monitors don't have success assertions
	}

	// Call API to update monitor using ID from current state
	apiResp, err := scopedClient.UpdateMonitor(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating Monitor",
			"Could not update monitor ID "+state.Id.String()+": "+err.Error(),
		)
		return
	}

	// Update state from API response
	plan.Name = types.StringValue(apiResp.Name)
	plan.Interval = types.Int64Value(apiResp.Interval)
	plan.Timeout = types.Int64Value(apiResp.Timeout)
	plan.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	plan.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	plan.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)

	// Convert request back to model
	requestModel, err := clientConfigToIcmpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	plan.Request = requestModel

	// Convert regions back to List
	regionsElements := make([]attr.Value, len(apiResp.Regions))
	for i, region := range apiResp.Regions {
		regionsElements[i] = types.StringValue(region)
	}
	regionsList, diags := types.ListValue(types.StringType, regionsElements)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Regions = regionsList

	// Update state from API response
	plan.Id = types.Int64Value(apiResp.ID)
	plan.Name = types.StringValue(apiResp.Name)
	plan.Interval = types.Int64Value(apiResp.Interval)
	plan.Timeout = types.Int64Value(apiResp.Timeout)
	plan.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	plan.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	plan.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)
	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorIcmpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uptimeMonitorIcmpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete monitor
	err := scopedClient.DeleteMonitor(ctx, state.Id.ValueInt64())
	if err != nil {
		// Ignore 404 errors - resource already deleted
		if !client.IsNotFoundError(err) {
			resp.Diagnostics.AddError(
				"Error deleting monitor",
				"Could not delete monitor ID "+state.Id.String()+": "+err.Error(),
			)
			return
		}
	}
}

func (r *uptimeMonitorIcmpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse ID from import string
	id, err := strconv.ParseInt(req.ID, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			"Monitor ID must be a valid integer: "+err.Error(),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)
}
