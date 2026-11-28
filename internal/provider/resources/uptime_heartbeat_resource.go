package resources

import (
	"context"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// UptimeHeartbeatModel represents the model for the uptime heartbeat resource.
type UptimeHeartbeatModel struct {
	CreatedAt             types.String  `tfsdk:"created_at"`
	DisplayName           types.String  `tfsdk:"display_name"`
	Grace                 types.Int64   `tfsdk:"grace"`
	Id                    types.Int64   `tfsdk:"id"`
	IncidentConfirmations types.Int64   `tfsdk:"incident_confirmations"`
	Interval              types.Int64   `tfsdk:"interval"`
	Name                  types.String  `tfsdk:"name"`
	Paused                types.Bool    `tfsdk:"paused"`
	PingUrl               types.String  `tfsdk:"ping_url"`
	ProjectId             types.Int64   `tfsdk:"project_id"`
	RecoveryConfirmations types.Int64   `tfsdk:"recovery_confirmations"`
	Status                types.String  `tfsdk:"status"`
	Tags                  types.Set     `tfsdk:"tags"`
	UpdatedAt             types.String  `tfsdk:"updated_at"`
	ProjectScope          types.Dynamic `tfsdk:"project_scope"`
}

// UptimeHeartbeatResourceSchema defines the schema for the uptime heartbeat resource.
func UptimeHeartbeatResourceSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Description: "Manages an uptime heartbeat in Phare. A heartbeat expects a ping from a cron job or a worker at a regular interval, " +
			"and opens an incident once the missed or failed pings are confirmed.",
		MarkdownDescription: "Manages an uptime heartbeat in Phare. A heartbeat expects a ping from a cron job or a worker at a regular interval, " +
			"and opens an incident once the missed or failed pings are confirmed. The job requests the `ping_url` with GET, HEAD or POST " +
			"when it succeeds, with `/fail` appended when it fails, and can request it with `/start` appended when it begins to measure its run time, " +
			"a started run must then finish within the `grace` period. The heartbeat stays `waiting` without any incident until its first ping.",
		Attributes: map[string]schema.Attribute{
			"created_at": schema.StringAttribute{
				Computed:            true,
				Description:         "Date of creation",
				MarkdownDescription: "Date of creation",
			},
			"display_name": schema.StringAttribute{
				Optional:            true,
				Description:         "Public name of the heartbeat on status pages, defaults to the heartbeat name",
				MarkdownDescription: "Public name of the heartbeat on status pages, defaults to the heartbeat `name`",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
				Validators: []validator.String{
					helpers.TrimmedLengthBetween(2, 45),
					helpers.SafeEntityName('/'),
				},
			},
			"grace": schema.Int64Attribute{
				Required:            true,
				Description:         "Time allowed after an expected ping before it counts as missed, in seconds, a whole number of minutes from 1 minute (60) to 7 days (604800)",
				MarkdownDescription: "Time allowed after an expected ping before it counts as missed, in seconds, a whole number of minutes from 1 minute (`60`) to 7 days (`604800`)",
				Validators: []validator.Int64{
					int64validator.Between(60, 604800),
					helpers.Int64MultipleOf(60),
				},
			},
			"id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Heartbeat ID",
				MarkdownDescription: "Heartbeat ID",
			},
			"incident_confirmations": schema.Int64Attribute{
				Required:            true,
				Description:         "Number of uninterrupted missed or failed pings required to create an incident (1-5)",
				MarkdownDescription: "Number of uninterrupted missed or failed pings required to create an incident (1-5)",
				Validators: []validator.Int64{
					int64validator.OneOf(1, 2, 3, 4, 5),
				},
			},
			"interval": schema.Int64Attribute{
				Required:            true,
				Description:         "Expected time between two pings, in seconds, a whole number of minutes from 1 minute (60) to 31 days (2678400)",
				MarkdownDescription: "Expected time between two pings, in seconds, a whole number of minutes from 1 minute (`60`) to 31 days (`2678400`)",
				Validators: []validator.Int64{
					int64validator.Between(60, 2678400),
					helpers.Int64MultipleOf(60),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Description:         "Heartbeat name, unique in the project",
				MarkdownDescription: "Heartbeat name, unique in the project",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
				Validators: []validator.String{
					helpers.TrimmedLengthBetween(2, 45),
					helpers.SafeEntityName('/'),
				},
			},
			"paused": schema.BoolAttribute{
				Computed:            true,
				Description:         "Whether the heartbeat is paused",
				MarkdownDescription: "Whether the heartbeat is paused",
			},
			"ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				Description:         "URL to request with GET, HEAD or POST when the job succeeds, add /start when it begins and /fail to report a failure",
				MarkdownDescription: "URL to request with GET, HEAD or POST when the job succeeds, add `/start` when it begins and `/fail` to report a failure",
				PlanModifiers: []planmodifier.String{
					// The ping URL stays the same when the heartbeat is updated
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_id": schema.Int64Attribute{
				Computed:            true,
				Description:         "Parent project ID",
				MarkdownDescription: "Parent project ID",
			},
			"recovery_confirmations": schema.Int64Attribute{
				Required:            true,
				Description:         "Number of uninterrupted successful pings required to resolve an incident (1-5)",
				MarkdownDescription: "Number of uninterrupted successful pings required to resolve an incident (1-5)",
				Validators: []validator.Int64{
					int64validator.OneOf(1, 2, 3, 4, 5),
				},
			},
			"status": schema.StringAttribute{
				Computed:            true,
				Description:         "Heartbeat status (waiting, online, offline or paused), waiting until its first ping",
				MarkdownDescription: "Heartbeat status (`waiting`, `online`, `offline` or `paused`), `waiting` until its first ping",
			},
			"tags": schema.SetAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Description:         "Resource tags (max 20, each 1-100 characters, only letters, numbers and . _ : - characters)",
				MarkdownDescription: "Resource tags (max 20, each 1-100 characters, only letters, numbers and . _ : - characters)",
				Validators:          helpers.TagSetValidators(),
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				Description:         "Date of last update",
				MarkdownDescription: "Date of last update",
			},
			"project_scope": helpers.ProjectScopeAttribute(),
		},
	}
}

var (
	_ resource.Resource                = &uptimeHeartbeatResource{}
	_ resource.ResourceWithConfigure   = &uptimeHeartbeatResource{}
	_ resource.ResourceWithImportState = &uptimeHeartbeatResource{}
	_ resource.ResourceWithModifyPlan  = &uptimeHeartbeatResource{}
)

// NewUptimeHeartbeatResource returns a new uptime heartbeat resource.
func NewUptimeHeartbeatResource() resource.Resource {
	return &uptimeHeartbeatResource{}
}

type uptimeHeartbeatResource struct {
	helpers.ResourceBase
}

func (r *uptimeHeartbeatResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_heartbeat"
}

func (r *uptimeHeartbeatResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = UptimeHeartbeatResourceSchema(ctx)
}

func (r *uptimeHeartbeatResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan UptimeHeartbeatModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	helpers.CheckProjectScopeRequiresReplace(ctx, req, resp)
}

func buildHeartbeatAPIRequest(ctx context.Context, plan *UptimeHeartbeatModel, diags *diag.Diagnostics) (*client.HeartbeatRequest, bool) {
	// nil when unset so the API clears the tags
	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		diags.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if diags.HasError() {
			return nil, false
		}
	}

	return &client.HeartbeatRequest{
		Name:                  plan.Name.ValueString(),
		DisplayName:           plan.DisplayName.ValueStringPointer(),
		Interval:              plan.Interval.ValueInt64(),
		Grace:                 plan.Grace.ValueInt64(),
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		Tags:                  tags,
	}, true
}

func updateHeartbeatModelFromResponse(apiResp *client.HeartbeatResponse, model *UptimeHeartbeatModel, diags *diag.Diagnostics) {
	model.Id = types.Int64Value(apiResp.ID)
	model.ProjectId = types.Int64Value(apiResp.ProjectID)
	model.Name = types.StringValue(apiResp.Name)
	model.DisplayName = types.StringPointerValue(apiResp.DisplayName)
	model.Interval = types.Int64Value(apiResp.Interval)
	model.Grace = types.Int64Value(apiResp.Grace)
	model.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	model.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	model.Status = types.StringValue(apiResp.Status)
	model.Paused = types.BoolValue(apiResp.Paused)
	model.PingUrl = types.StringValue(apiResp.PingURL)
	model.CreatedAt = types.StringValue(apiResp.CreatedAt)
	model.UpdatedAt = types.StringValue(apiResp.UpdatedAt)
	model.Tags = helpers.StringSliceToSet(apiResp.Tags, model.Tags, diags)
}

func (r *uptimeHeartbeatResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan UptimeHeartbeatModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, ok := buildHeartbeatAPIRequest(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}

	apiResp, err := scopedClient.CreateHeartbeat(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Creating Heartbeat", "Could not create heartbeat: "+err.Error())
		return
	}

	updateHeartbeatModelFromResponse(apiResp, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeHeartbeatResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state UptimeHeartbeatModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := scopedClient.GetHeartbeat(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error Reading Heartbeat", "Could not read heartbeat ID "+state.Id.String()+": "+err.Error())
		return
	}

	updateHeartbeatModelFromResponse(apiResp, &state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *uptimeHeartbeatResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state UptimeHeartbeatModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !helpers.ValidateNoProjectScopeChange(ctx, req.State, req.Plan, &resp.Diagnostics) {
		return
	}

	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq, ok := buildHeartbeatAPIRequest(ctx, &plan, &resp.Diagnostics)
	if !ok {
		return
	}

	apiResp, err := scopedClient.UpdateHeartbeat(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error Updating Heartbeat", "Could not update heartbeat ID "+state.Id.String()+": "+err.Error())
		return
	}

	updateHeartbeatModelFromResponse(apiResp, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeHeartbeatResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state UptimeHeartbeatModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_heartbeat", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	err := scopedClient.DeleteHeartbeat(ctx, state.Id.ValueInt64())
	if err != nil && !client.IsNotFoundError(err) {
		resp.Diagnostics.AddError("Error Deleting Heartbeat", "Could not delete heartbeat ID "+state.Id.String()+": "+err.Error())
	}
}

func (r *uptimeHeartbeatResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	helpers.ImportStateWithProjectScope(ctx, req, resp, r.GetClient(), "Heartbeat", true)
}
