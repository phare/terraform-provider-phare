package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"
)

// HeaderModel represents a header configuration for HTTP requests
type HeaderModel struct {
	Name  types.String `tfsdk:"name"`
	Value types.String `tfsdk:"value"`
}

// HeaderModelAttrTypes defines the attribute types for HeaderModel
var HeaderModelAttrTypes = map[string]attr.Type{
	"name":  types.StringType,
	"value": types.StringType,
}

// SuccessAssertionModel represents a success assertion for HTTP monitors
type SuccessAssertionModel struct {
	Type     types.String `tfsdk:"type"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
	Selector types.String `tfsdk:"selector"`
}

// SuccessAssertionModelAttrTypes defines the attribute types for SuccessAssertionModel
var SuccessAssertionModelAttrTypes = map[string]attr.Type{
	"type":     types.StringType,
	"operator": types.StringType,
	"value":    types.StringType,
	"selector": types.StringType,
}

// ClientAssertionsToHttpTerraformList converts client success assertions to Terraform HTTP list
func ClientAssertionsToHttpTerraformList(ctx context.Context, assertions []map[string]interface{}) (types.List, error) {
	if len(assertions) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}), nil
	}

	assertionsList := make([]attr.Value, len(assertions))
	for i, a := range assertions {
		assertionAttrs := map[string]attr.Value{
			"type":     types.StringValue(a["type"].(string)),
			"operator": types.StringValue(a["operator"].(string)),
			"value":    types.StringValue(a["value"].(string)),
			"selector": types.StringNull(),
		}
		if selector, ok := a["selector"]; ok && selector != nil {
			assertionAttrs["selector"] = types.StringValue(selector.(string))
		}
		assertionObj, _ := types.ObjectValue(SuccessAssertionModelAttrTypes, assertionAttrs)
		assertionsList[i] = assertionObj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}, assertionsList)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}), fmt.Errorf("error creating list: %v", diags)
	}
	return list, nil
}

// HttpRequestModel represents the HTTP request configuration
type HttpRequestModel struct {
	Body            types.String `tfsdk:"body"`
	FollowRedirects types.Bool   `tfsdk:"follow_redirects"`
	Headers         types.List   `tfsdk:"headers"`
	Method          types.String `tfsdk:"method"`
	TlsSkipVerify   types.Bool   `tfsdk:"tls_skip_verify"`
	Url             types.String `tfsdk:"url"`
	UserAgentSecret types.String `tfsdk:"user_agent_secret"`
}

// UptimeMonitorHttpModel represents the main model for HTTP uptime monitor
// It embeds the base model and adds HTTP-specific fields
type UptimeMonitorHttpModel struct {
	UptimeMonitorBaseModel
	Request           HttpRequestModel `tfsdk:"request"`
	SuccessAssertions types.List       `tfsdk:"success_assertions"`
}

// UptimeMonitorHttpResourceSchema defines the schema for the HTTP uptime monitor resource
func UptimeMonitorHttpResourceSchema(ctx context.Context) schema.Schema {
	// Start with the base schema
	baseAttributes := UptimeMonitorBaseResourceSchema(ctx)

	// Add HTTP-specific attributes
	baseAttributes["request"] = schema.SingleNestedAttribute{
		Attributes: map[string]schema.Attribute{
			"body": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Request body for POST, PUT, PATCH",
				MarkdownDescription: "Request body for POST, PUT, PATCH",
			},
			"follow_redirects": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Follow HTTP redirects (default: true)",
				MarkdownDescription: "Follow HTTP redirects (default: true)",
				Default:             booldefault.StaticBool(true),
			},
			"headers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Required:            true,
							Description:         "Header name",
							MarkdownDescription: "Header name",
						},
						"value": schema.StringAttribute{
							Required:            true,
							Description:         "Header value",
							MarkdownDescription: "Header value",
						},
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "Additional HTTP headers (max 10)",
				MarkdownDescription: "Additional HTTP headers (max 10), [see docs](https://docs.phare.io/uptime/monitors#headers)",
			},
			"method": schema.StringAttribute{
				Required:            true,
				Description:         "HTTP method (GET, POST, PUT, PATCH, HEAD, OPTIONS)",
				MarkdownDescription: "HTTP method (GET, POST, PUT, PATCH, HEAD, OPTIONS)",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
			},
			"tls_skip_verify": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Skip TLS certificate verification (default: false)",
				MarkdownDescription: "Skip TLS certificate verification (default: false)",
				Default:             booldefault.StaticBool(false),
			},
			"url": schema.StringAttribute{
				Required:            true,
				Description:         "URL to monitor",
				MarkdownDescription: "URL to monitor",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
			},
			"user_agent_secret": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Sensitive:           true,
				Description:         "Secret value in User-Agent header for authentication",
				MarkdownDescription: "Secret value in User-Agent header for authentication, [see docs](https://docs.phare.io/uptime/monitors#user-agent)",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
			},
		},
		Required:            true,
		Description:         "HTTP/HTTPS request configuration",
		MarkdownDescription: "HTTP/HTTPS request configuration",
	}

	baseAttributes["success_assertions"] = schema.ListNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"operator": schema.StringAttribute{
					Required:            true,
					Description:         "Operator (in, not_in, equals, not_equals, contains, not_contains)",
					MarkdownDescription: "Operator (in, not_in, equals, not_equals, contains, not_contains)",
					PlanModifiers: []planmodifier.String{
						helpers.TrimString(),
					},
				},
				"selector": schema.StringAttribute{
					Optional:            true,
					Computed:            true,
					Description:         "Selector (header name for response_header type)",
					MarkdownDescription: "Selector (header name for response_header type)",
					PlanModifiers: []planmodifier.String{
						helpers.TrimString(),
					},
				},
				"type": schema.StringAttribute{
					Required:            true,
					Description:         "Assertion type (status_code, response_header, response_body)",
					MarkdownDescription: "Assertion type (status_code, response_header, response_body)",
					PlanModifiers: []planmodifier.String{
						helpers.TrimString(),
					},
				},
				"value": schema.StringAttribute{
					Required:            true,
					Description:         "Value to assert against",
					MarkdownDescription: "Value to assert against",
					PlanModifiers: []planmodifier.String{
						helpers.TrimString(),
					},
				},
			},
		},
		Required:            true,
		Description:         "List of assertions that must be true for the check to be considered successful",
		MarkdownDescription: "List of assertions that must be true for the check to be considered successful, [see docs](https://docs.phare.io/uptime/monitors#success-assertions)",
	}

	return schema.Schema{
		Attributes: baseAttributes,
	}
}

var (
	_ resource.Resource                = &uptimeMonitorHttpResource{}
	_ resource.ResourceWithConfigure   = &uptimeMonitorHttpResource{}
	_ resource.ResourceWithImportState = &uptimeMonitorHttpResource{}
	_ resource.ResourceWithModifyPlan  = &uptimeMonitorHttpResource{}
)

// NewUptimeMonitorHttpResource returns a new HTTP uptime monitor resource.
func NewUptimeMonitorHttpResource() resource.Resource {
	return &uptimeMonitorHttpResource{}
}

// uptimeMonitorHttpResource is the resource implementation.
type uptimeMonitorHttpResource struct {
	helpers.ResourceBase
}

// uptimeMonitorHttpModel is now the same as UptimeMonitorHttpModel since we've moved everything
type uptimeMonitorHttpModel = UptimeMonitorHttpModel

func (r *uptimeMonitorHttpResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor_http"
}

func (r *uptimeMonitorHttpResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resourceSchema := UptimeMonitorHttpResourceSchema(ctx)

	// Add description to the resource
	resourceSchema.Description = "Manages an HTTP/HTTPS uptime monitor in Phare. Monitors website availability and performance using HTTP/HTTPS requests."
	resourceSchema.MarkdownDescription = "Manages an HTTP/HTTPS uptime monitor in Phare. Monitors website availability and performance using HTTP/HTTPS requests."

	// Add common validators
	AddCommonValidators(ctx, &resourceSchema)

	resp.Schema = resourceSchema
}

// Configure is provided by helpers.ResourceBase

func (r *uptimeMonitorHttpResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Skip validation if resource is being destroyed
	if req.Plan.Raw.IsNull() {
		return
	}

	var plan uptimeMonitorHttpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate project scope configuration at plan time
	r.ValidateProjectScopeAtPlanTime(ctx, plan.ProjectScope, "phare_uptime_monitor_http", &resp.Diagnostics)
}

// Helper function to convert Terraform HTTP request model to client request config
func httpRequestModelToClientConfig(ctx context.Context, request HttpRequestModel) (client.MonitorRequestConfig, error) {
	config := client.MonitorRequestConfig{}

	// Extract request attributes
	if !request.Method.IsNull() && !request.Method.IsUnknown() {
		m := request.Method.ValueString()
		config.Method = &m
	}

	if !request.Url.IsNull() && !request.Url.IsUnknown() {
		u := request.Url.ValueString()
		config.URL = &u
	}

	if !request.Body.IsNull() && !request.Body.IsUnknown() {
		b := request.Body.ValueString()
		config.Body = &b
	}

	if !request.FollowRedirects.IsNull() && !request.FollowRedirects.IsUnknown() {
		b := request.FollowRedirects.ValueBool()
		config.FollowRedirects = &b
	}

	if !request.TlsSkipVerify.IsNull() && !request.TlsSkipVerify.IsUnknown() {
		b := request.TlsSkipVerify.ValueBool()
		config.TLSSkipVerify = &b
	}

	if !request.UserAgentSecret.IsNull() && !request.UserAgentSecret.IsUnknown() {
		s := request.UserAgentSecret.ValueString()
		config.UserAgentSecret = &s
	}

	// Headers
	if !request.Headers.IsNull() && !request.Headers.IsUnknown() {
		var headersModels []HeaderModel
		if diags := request.Headers.ElementsAs(ctx, &headersModels, false); !diags.HasError() {
			clientHeaders := make([]client.MonitorRequestHeader, len(headersModels))
			for i, h := range headersModels {
				clientHeaders[i] = client.MonitorRequestHeader{
					Name:  h.Name.ValueString(),
					Value: h.Value.ValueString(),
				}
			}
			config.Headers = clientHeaders
		}
	}

	return config, nil
}

// Helper function to convert client request config to Terraform HTTP request model
func clientConfigToHttpRequestModel(ctx context.Context, config client.MonitorRequestConfig) (HttpRequestModel, error) {
	request := HttpRequestModel{}

	if config.Method != nil {
		request.Method = types.StringValue(*config.Method)
	} else {
		request.Method = types.StringNull()
	}

	if config.URL != nil {
		request.Url = types.StringValue(*config.URL)
	} else {
		request.Url = types.StringNull()
	}

	if config.Body != nil {
		request.Body = types.StringValue(*config.Body)
	} else {
		request.Body = types.StringNull()
	}

	if config.FollowRedirects != nil {
		request.FollowRedirects = types.BoolValue(*config.FollowRedirects)
	} else {
		request.FollowRedirects = types.BoolNull()
	}

	if config.TLSSkipVerify != nil {
		request.TlsSkipVerify = types.BoolValue(*config.TLSSkipVerify)
	} else {
		request.TlsSkipVerify = types.BoolNull()
	}

	if config.UserAgentSecret != nil {
		request.UserAgentSecret = types.StringValue(*config.UserAgentSecret)
	} else {
		request.UserAgentSecret = types.StringNull()
	}

	// Convert headers
	if len(config.Headers) > 0 {
		headersList := make([]attr.Value, len(config.Headers))
		for i, h := range config.Headers {
			headerAttrs := map[string]attr.Value{
				"name":  types.StringValue(h.Name),
				"value": types.StringValue(h.Value),
			}
			headerObj, _ := types.ObjectValue(HeaderModelAttrTypes, headerAttrs)
			headersList[i] = headerObj
		}
		headersListVal, _ := types.ListValue(types.ObjectType{AttrTypes: HeaderModelAttrTypes}, headersList)
		request.Headers = headersListVal
	} else {
		request.Headers = types.ListNull(types.ObjectType{AttrTypes: HeaderModelAttrTypes})
	}

	return request, nil
}

// Helper to convert success assertions from client to Terraform
func clientAssertionsToHttpTerraformList(ctx context.Context, assertions []map[string]interface{}) (types.List, error) {
	if len(assertions) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}), nil
	}

	assertionsList := make([]attr.Value, len(assertions))
	for i, a := range assertions {
		assertionAttrs := map[string]attr.Value{
			"type":     types.StringValue(a["type"].(string)),
			"operator": types.StringValue(a["operator"].(string)),
			"value":    types.StringValue(a["value"].(string)),
			"selector": types.StringNull(),
		}
		if selector, ok := a["selector"]; ok && selector != nil {
			assertionAttrs["selector"] = types.StringValue(selector.(string))
		}
		assertionObj, _ := types.ObjectValue(SuccessAssertionModelAttrTypes, assertionAttrs)
		assertionsList[i] = assertionObj
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}, assertionsList)
	if diags.HasError() {
		return types.ListNull(types.ObjectType{AttrTypes: SuccessAssertionModelAttrTypes}), fmt.Errorf("error creating list: %v", diags)
	}
	return list, nil
}

func (r *uptimeMonitorHttpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan uptimeMonitorHttpModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_http", &resp.Diagnostics)
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
	reqConfig, err := httpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	// Convert success assertions (required for HTTP)
	var assertionsModels []SuccessAssertionModel
	resp.Diagnostics.Append(plan.SuccessAssertions.ElementsAs(ctx, &assertionsModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	successAssertions := make([]map[string]interface{}, len(assertionsModels))
	for i, a := range assertionsModels {
		assertionType := a.Type.ValueString()
		assertion := map[string]interface{}{
			"type":     assertionType,
			"operator": a.Operator.ValueString(),
			"value":    a.Value.ValueString(),
		}

		// For response_header type, selector is required
		if assertionType == "response_header" {
			if a.Selector.IsNull() || a.Selector.IsUnknown() {
				resp.Diagnostics.AddError(
					"Missing required selector",
					"Selector is required for response_header assertion type",
				)
				return
			}
			assertion["selector"] = a.Selector.ValueString()
		} else if !a.Selector.IsNull() && !a.Selector.IsUnknown() {
			// For other types, selector should not be present
			resp.Diagnostics.AddError(
				"Invalid selector for assertion type",
				fmt.Sprintf("Selector should not be specified for %s assertion type", assertionType),
			)
			return
		}

		successAssertions[i] = assertion
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "http", // Always http - TLS is determined by URL scheme
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		SuccessAssertions:     successAssertions,
	}

	// Call API to create monitor
	apiResp, err := scopedClient.CreateMonitor(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Creating monitor",
			fmt.Sprintf("Could not create monitor ID %d: %s", plan.Id.ValueInt64(), err.Error()),
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
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	plan.Request = requestModel

	// Convert success assertions
	assertionsList, err := ClientAssertionsToHttpTerraformList(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	plan.SuccessAssertions = assertionsList

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

func (r *uptimeMonitorHttpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state uptimeMonitorHttpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_http", &resp.Diagnostics)
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
			"Error Reading monitor",
			fmt.Sprintf("Could not read monitor ID %d: %s", state.Id.ValueInt64(), err.Error()),
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
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	state.Request = requestModel

	// Convert success assertions
	assertionsList, err := ClientAssertionsToHttpTerraformList(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	state.SuccessAssertions = assertionsList

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

func (r *uptimeMonitorHttpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan uptimeMonitorHttpModel
	var state uptimeMonitorHttpModel

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
	scopedClient := r.GetScopedClient(ctx, plan.ProjectScope, "phare_uptime_monitor_http", &resp.Diagnostics)
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
	reqConfig, err := httpRequestModelToClientConfig(ctx, plan.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	// Convert success assertions
	var assertionsModels []SuccessAssertionModel
	resp.Diagnostics.Append(plan.SuccessAssertions.ElementsAs(ctx, &assertionsModels, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	successAssertions := make([]map[string]interface{}, len(assertionsModels))
	for i, a := range assertionsModels {
		assertionType := a.Type.ValueString()
		assertion := map[string]interface{}{
			"type":     assertionType,
			"operator": a.Operator.ValueString(),
			"value":    a.Value.ValueString(),
		}

		if assertionType == "response_header" {
			if a.Selector.IsNull() || a.Selector.IsUnknown() {
				resp.Diagnostics.AddError(
					"Missing required selector",
					"Selector is required for response_header assertion type",
				)
				return
			}
			assertion["selector"] = a.Selector.ValueString()
		} else if !a.Selector.IsNull() && !a.Selector.IsUnknown() {
			resp.Diagnostics.AddError(
				"Invalid selector for assertion type",
				fmt.Sprintf("Selector should not be specified for %s assertion type", assertionType),
			)
			return
		}

		successAssertions[i] = assertion
	}

	apiReq := &client.MonitorRequest{
		Name:                  plan.Name.ValueString(),
		Protocol:              "http",
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		SuccessAssertions:     successAssertions,
	}

	// Call API to update monitor using ID from current state
	apiResp, err := scopedClient.UpdateMonitor(ctx, state.Id.ValueInt64(), apiReq)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Updating monitor",
			fmt.Sprintf("Could not update monitor ID %d: %s", state.Id.ValueInt64(), err.Error()),
		)
		return
	}

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

	// Convert request back to model
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Request",
			"Could not convert API response request: "+err.Error(),
		)
		return
	}
	plan.Request = requestModel

	// Convert success assertions
	assertionsList, err := clientAssertionsToHttpTerraformList(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	plan.SuccessAssertions = assertionsList

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

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *uptimeMonitorHttpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state uptimeMonitorHttpModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this resource
	scopedClient := r.GetScopedClient(ctx, state.ProjectScope, "phare_uptime_monitor_http", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API to delete monitor
	err := scopedClient.DeleteMonitor(ctx, state.Id.ValueInt64())
	if err != nil {
		if client.IsNotFoundError(err) {
			// Resource already deleted
			return
		}
		resp.Diagnostics.AddError(
			"Error Deleting monitor",
			fmt.Sprintf("Could not delete monitor ID %d: %s", state.Id.ValueInt64(), err.Error()),
		)
		return
	}
}

func (r *uptimeMonitorHttpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
