package resources

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// IcmpRequestModel represents the ICMP request configuration.
type IcmpRequestModel struct {
	Host types.String `tfsdk:"host"`
}

// UptimeMonitorIcmpModel represents the model for the ICMP uptime monitor resource.
type UptimeMonitorIcmpModel struct {
	UptimeMonitorBaseModel
	Request *IcmpRequestModel `tfsdk:"request"`
}

// UptimeMonitorIcmpResourceSchema defines the schema for the ICMP uptime monitor resource.
func UptimeMonitorIcmpResourceSchema(ctx context.Context) schema.Schema {
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
							helpers.TrimmedLengthBetween(1, 255),
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

type uptimeMonitorIcmpResource struct {
	helpers.ResourceBase
}

type uptimeMonitorIcmpModel = UptimeMonitorIcmpModel

func (r *uptimeMonitorIcmpResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor_icmp"
}

func (r *uptimeMonitorIcmpResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceSchema := UptimeMonitorIcmpResourceSchema(ctx)
	resourceSchema.Description = "Manages an ICMP uptime monitor in Phare. Monitors service availability using ICMP ping."
	resourceSchema.MarkdownDescription = "Manages an ICMP uptime monitor in Phare. Monitors service availability using ICMP ping."

	AddCommonValidators(ctx, &resourceSchema)
	resp.Schema = resourceSchema
}

func (r *uptimeMonitorIcmpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan uptimeMonitorIcmpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	helpers.CheckProjectScopeRequiresReplace(ctx, req, resp)

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

	if plan.Request == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("request"),
			"Missing Request Configuration",
			"The 'request' block is required for ICMP uptime monitors.",
		)
	} else if !plan.Request.Host.IsNull() && !plan.Request.Host.IsUnknown() {
		trimmedLen := utf8.RuneCountInString(strings.TrimSpace(plan.Request.Host.ValueString()))
		if trimmedLen < 1 || trimmedLen > 255 {
			resp.Diagnostics.AddAttributeError(
				path.Root("request").AtName("host"),
				"Invalid Host Length",
				"Host must be between 1 and 255 characters after trimming whitespace.",
			)
		}
	}

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

func (r *uptimeMonitorIcmpResource) buildAPIRequest(ctx context.Context, plan *uptimeMonitorIcmpModel, diags *diag.Diagnostics) (*client.MonitorRequest, bool) {
	var regions []string
	diags.Append(plan.Regions.ElementsAs(ctx, &regions, false)...)
	if diags.HasError() {
		return nil, false
	}

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return nil, false
		}
	}

	reqConfig, err := icmpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		diags.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return nil, false
	}

	return &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "icmp",
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		RegionThreshold:       plan.RegionThreshold.ValueInt64(),
		Tags:                  tags,
	}, true
}

func updateIcmpModelFromResponse(ctx context.Context, apiResp *client.MonitorResponse, model *UptimeMonitorIcmpModel, diags *diag.Diagnostics) {
	model.Id = types.Int64Value(apiResp.ID)
	model.Name = types.StringValue(apiResp.Name)
	model.Interval = types.Int64Value(apiResp.Interval)
	model.Timeout = types.Int64Value(apiResp.Timeout)
	model.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	model.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	model.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	model.Status = types.StringValue(apiResp.Status)
	model.Paused = types.BoolValue(apiResp.Paused)
	model.ProjectId = types.Int64Value(apiResp.ProjectID)
	model.CreatedAt = types.StringValue(apiResp.CreatedAt)
	model.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	requestModel, err := clientConfigToIcmpRequestModel(ctx, apiResp.Request)
	if err != nil {
		diags.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	model.Request = requestModel

	regionsElements := make([]attr.Value, len(apiResp.Regions))
	for i, region := range apiResp.Regions {
		regionsElements[i] = types.StringValue(region)
	}
	regionsList, listDiags := types.ListValue(types.StringType, regionsElements)
	diags.Append(listDiags...)
	if diags.HasError() {
		return
	}
	model.Regions = regionsList

	model.Tags = helpers.StringSliceToList(apiResp.Tags, diags)
}

func icmpRequestModelToClientConfig(ctx context.Context, request *IcmpRequestModel) (client.MonitorRequestConfig, error) {
	config := client.MonitorRequestConfig{}
	if request == nil {
		return config, nil
	}

	if !request.Host.IsNull() && !request.Host.IsUnknown() {
		if h := strings.TrimSpace(request.Host.ValueString()); h != "" {
			config.Host = &h
		}
	}

	return config, nil
}

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
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, ok := r.buildAPIRequest(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}

	apiResp, err := scopedClient.CreateMonitor(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Monitor", "Could not create monitor: "+err.Error())
		return
	}

	updateIcmpModelFromResponse(ctx, apiResp, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorIcmpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state uptimeMonitorIcmpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := scopedClient.GetMonitor(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Monitor", "Could not read monitor ID "+state.Id.String()+": "+err.Error())
		return
	}

	updateIcmpModelFromResponse(ctx, apiResp, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *uptimeMonitorIcmpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state uptimeMonitorIcmpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !helpers.ValidateNoProjectScopeChange(ctx, req.State, req.Plan, &resp.Diagnostics) {
		return
	}

	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, ok := r.buildAPIRequest(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}

	apiResp, err := scopedClient.UpdateMonitor(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Monitor", "Could not update monitor ID "+state.Id.String()+": "+err.Error())
		return
	}

	updateIcmpModelFromResponse(ctx, apiResp, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorIcmpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uptimeMonitorIcmpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_icmp", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	err := scopedClient.DeleteMonitor(ctx, state.Id.ValueInt64())
	if err != nil && !client.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error deleting monitor", "Could not delete monitor ID "+state.Id.String()+": "+err.Error())
	}
}

func (r *uptimeMonitorIcmpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	helpers.ImportStateWithProjectScope(ctx, req, resp, r.GetClient(), "Monitor", true)
}
