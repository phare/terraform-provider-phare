package resources

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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
	Name   types.String `tfsdk:"name"`
	Value  types.String `tfsdk:"value"`
	Secret types.Bool   `tfsdk:"secret"`
}

// HeaderModelAttrTypes defines the attribute types for HeaderModel
var HeaderModelAttrTypes = map[string]attr.Type{
	"name":   types.StringType,
	"value":  types.StringType,
	"secret": types.BoolType,
}

// uptimeMonitorSecretNote is appended to the description of the request secrets the API never returns
const uptimeMonitorSecretNote = " The API never returns this value, it is kept from the Terraform state and changes made outside of Terraform are not detected."

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
	Request           *HttpRequestModel       `tfsdk:"request"`
	SuccessAssertions *SuccessAssertionsModel `tfsdk:"success_assertions"`
}

// UptimeMonitorHttpResourceSchema defines the schema for the HTTP uptime monitor resource
func UptimeMonitorHttpResourceSchema(ctx context.Context) schema.Schema {
	// Start with the base schema
	baseAttributes := UptimeMonitorBaseResourceSchema(ctx)

	return schema.Schema{
		Attributes: baseAttributes,
		Blocks: map[string]schema.Block{
			"request": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"body": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Request body for POST, PUT, PATCH",
						MarkdownDescription: "Request body for POST, PUT, PATCH",
						PlanModifiers: []planmodifier.String{
							helpers.TrimString(),
						},
						Validators: []validator.String{
							helpers.TrimmedLengthAtMost(500),
						},
					},
					"follow_redirects": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Follow HTTP redirects (default: true)",
						MarkdownDescription: "Follow HTTP redirects (default: true)",
						Default:             booldefault.StaticBool(true),
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
						Validators: []validator.String{
							stringvalidator.LengthAtMost(255),
							stringvalidator.RegexMatches(regexp.MustCompile(`^https?://`), "must be a valid http or https URL"),
						},
					},
					"user_agent_secret": schema.StringAttribute{
						Optional:            true,
						Sensitive:           true,
						Description:         "Secret value in User-Agent header for authentication." + uptimeMonitorSecretNote,
						MarkdownDescription: "Secret value in User-Agent header for authentication, [see docs](https://docs.phare.io/uptime/monitors#user-agent)." + uptimeMonitorSecretNote,
						PlanModifiers: []planmodifier.String{
							helpers.TrimString(),
						},
						Validators: []validator.String{
							helpers.TrimmedLengthBetween(1, 50),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"headers": schema.ListNestedBlock{
						NestedObject: schema.NestedBlockObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Required:            true,
									Description:         "Header name",
									MarkdownDescription: "Header name",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
									Validators: []validator.String{
										stringvalidator.LengthBetween(1, 50),
										stringvalidator.RegexMatches(regexp.MustCompile("^[a-zA-Z0-9-]+$"), "must contain only alphanumeric characters and dashes"),
										stringvalidator.NoneOf("User-Agent", "Signature", "Signature-Input", "Signature-Agent"),
									},
								},
								"value": schema.StringAttribute{
									Required: true,
									// The framework can't hide a single list element, so public header values are hidden too
									Sensitive:           true,
									Description:         "Header value. The API never returns the value of a secret header, it is kept from the Terraform state and changes made outside of Terraform are not detected.",
									MarkdownDescription: "Header value. The API never returns the value of a secret header, it is kept from the Terraform state and changes made outside of Terraform are not detected.",
									PlanModifiers: []planmodifier.String{
										helpers.TrimString(),
									},
									Validators: []validator.String{
										helpers.TrimmedLengthAtMost(1024),
									},
								},
								// No default: a header without it keeps the flag stored by Phare, as the
								// platform flags existing credential headers as secret on its own
								"secret": schema.BoolAttribute{
									Optional:            true,
									Computed:            true,
									Description:         "Whether the header value is a secret, encrypted by Phare and never returned by the API. When left out, a header keeps its current flag, false for a new header, and only an explicit false makes a secret header public again.",
									MarkdownDescription: "Whether the header value is a secret, encrypted by Phare and never returned by the API. When left out, a header keeps its current flag, `false` for a new header, and only an explicit `false` makes a secret header public again.",
								},
							},
						},
						Description:         "Additional HTTP headers (max 10), names are unique ignoring case",
						MarkdownDescription: "Additional HTTP headers (max 10), names are unique ignoring case, [see docs](https://docs.phare.io/uptime/monitors#headers)",
						Validators: []validator.List{
							helpers.UniqueAttributeIgnoreCase("name"),
						},
					},
				},
				Description:         "HTTP/HTTPS request configuration",
				MarkdownDescription: "HTTP/HTTPS request configuration",
			},
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
									Validators: []validator.String{
										helpers.TrimmedLengthBetween(1, 100),
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
	description := "Manages an HTTP/HTTPS uptime monitor in Phare. Monitors website availability and performance using HTTP/HTTPS requests."
	importNote := " The API never returns the user agent secret and the values of secret headers: an imported monitor has none in its state, so the first plan after the import shows them as changes, and applying it sets them again."
	resourceSchema.Description = description + importNote
	resourceSchema.MarkdownDescription = description + importNote

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

	helpers.CheckProjectScopeRequiresReplace(ctx, req, resp)

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
			"The 'request' block is required for HTTP uptime monitors.",
		)
	} else {
		// Validate request body is only allowed for POST, PUT, PATCH
		if !plan.Request.Body.IsNull() && !plan.Request.Body.IsUnknown() && plan.Request.Body.ValueString() != "" {
			if !plan.Request.Method.IsUnknown() {
				method := plan.Request.Method.ValueString()
				if method != "POST" && method != "PUT" && method != "PATCH" {
					resp.Diagnostics.AddAttributeError(
						path.Root("request").AtName("body"),
						"Invalid Request Body",
						"Request body is prohibited unless method is POST, PUT, or PATCH",
					)
				}
			}
		}

		// Validate headers count does not exceed 10
		if !plan.Request.Headers.IsNull() && !plan.Request.Headers.IsUnknown() && len(plan.Request.Headers.Elements()) > 10 {
			resp.Diagnostics.AddAttributeError(
				path.Root("request").AtName("headers"),
				"Too Many Headers",
				"Cannot configure more than 10 HTTP headers",
			)
		}

		// Plan the secret flag of the headers that leave it out of the configuration,
		// without stopping the validations below on an error
		var config, state uptimeMonitorHttpModel
		var secretDiags diag.Diagnostics
		if !req.Config.Raw.IsNull() {
			secretDiags.Append(req.Config.Get(ctx, &config)...)
		}
		if !req.State.Raw.IsNull() {
			secretDiags.Append(req.State.Get(ctx, &state)...)
		}
		if !secretDiags.HasError() {
			headers, diags := planHeaderSecrets(ctx, plan.Request, config.Request, state.Request)
			secretDiags.Append(diags...)
			if !secretDiags.HasError() && !headers.Equal(plan.Request.Headers) {
				secretDiags.Append(resp.Plan.SetAttribute(ctx, path.Root("request").AtName("headers"), headers)...)
			}
		}
		resp.Diagnostics.Append(secretDiags...)
	}

	// Validate success_assertions block is provided
	if plan.SuccessAssertions == nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("success_assertions"),
			"Missing Success Assertions",
			"The 'success_assertions' block is required for HTTP uptime monitors.",
		)
	} else {
		hasAssertions := false
		hasUnknown := plan.SuccessAssertions.StatusCode.IsUnknown() ||
			plan.SuccessAssertions.ResponseHeader.IsUnknown() ||
			plan.SuccessAssertions.ResponseBody.IsUnknown()

		if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() && len(plan.SuccessAssertions.StatusCode.Elements()) > 0 {
			hasAssertions = true
		}
		if !plan.SuccessAssertions.ResponseHeader.IsNull() && !plan.SuccessAssertions.ResponseHeader.IsUnknown() && len(plan.SuccessAssertions.ResponseHeader.Elements()) > 0 {
			hasAssertions = true
		}
		if !plan.SuccessAssertions.ResponseBody.IsNull() && !plan.SuccessAssertions.ResponseBody.IsUnknown() && len(plan.SuccessAssertions.ResponseBody.Elements()) > 0 {
			hasAssertions = true
		}

		if !hasAssertions && !hasUnknown {
			resp.Diagnostics.AddAttributeError(
				path.Root("success_assertions"),
				"Missing Success Assertions",
				"At least one success assertion (status_code, response_header, or response_body) must be provided.",
			)
		}

		if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() && len(plan.SuccessAssertions.StatusCode.Elements()) > 1 {
			resp.Diagnostics.AddAttributeError(
				path.Root("success_assertions").AtName("status_code"),
				"Too Many Status Code Assertions",
				"Only one status code assertion is allowed",
			)
		}
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

// headersByName returns the known headers of a request model by lowercase name, as header
// names are case-insensitive and the API normalizes their case
func headersByName(ctx context.Context, request *HttpRequestModel) (map[string]HeaderModel, diag.Diagnostics) {
	byName := make(map[string]HeaderModel)
	if request == nil || request.Headers.IsNull() || request.Headers.IsUnknown() {
		return byName, nil
	}

	var headers []HeaderModel
	diags := request.Headers.ElementsAs(ctx, &headers, false)
	for _, h := range headers {
		if !h.Name.IsNull() && !h.Name.IsUnknown() {
			byName[strings.ToLower(h.Name.ValueString())] = h
		}
	}

	return byName, diags
}

// planHeaderSecrets returns the planned headers with the secret flag of the headers that leave it
// out of the configuration: the flag of the header with the same name in the prior state, as the
// API keeps the stored flag, or false for a new header. Only an explicit false makes a secret
// header public. The framework plans these flags as unknown, by list index, as soon as the monitor
// changes; matching by name keeps the plan accurate when headers are added, removed or reordered.
func planHeaderSecrets(ctx context.Context, plan, config, state *HttpRequestModel) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics
	if plan == nil || plan.Headers.IsNull() || plan.Headers.IsUnknown() {
		if plan == nil {
			return types.ListNull(types.ObjectType{AttrTypes: HeaderModelAttrTypes}), nil
		}
		return plan.Headers, nil
	}

	var planHeaders, configHeaders []HeaderModel
	diags.Append(plan.Headers.ElementsAs(ctx, &planHeaders, false)...)
	if config != nil && !config.Headers.IsNull() && !config.Headers.IsUnknown() {
		diags.Append(config.Headers.ElementsAs(ctx, &configHeaders, false)...)
	}
	priorHeaders, priorDiags := headersByName(ctx, state)
	diags.Append(priorDiags...)
	if diags.HasError() {
		return plan.Headers, diags
	}

	for i, h := range planHeaders {
		// Set in the configuration, or kept from the prior state when the monitor doesn't change
		if (i < len(configHeaders) && !configHeaders[i].Secret.IsNull()) || !h.Secret.IsUnknown() {
			continue
		}
		if h.Name.IsUnknown() {
			continue
		}

		prior, ok := priorHeaders[strings.ToLower(h.Name.ValueString())]
		planHeaders[i].Secret = types.BoolValue(ok && prior.Secret.ValueBool())
	}

	headers, listDiags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: HeaderModelAttrTypes}, planHeaders)
	diags.Append(listDiags...)

	return headers, diags
}

// Helper function to convert Terraform HTTP request model to client request config,
// prior is the request of the prior state on update and nil on create
func httpRequestModelToClientConfig(ctx context.Context, request, prior *HttpRequestModel) (client.MonitorRequestConfig, error) {
	config := client.MonitorRequestConfig{}

	if request == nil {
		return config, nil
	}

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

	// The API keeps the user agent secret when it is omitted, it is only removed when it is
	// removed from the configuration: a secret unknown to Terraform (imported or set outside
	// of Terraform) is kept, as the plan doesn't show its removal
	if !request.UserAgentSecret.IsNull() && !request.UserAgentSecret.IsUnknown() {
		s := request.UserAgentSecret.ValueString()
		config.UserAgentSecret = &s
	} else if request.UserAgentSecret.IsNull() && prior != nil && !prior.UserAgentSecret.IsNull() {
		config.RemoveUserAgentSecret = true
	}

	// Headers are always sent with their value, the API keeps the encrypted value of a secret
	// header when it doesn't change. A header without a secret flag keeps its current flag.
	if !request.Headers.IsNull() && !request.Headers.IsUnknown() {
		var headersModels []HeaderModel
		if diags := request.Headers.ElementsAs(ctx, &headersModels, false); diags.HasError() {
			return config, fmt.Errorf("could not read headers: %v", diags)
		}

		clientHeaders := make([]client.MonitorRequestHeader, len(headersModels))
		for i, h := range headersModels {
			clientHeaders[i] = client.MonitorRequestHeader{
				Name:   h.Name.ValueString(),
				Value:  h.Value.ValueStringPointer(),
				Secret: h.Secret.ValueBoolPointer(),
			}
		}
		config.Headers = clientHeaders
	}

	return config, nil
}

// Helper function to convert client request config to Terraform HTTP request model. The API
// never returns the secrets, they are kept from prior: the plan on create and update, the
// prior state on read, nil on import.
func clientConfigToHttpRequestModel(ctx context.Context, config client.MonitorRequestConfig, prior *HttpRequestModel) (*HttpRequestModel, error) {
	request := &HttpRequestModel{}

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

	// The user agent secret is kept from prior while the API reports that one is set, it is null
	// when none is set, which detects a secret removed outside of Terraform
	switch {
	case config.UserAgentSecret != nil:
		request.UserAgentSecret = types.StringValue(*config.UserAgentSecret)
	case config.UserAgentSecretEnabled != nil && *config.UserAgentSecretEnabled && prior != nil && !prior.UserAgentSecret.IsUnknown():
		request.UserAgentSecret = prior.UserAgentSecret
	default:
		request.UserAgentSecret = types.StringNull()
	}

	priorHeaders, diags := headersByName(ctx, prior)
	if diags.HasError() {
		return nil, fmt.Errorf("could not read prior headers: %v", diags)
	}

	// Convert headers
	if len(config.Headers) > 0 {
		headersList := make([]attr.Value, len(config.Headers))
		for i, h := range config.Headers {
			secret := h.Secret != nil && *h.Secret
			name := types.StringValue(h.Name)
			value := types.StringPointerValue(h.Value)

			if prior, ok := priorHeaders[strings.ToLower(h.Name)]; ok {
				// The API normalizes the case of header names, keep the configured one
				name = prior.Name
				// The value of a secret header is never returned
				if h.Value == nil && secret && !prior.Value.IsUnknown() {
					value = prior.Value
				}
			}

			headerObj, objDiags := types.ObjectValue(HeaderModelAttrTypes, map[string]attr.Value{
				"name":   name,
				"value":  value,
				"secret": types.BoolValue(secret),
			})
			if objDiags.HasError() {
				return nil, fmt.Errorf("could not convert header %q: %v", h.Name, objDiags)
			}
			headersList[i] = headerObj
		}
		headersListVal, listDiags := types.ListValue(types.ObjectType{AttrTypes: HeaderModelAttrTypes}, headersList)
		if listDiags.HasError() {
			return nil, fmt.Errorf("could not convert headers: %v", listDiags)
		}
		request.Headers = headersListVal
	} else {
		request.Headers = types.ListNull(types.ObjectType{AttrTypes: HeaderModelAttrTypes})
	}

	return request, nil
}

// Helper to convert success assertions from client to Terraform
func clientAssertionsToHttpTerraformModel(ctx context.Context, assertions []map[string]any) (*SuccessAssertionsModel, error) {
	result := &SuccessAssertionsModel{}

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

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert request object
	reqConfig, err := httpRequestModelToClientConfig(ctx, plan.Request, nil)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	// Convert success assertions (required for HTTP)
	var successAssertions []map[string]any

	if plan.SuccessAssertions != nil {
		// Process status_code assertions
		if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() {
			var statusCodeModels []StatusCodeAssertionModel
			resp.Diagnostics.Append(plan.SuccessAssertions.StatusCode.ElementsAs(ctx, &statusCodeModels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			for _, a := range statusCodeModels {
				assertion := map[string]any{
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
				assertion := map[string]any{
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
				assertion := map[string]any{
					"type":     "response_body",
					"operator": a.Operator.ValueString(),
					"value":    a.Value.ValueString(),
				}
				successAssertions = append(successAssertions, assertion)
			}
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
		DisplayName:           plan.DisplayName.ValueStringPointer(),
		Protocol:              "http", // Always http - TLS is determined by URL scheme
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		RegionThreshold:       plan.RegionThreshold.ValueInt64(),
		SuccessAssertions:     successAssertions,
		Tags:                  tags,
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
	plan.DisplayName = types.StringPointerValue(apiResp.DisplayName)
	plan.Interval = types.Int64Value(apiResp.Interval)
	plan.Timeout = types.Int64Value(apiResp.Timeout)
	plan.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	plan.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	plan.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request back to model
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request, plan.Request)
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

	plan.Tags = helpers.StringSliceToSet(apiResp.Tags, plan.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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
	state.DisplayName = types.StringPointerValue(apiResp.DisplayName)
	state.Interval = types.Int64Value(apiResp.Interval)
	state.Timeout = types.Int64Value(apiResp.Timeout)
	state.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	state.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	state.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	state.Status = types.StringValue(apiResp.Status)
	state.Paused = types.BoolValue(apiResp.Paused)
	state.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request, state.Request)
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

	state.Tags = helpers.StringSliceToSet(apiResp.Tags, state.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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

	if !helpers.ValidateNoProjectScopeChange(ctx, req.State, req.Plan, &resp.Diagnostics) {
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

	var tags []string
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		resp.Diagnostics.Append(plan.Tags.ElementsAs(ctx, &tags, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Convert request object
	reqConfig, err := httpRequestModelToClientConfig(ctx, plan.Request, state.Request)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error Parsing Request Configuration",
			"Could not parse request configuration: "+err.Error(),
		)
		return
	}

	// Convert success assertions
	var successAssertions []map[string]any

	if plan.SuccessAssertions != nil {
		// Process status_code assertions
		if !plan.SuccessAssertions.StatusCode.IsNull() && !plan.SuccessAssertions.StatusCode.IsUnknown() {
			var statusCodeModels []StatusCodeAssertionModel
			resp.Diagnostics.Append(plan.SuccessAssertions.StatusCode.ElementsAs(ctx, &statusCodeModels, false)...)
			if resp.Diagnostics.HasError() {
				return
			}

			for _, a := range statusCodeModels {
				assertion := map[string]any{
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
				assertion := map[string]any{
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
				assertion := map[string]any{
					"type":     "response_body",
					"operator": a.Operator.ValueString(),
					"value":    a.Value.ValueString(),
				}
				successAssertions = append(successAssertions, assertion)
			}
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
		DisplayName:           plan.DisplayName.ValueStringPointer(),
		Protocol:              "http",
		Request:               reqConfig,
		Interval:              plan.Interval.ValueInt64(),
		Timeout:               plan.Timeout.ValueInt64(),
		Regions:               regions,
		IncidentConfirmations: plan.IncidentConfirmations.ValueInt64(),
		RecoveryConfirmations: plan.RecoveryConfirmations.ValueInt64(),
		RegionThreshold:       plan.RegionThreshold.ValueInt64(),
		SuccessAssertions:     successAssertions,
		Tags:                  tags,
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
	plan.DisplayName = types.StringPointerValue(apiResp.DisplayName)
	plan.Interval = types.Int64Value(apiResp.Interval)
	plan.Timeout = types.Int64Value(apiResp.Timeout)
	plan.IncidentConfirmations = types.Int64Value(apiResp.IncidentConfirmations)
	plan.RecoveryConfirmations = types.Int64Value(apiResp.RecoveryConfirmations)
	plan.RegionThreshold = types.Int64Value(apiResp.RegionThreshold)
	plan.Status = types.StringValue(apiResp.Status)
	plan.Paused = types.BoolValue(apiResp.Paused)
	plan.ProjectId = types.Int64Value(apiResp.ProjectID)

	// Convert request back to model
	requestModel, err := clientConfigToHttpRequestModel(ctx, apiResp.Request, plan.Request)
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

	plan.Tags = helpers.StringSliceToSet(apiResp.Tags, plan.Tags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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
	helpers.ImportStateWithProjectScope(ctx, req, resp, r.GetClient(), "Monitor", true)
}
