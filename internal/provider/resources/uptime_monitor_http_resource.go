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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"

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

// StatusCodeAssertionModel represents a status code assertion for HTTP monitors
type StatusCodeAssertionModel struct {
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// StatusCodeAssertionModelAttrTypes defines the attribute types for StatusCodeAssertionModel
var StatusCodeAssertionModelAttrTypes = map[string]attr.Type{
	"operator": types.StringType,
	"value":    types.StringType,
}

// ResponseHeaderAssertionModel represents a response header assertion for HTTP monitors
type ResponseHeaderAssertionModel struct {
	Selector types.String `tfsdk:"selector"`
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// ResponseHeaderAssertionModelAttrTypes defines the attribute types for ResponseHeaderAssertionModel
var ResponseHeaderAssertionModelAttrTypes = map[string]attr.Type{
	"selector": types.StringType,
	"operator": types.StringType,
	"value":    types.StringType,
}

// ResponseBodyAssertionModel represents a response body assertion for HTTP monitors
type ResponseBodyAssertionModel struct {
	Operator types.String `tfsdk:"operator"`
	Value    types.String `tfsdk:"value"`
}

// ResponseBodyAssertionModelAttrTypes defines the attribute types for ResponseBodyAssertionModel
var ResponseBodyAssertionModelAttrTypes = map[string]attr.Type{
	"operator": types.StringType,
	"value":    types.StringType,
}

// SuccessAssertionsModel represents the success assertions configuration
type SuccessAssertionsModel struct {
	StatusCode     types.List `tfsdk:"status_code"`
	ResponseHeader types.List `tfsdk:"response_header"`
	ResponseBody   types.List `tfsdk:"response_body"`
}

// SuccessAssertionsModelAttrTypes defines the attribute types for SuccessAssertionsModel
var SuccessAssertionsModelAttrTypes = map[string]attr.Type{
	"status_code":     types.ListType{ElemType: types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}},
	"response_header": types.ListType{ElemType: types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}},
	"response_body":   types.ListType{ElemType: types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}},
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
	Request           HttpRequestModel       `tfsdk:"request"`
	SuccessAssertions SuccessAssertionsModel `tfsdk:"success_assertions"`
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
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
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
							PlanModifiers: []planmodifier.String{
								helpers.TrimString(),
							},
						},
						"value": schema.StringAttribute{
							Required:            true,
							Description:         "Header value",
							MarkdownDescription: "Header value",
							PlanModifiers: []planmodifier.String{
								helpers.TrimString(),
							},
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
				Validators: []validator.String{
					stringvalidator.OneOf("GET", "POST", "PUT", "PATCH", "HEAD", "OPTIONS"),
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

	return schema.Schema{
		Attributes: baseAttributes,
		Blocks: map[string]schema.Block{
			"success_assertions": schema.SingleNestedBlock{
				Blocks: map[string]schema.Block{
					"status_code": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"operator": schema.StringAttribute{
									Required:            true,
									Description:         "Operator to use for the assertion (in, not_in)",
									MarkdownDescription: "Operator to use for the assertion (in, not_in)",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
									Validators: []validator.String{
										stringvalidator.OneOf("in", "not_in"),
									},
								},
								"value": schema.StringAttribute{
									Required:            true,
									Description:         "A comma-separated list of status code values, you can use x as a wildcard for any digit",
									MarkdownDescription: "A comma-separated list of status code values, you can use x as a wildcard for any digit",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
								},
							},
						},
						Description:         "Status code assertions",
						MarkdownDescription: "Status code assertions",
					},

					"response_header": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"selector": schema.StringAttribute{
									Required:            true,
									Description:         "The name of the header to assert",
									MarkdownDescription: "The name of the header to assert",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
								},
								"operator": schema.StringAttribute{
									Required:            true,
									Description:         "Operator to use for the assertion (equals, not_equals)",
									MarkdownDescription: "Operator to use for the assertion (equals, not_equals)",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
									Validators: []validator.String{
										stringvalidator.OneOf("equals", "not_equals"),
									},
								},
								"value": schema.StringAttribute{
									Required:            true,
									Description:         "The value of the header to assert",
									MarkdownDescription: "The value of the header to assert",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
								},
							},
						},
						Description:         "Response header assertions",
						MarkdownDescription: "Response header assertions",
					},

					"response_body": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"operator": schema.StringAttribute{
									Required:            true,
									Description:         "Operator to use for the assertion (contains, not_contains)",
									MarkdownDescription: "Operator to use for the assertion (contains, not_contains)",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
									Validators: []validator.String{
										stringvalidator.OneOf("contains", "not_contains"),
									},
								},
								"value": schema.StringAttribute{
									Required:            true,
									Description:         "A word or sentence to check in the response body",
									MarkdownDescription: "A word or sentence to check in the response body",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
								},
							},
						},
						Description:         "Response body assertions",
						MarkdownDescription: "Response body assertions",
					},
				},
				Description:         "List of assertions that must be true for the check to be considered successful",
				MarkdownDescription: "List of assertions that must be true for the check to be considered successful, [see docs](https://docs.phare.io/uptime/monitors#success-assertions)",
			},
		},
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
func clientAssertionsToHttpTerraformModel(ctx context.Context, assertions []map[string]interface{}) (SuccessAssertionsModel, error) {
	result := SuccessAssertionsModel{}

	if len(assertions) == 0 {
		// Return empty model with null lists
		result.StatusCode = types.ListNull(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes})
		result.ResponseHeader = types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes})
		result.ResponseBody = types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes})
		return result, nil
	}

	// Separate assertions by type
	var statusCodeAssertions []StatusCodeAssertionModel
	var responseHeaderAssertions []ResponseHeaderAssertionModel
	var responseBodyAssertions []ResponseBodyAssertionModel

	for _, a := range assertions {
		assertionType := a["type"].(string)
		operator := a["operator"].(string)
		value := a["value"].(string)

		switch assertionType {
		case "status_code":
			statusCodeAssertions = append(statusCodeAssertions, StatusCodeAssertionModel{
				Operator: types.StringValue(operator),
				Value:    types.StringValue(value),
			})
		case "response_header":
			if selector, ok := a["selector"]; ok && selector != nil {
				responseHeaderAssertions = append(responseHeaderAssertions, ResponseHeaderAssertionModel{
					Selector: types.StringValue(selector.(string)),
					Operator: types.StringValue(operator),
					Value:    types.StringValue(value),
				})
			}
		case "response_body":
			responseBodyAssertions = append(responseBodyAssertions, ResponseBodyAssertionModel{
				Operator: types.StringValue(operator),
				Value:    types.StringValue(value),
			})
		}
	}

	// Convert to types.List
	if len(statusCodeAssertions) > 0 {
		statusCodeElements := make([]attr.Value, len(statusCodeAssertions))
		for i, a := range statusCodeAssertions {
			attrMap := map[string]attr.Value{
				"operator": a.Operator,
				"value":    a.Value,
			}
			obj, _ := types.ObjectValue(StatusCodeAssertionModelAttrTypes, attrMap)
			statusCodeElements[i] = obj
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, statusCodeElements)
		if diags.HasError() {
			return result, fmt.Errorf("error creating status_code list: %v", diags)
		}
		result.StatusCode = list
	} else {
		result.StatusCode = types.ListNull(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes})
	}

	if len(responseHeaderAssertions) > 0 {
		responseHeaderElements := make([]attr.Value, len(responseHeaderAssertions))
		for i, a := range responseHeaderAssertions {
			attrMap := map[string]attr.Value{
				"selector": a.Selector,
				"operator": a.Operator,
				"value":    a.Value,
			}
			obj, _ := types.ObjectValue(ResponseHeaderAssertionModelAttrTypes, attrMap)
			responseHeaderElements[i] = obj
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}, responseHeaderElements)
		if diags.HasError() {
			return result, fmt.Errorf("error creating response_header list: %v", diags)
		}
		result.ResponseHeader = list
	} else {
		result.ResponseHeader = types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes})
	}

	if len(responseBodyAssertions) > 0 {
		responseBodyElements := make([]attr.Value, len(responseBodyAssertions))
		for i, a := range responseBodyAssertions {
			attrMap := map[string]attr.Value{
				"operator": a.Operator,
				"value":    a.Value,
			}
			obj, _ := types.ObjectValue(ResponseBodyAssertionModelAttrTypes, attrMap)
			responseBodyElements[i] = obj
		}
		list, diags := types.ListValue(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}, responseBodyElements)
		if diags.HasError() {
			return result, fmt.Errorf("error creating response_body list: %v", diags)
		}
		result.ResponseBody = list
	} else {
		result.ResponseBody = types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes})
	}

	return result, nil
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
	var successAssertions []map[string]interface{}

	// Process status_code assertions
	if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() {
		var statusCodeModels []StatusCodeAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.StatusCode.ElementsAs(ctx, &statusCodeModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range statusCodeModels {
			assertion := map[string]interface{}{
				"type":     "status_code",
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Process response_header assertions
	if !plan.SuccessAssertions.ResponseHeader.IsNull() && !plan.SuccessAssertions.ResponseHeader.IsUnknown() {
		var responseHeaderModels []ResponseHeaderAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.ResponseHeader.ElementsAs(ctx, &responseHeaderModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range responseHeaderModels {
			assertion := map[string]interface{}{
				"type":     "response_header",
				"selector": a.Selector.ValueString(),
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Process response_body assertions
	if !plan.SuccessAssertions.ResponseBody.IsNull() && !plan.SuccessAssertions.ResponseBody.IsUnknown() {
		var responseBodyModels []ResponseBodyAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.ResponseBody.ElementsAs(ctx, &responseBodyModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range responseBodyModels {
			assertion := map[string]interface{}{
				"type":     "response_body",
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Validate that at least one assertion type is provided
	if len(successAssertions) == 0 {
		resp.Diagnostics.AddError(
			"Missing success assertions",
			"At least one success assertion must be provided",
		)
		return
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
	successAssertionsModel, err := clientAssertionsToHttpTerraformModel(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	plan.SuccessAssertions = successAssertionsModel

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
	successAssertionsModel, err := clientAssertionsToHttpTerraformModel(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	state.SuccessAssertions = successAssertionsModel

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
	var successAssertions []map[string]interface{}

	// Process status_code assertions
	if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() {
		var statusCodeModels []StatusCodeAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.StatusCode.ElementsAs(ctx, &statusCodeModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range statusCodeModels {
			assertion := map[string]interface{}{
				"type":     "status_code",
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Process response_header assertions
	if !plan.SuccessAssertions.ResponseHeader.IsNull() && !plan.SuccessAssertions.ResponseHeader.IsUnknown() {
		var responseHeaderModels []ResponseHeaderAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.ResponseHeader.ElementsAs(ctx, &responseHeaderModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range responseHeaderModels {
			assertion := map[string]interface{}{
				"type":     "response_header",
				"selector": a.Selector.ValueString(),
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Process response_body assertions
	if !plan.SuccessAssertions.ResponseBody.IsNull() && !plan.SuccessAssertions.ResponseBody.IsUnknown() {
		var responseBodyModels []ResponseBodyAssertionModel
		resp.Diagnostics.Append(plan.SuccessAssertions.ResponseBody.ElementsAs(ctx, &responseBodyModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, a := range responseBodyModels {
			assertion := map[string]interface{}{
				"type":     "response_body",
				"operator": a.Operator.ValueString(),
				"value":    a.Value.ValueString(),
			}
			successAssertions = append(successAssertions, assertion)
		}
	}

	// Validate that at least one assertion type is provided
	if len(successAssertions) == 0 {
		resp.Diagnostics.AddError(
			"Missing success assertions",
			"At least one success assertion must be provided",
		)
		return
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
	successAssertionsModel, err := clientAssertionsToHttpTerraformModel(ctx, apiResp.SuccessAssertions)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Converting Assertions",
			"Could not convert API response assertions: "+err.Error(),
		)
		return
	}
	plan.SuccessAssertions = successAssertionsModel

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
