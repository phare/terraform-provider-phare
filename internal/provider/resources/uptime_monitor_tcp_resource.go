package resources

import (
	"context"
	"strconv"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// SHARED CODE - BASE MODELS AND SCHEMAS FOR ALL UPTIME MONITOR TYPES

// UptimeMonitorBaseModel represents the common fields for all uptime monitor types
// This is embedded in both HTTP and TCP monitor models
type UptimeMonitorBaseModel struct {
	CreatedAt             types.String  `tfsdk:"created_at"`
	Id                    types.Int64   `tfsdk:"id"`
	IncidentConfirmations types.Int64   `tfsdk:"incident_confirmations"`
	Interval              types.Int64   `tfsdk:"interval"`
	Name                  types.String  `tfsdk:"name"`
	Paused                types.Bool    `tfsdk:"paused"`
	ProjectId             types.Int64   `tfsdk:"project_id"`
	RecoveryConfirmations types.Int64   `tfsdk:"recovery_confirmations"`
	Regions               types.List    `tfsdk:"regions"`
	Status                types.String  `tfsdk:"status"`
	Timeout               types.Int64   `tfsdk:"timeout"`
	UpdatedAt             types.String  `tfsdk:"updated_at"`
	ProjectScope          types.Dynamic `tfsdk:"project_scope"`
}

// UptimeMonitorBaseResourceSchema defines the common schema attributes for all uptime monitor resources
func UptimeMonitorBaseResourceSchema(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"created_at": schema.StringAttribute{
			Computed:            true,
			Description:         "Date of creation",
			MarkdownDescription: "Date of creation",
		},
		"id": schema.Int64Attribute{
			Computed:            true,
			Description:         "Monitor ID",
			MarkdownDescription: "Monitor ID",
		},
		"incident_confirmations": schema.Int64Attribute{
			Required:            true,
			Description:         "Number of uninterrupted failed checks required to create an incident (1-5)",
			MarkdownDescription: "Number of uninterrupted failed checks required to create an incident (1-5)",
		},
		"interval": schema.Int64Attribute{
			Required:            true,
			Description:         "Monitoring interval in seconds (30, 60, 120, 180, 300, 600, 900, 1800, 3600)",
			MarkdownDescription: "Monitoring interval in seconds (30, 60, 120, 180, 300, 600, 900, 1800, 3600)",
		},
		"name": schema.StringAttribute{
			Required:            true,
			Description:         "Monitor name",
			MarkdownDescription: "Monitor name",
		},
		"paused": schema.BoolAttribute{
			Optional:            true,
			Computed:            true,
			Description:         "Whether the monitor is paused",
			MarkdownDescription: "Whether the monitor is paused",
		},
		"project_id": schema.Int64Attribute{
			Computed:            true,
			Description:         "Parent project ID",
			MarkdownDescription: "Parent project ID",
		},
		"recovery_confirmations": schema.Int64Attribute{
			Required:            true,
			Description:         "Number of uninterrupted successful checks required to resolve an incident (1-5)",
			MarkdownDescription: "Number of uninterrupted successful checks required to resolve an incident (1-5)",
		},
		"regions": schema.ListAttribute{
			ElementType:         types.StringType,
			Required:            true,
			Description:         "Regions to monitor from",
			MarkdownDescription: "Regions to monitor from",
		},
		"status": schema.StringAttribute{
			Computed:            true,
			Description:         "Monitor status",
			MarkdownDescription: "Monitor status",
		},
		"timeout": schema.Int64Attribute{
			Required:            true,
			Description:         "Monitoring timeout in milliseconds",
			MarkdownDescription: "Monitoring timeout in milliseconds",
		},
		"updated_at": schema.StringAttribute{
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
	}
}

// AddCommonValidators adds the common validators to the schema
func AddCommonValidators(ctx context.Context, resourceSchema *schema.Schema) {
	// Add validators for enum fields
	if attr, ok := resourceSchema.Attributes["interval"].(schema.Int64Attribute); ok {
		attr.Validators = []validator.Int64{
			int64validator.OneOf(30, 60, 120, 180, 300, 600, 900, 1800, 3600),
		}
		resourceSchema.Attributes["interval"] = attr
	}

	if attr, ok := resourceSchema.Attributes["timeout"].(schema.Int64Attribute); ok {
		attr.Validators = []validator.Int64{
			int64validator.OneOf(1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 15000, 20000, 25000, 30000),
		}
		resourceSchema.Attributes["timeout"] = attr
	}

	if attr, ok := resourceSchema.Attributes["incident_confirmations"].(schema.Int64Attribute); ok {
		attr.Validators = []validator.Int64{
			int64validator.OneOf(1, 2, 3, 4, 5),
		}
		resourceSchema.Attributes["incident_confirmations"] = attr
	}

	if attr, ok := resourceSchema.Attributes["recovery_confirmations"].(schema.Int64Attribute); ok {
		attr.Validators = []validator.Int64{
			int64validator.OneOf(1, 2, 3, 4, 5),
		}
		resourceSchema.Attributes["recovery_confirmations"] = attr
	}
}

// TcpRequestModel represents the TCP request configuration
type TcpRequestModel struct {
	Connection    types.String `tfsdk:"connection"`
	Host          types.String `tfsdk:"host"`
	Port          types.Int64  `tfsdk:"port"`
	TlsSkipVerify types.Bool   `tfsdk:"tls_skip_verify"`
}

// Import the base model from the HTTP resource file
// This allows us to share the common base model between HTTP and TCP monitors

// UptimeMonitorTcpModel represents the main model for TCP uptime monitor
// It embeds the base model from HTTP resource and adds TCP-specific fields
type UptimeMonitorTcpModel struct {
	UptimeMonitorBaseModel
	Request TcpRequestModel `tfsdk:"request"`
}

// UptimeMonitorTcpResourceSchema defines the schema for the TCP uptime monitor resource
func UptimeMonitorTcpResourceSchema(ctx context.Context) schema.Schema {
	// Start with the base schema
	baseAttributes := UptimeMonitorBaseResourceSchema(ctx)

	// Add TCP-specific attributes
	baseAttributes["request"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"connection": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Connection type (plain or tls, defaults to plain)",
				MarkdownDescription: "Connection type (plain or tls, defaults to plain)",
				Default:             stringdefault.StaticString("tcp"),
				Validators: []validator.String{
					stringvalidator.OneOf("plain", "tls"),
				},
			},
			"host": schema.StringAttribute{
				Required:            true,
				Description:         "Hostname or IP address",
				MarkdownDescription: "Hostname or IP address",
			},
			"port": schema.Int64Attribute{
				Required:            true,
				Description:         "Port number",
				MarkdownDescription: "Port number",
			},
			"tls_skip_verify": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Skip TLS certificate verification (default: false)",
				MarkdownDescription: "Skip TLS certificate verification (default: false)",
				Default:             booldefault.StaticBool(false),
			},
		},
		Required:            true,
		Description:         "TCP request configuration",
		MarkdownDescription: "TCP request configuration",
	}

	return schema.Schema{
		Attributes: baseAttributes,
	}
}

var (
	_ resource.Resource                = &uptimeMonitorTcpResource{}
	_ resource.ResourceWithConfigure   = &uptimeMonitorTcpResource{}
	_ resource.ResourceWithImportState = &uptimeMonitorTcpResource{}
	_ resource.ResourceWithModifyPlan  = &uptimeMonitorTcpResource{}
)

// NewUptimeMonitorTcpResource returns a new TCP uptime monitor resource.
func NewUptimeMonitorTcpResource() resource.Resource {
	return &uptimeMonitorTcpResource{}
}

// uptimeMonitorTcpResource is the resource implementation.
type uptimeMonitorTcpResource struct {
	helpers.ResourceBase
}

// uptimeMonitorTcpModel is now the same as UptimeMonitorTcpModel since we've moved everything
type uptimeMonitorTcpModel = UptimeMonitorTcpModel

func (r *uptimeMonitorTcpResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor_tcp"
}

func (r *uptimeMonitorTcpResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceSchema := UptimeMonitorTcpResourceSchema(ctx)

	// Add description to the resource
	resourceSchema.Description = "Manages a TCP uptime monitor in Phare. Monitors service availability using TCP connections."
	resourceSchema.MarkdownDescription = "Manages a TCP uptime monitor in Phare. Monitors service availability using TCP connections."

	// Add common validators
	AddCommonValidators(ctx, &resourceSchema)

	resp.Schema = resourceSchema
}

// Configure is provided by helpers.ResourceBase

func (r *uptimeMonitorTcpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip validation if resource is being destroyed
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan uptimeMonitorTcpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project scope configuration at plan time
	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_monitor_tcp", &resp.Diagnostics)
}

// getScopedClient is provided by helpers.ResourceBase

// Helper function to convert Terraform TCP request model to client request config
func tcpRequestModelToClientConfig(ctx context.Context, request TcpRequestModel) (client.MonitorRequestConfig, error) {
	config := client.MonitorRequestConfig{}

	// Extract request attributes
	if !request.Host.IsNull() && !request.Host.IsUnknown() {
		h := request.Host.ValueString()
		config.Host = &h
	}

	if !request.Port.IsNull() && !request.Port.IsUnknown() {
		// Convert int64 to string for the client
		portStr := strconv.FormatInt(request.Port.ValueInt64(), 10)
		config.Port = &portStr
	}

	if !request.Connection.IsNull() && !request.Connection.IsUnknown() {
		c := request.Connection.ValueString()
		config.Connection = &c
	}

	if !request.TlsSkipVerify.IsNull() && !request.TlsSkipVerify.IsUnknown() {
		b := request.TlsSkipVerify.ValueBool()
		config.TLSSkipVerify = &b
	}

	return config, nil
}

// Helper function to convert client request config to Terraform TCP request model
func clientConfigToTcpRequestModel(ctx context.Context, config client.MonitorRequestConfig) (TcpRequestModel, error) {
	request := TcpRequestModel{}

	if config.Host != nil {
		request.Host = types.StringValue(*config.Host)
	} else {
		request.Host = types.StringNull()
	}

	if config.Port != nil {
		// Convert string port to int64
		if portInt, err := strconv.ParseInt(*config.Port, 10, 64); err == nil {
			request.Port = types.Int64Value(portInt)
		} else {
			request.Port = types.Int64Null()
		}
	} else {
		request.Port = types.Int64Null()
	}

	if config.Connection != nil {
		request.Connection = types.StringValue(*config.Connection)
	} else {
		request.Connection = types.StringNull()
	}

	if config.TLSSkipVerify != nil {
		request.TlsSkipVerify = types.BoolValue(*config.TLSSkipVerify)
	} else {
		request.TlsSkipVerify = types.BoolNull()
	}

	return request, nil
}

func (r *uptimeMonitorTcpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan uptimeMonitorTcpModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_tcp", &resp.Diagnostics)
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
	reqConfig, err := tcpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "tcp", // Always TCP for this resource
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		SuccessAssertions:     nil, // TCP monitors don't have success assertions
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
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request back to model
	requestModel, err := clientConfigToTcpRequestModel(ctx, apiResp.Request)
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

func (r *uptimeMonitorTcpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state uptimeMonitorTcpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_tcp", &resp.Diagnostics)
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
	state.Status = types.StringValue(apiResp.Status)
	state.Paused = types.BoolValue(apiResp.Paused)
	state.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request
	requestModel, err := clientConfigToTcpRequestModel(ctx, apiResp.Request)
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

func (r *uptimeMonitorTcpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan uptimeMonitorTcpModel
	var state uptimeMonitorTcpModel

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
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_tcp", &resp.Diagnostics)
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
	reqConfig, err := tcpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "tcp", // Always TCP for this resource
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		SuccessAssertions:     nil, // TCP monitors don't have success assertions
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
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)

	// Convert request back to model
	requestModel, err := clientConfigToTcpRequestModel(ctx, apiResp.Request)
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
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)
	plan.CreatedAt = types.StringValue(apiResp.CreatedAt)
	plan.UpdatedAt = types.StringValue(apiResp.UpdatedAt)

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorTcpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uptimeMonitorTcpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_tcp", &resp.Diagnostics)
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

func (r *uptimeMonitorTcpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
