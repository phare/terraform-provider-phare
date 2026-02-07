package datasources

import (
	"context"
	"encoding/json"
	"fmt"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &uptimeMonitorDataSource{}
	_ datasource.DataSourceWithConfigure = &uptimeMonitorDataSource{}
)

// NewUptimeMonitorDataSource returns a new monitor data source.
func NewUptimeMonitorDataSource() datasource.DataSource {
	return &uptimeMonitorDataSource{}
}

// uptimeMonitorDataSource is the data source implementation.
type uptimeMonitorDataSource struct {
	helpers.DataSourceBase
}

// monitorModel describes the common monitor data model used by both
// the single monitor and monitors list data sources.
type monitorModel struct {
	ID                    types.Int64   `tfsdk:"id"`
	ProjectID             types.Int64   `tfsdk:"project_id"`
	Name                  types.String  `tfsdk:"name"`
	Protocol              types.String  `tfsdk:"protocol"`
	Request               types.Object  `tfsdk:"request"`
	Interval              types.Int64   `tfsdk:"interval"`
	Timeout               types.Int64   `tfsdk:"timeout"`
	SuccessAssertions     types.String  `tfsdk:"success_assertions"`
	IncidentConfirmations types.Int64   `tfsdk:"incident_confirmations"`
	RecoveryConfirmations types.Int64   `tfsdk:"recovery_confirmations"`
	Regions               types.List    `tfsdk:"regions"`
	Status                types.String  `tfsdk:"status"`
	Paused                types.Bool    `tfsdk:"paused"`
	ResponseTime          types.Int64   `tfsdk:"response_time"`
	OneDayAvailability    types.Float64 `tfsdk:"one_day_availability"`
	SevenDaysAvailability types.Float64 `tfsdk:"seven_days_availability"`
	CreatedAt             types.String  `tfsdk:"created_at"`
	UpdatedAt             types.String  `tfsdk:"updated_at"`
}

// mapMonitorToModel maps an API monitor response to the shared monitor model
func mapMonitorToModel(ctx context.Context, monitor *client.MonitorResponse, resp *datasource.ReadResponse) monitorModel {
	model := monitorModel{
		ID:                    types.Int64Value(monitor.ID),
		ProjectID:             types.Int64Value(monitor.ProjectID),
		Name:                  types.StringValue(monitor.Name),
		Protocol:              types.StringValue(monitor.Protocol),
		Interval:              types.Int64Value(monitor.Interval),
		Timeout:               types.Int64Value(monitor.Timeout),
		IncidentConfirmations: types.Int64Value(monitor.IncidentConfirmations),
		RecoveryConfirmations: types.Int64Value(monitor.RecoveryConfirmations),
		Status:                types.StringValue(monitor.Status),
		Paused:                types.BoolValue(monitor.Paused),
		CreatedAt:             types.StringValue(monitor.CreatedAt),
		UpdatedAt:             types.StringValue(monitor.UpdatedAt),
	}

	// Build nested request object
	requestAttrs := make(map[string]attr.Value)

	// HTTP fields
	if monitor.Request.URL != nil {
		requestAttrs["url"] = types.StringValue(*monitor.Request.URL)
	} else {
		requestAttrs["url"] = types.StringNull()
	}
	if monitor.Request.Method != nil {
		requestAttrs["method"] = types.StringValue(*monitor.Request.Method)
	} else {
		requestAttrs["method"] = types.StringNull()
	}
	if monitor.Request.Body != nil {
		requestAttrs["body"] = types.StringValue(*monitor.Request.Body)
	} else {
		requestAttrs["body"] = types.StringNull()
	}
	if monitor.Request.TLSSkipVerify != nil {
		requestAttrs["tls_skip_verify"] = types.BoolValue(*monitor.Request.TLSSkipVerify)
	} else {
		requestAttrs["tls_skip_verify"] = types.BoolNull()
	}
	if monitor.Request.FollowRedirects != nil {
		requestAttrs["follow_redirects"] = types.BoolValue(*monitor.Request.FollowRedirects)
	} else {
		requestAttrs["follow_redirects"] = types.BoolNull()
	}
	if monitor.Request.UserAgentSecret != nil {
		requestAttrs["user_agent_secret"] = types.StringValue(*monitor.Request.UserAgentSecret)
	} else {
		requestAttrs["user_agent_secret"] = types.StringNull()
	}

	// Map headers
	if len(monitor.Request.Headers) > 0 {
		headerType := types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		}

		headerElements := make([]attr.Value, len(monitor.Request.Headers))
		for i, header := range monitor.Request.Headers {
			headerObj, diags := types.ObjectValue(
				headerType.AttrTypes,
				map[string]attr.Value{
					"name":  types.StringValue(header.Name),
					"value": types.StringValue(header.Value),
				},
			)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return monitorModel{}
			}
			headerElements[i] = headerObj
		}

		headersList, diags := types.ListValue(headerType, headerElements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return monitorModel{}
		}
		requestAttrs["headers"] = headersList
	} else {
		requestAttrs["headers"] = types.ListNull(types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name":  types.StringType,
				"value": types.StringType,
			},
		})
	}

	// TCP fields
	if monitor.Request.Host != nil {
		requestAttrs["host"] = types.StringValue(*monitor.Request.Host)
	} else {
		requestAttrs["host"] = types.StringNull()
	}
	if monitor.Request.Connection != nil {
		requestAttrs["connection"] = types.StringValue(*monitor.Request.Connection)
	} else {
		requestAttrs["connection"] = types.StringNull()
	}

	// Handle port conversion
	if monitor.Request.Port != nil {
		var portInt int64
		_, err := fmt.Sscanf(*monitor.Request.Port, "%d", &portInt)
		if err == nil {
			requestAttrs["port"] = types.Int64Value(portInt)
		} else {
			requestAttrs["port"] = types.Int64Null()
		}
	} else {
		requestAttrs["port"] = types.Int64Null()
	}

	// Create the request object
	requestAttrTypes := map[string]attr.Type{
		"url":               types.StringType,
		"method":            types.StringType,
		"body":              types.StringType,
		"tls_skip_verify":   types.BoolType,
		"follow_redirects":  types.BoolType,
		"user_agent_secret": types.StringType,
		"headers": types.ListType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"name":  types.StringType,
					"value": types.StringType,
				},
			},
		},
		"host":       types.StringType,
		"port":       types.Int64Type,
		"connection": types.StringType,
	}

	requestObj, diags := types.ObjectValue(requestAttrTypes, requestAttrs)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return monitorModel{}
	}
	model.Request = requestObj

	// Map success_assertions
	if len(monitor.SuccessAssertions) > 0 {
		assertionsJSON, err := json.Marshal(monitor.SuccessAssertions)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Marshal Success Assertions",
				err.Error(),
			)
			return monitorModel{}
		}
		model.SuccessAssertions = types.StringValue(string(assertionsJSON))
	} else {
		model.SuccessAssertions = types.StringNull()
	}

	// Map regions
	var regionsList types.List
	if len(monitor.Regions) == 0 {
		regionsList = types.ListNull(types.StringType)
	} else {
		elements := make([]attr.Value, len(monitor.Regions))
		for i, value := range monitor.Regions {
			elements[i] = types.StringValue(value)
		}
		var diags diag.Diagnostics
		regionsList, diags = types.ListValue(types.StringType, elements)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return monitorModel{}
		}
	}
	model.Regions = regionsList

	// Map optional statistics
	if monitor.ResponseTime != nil {
		model.ResponseTime = types.Int64Value(*monitor.ResponseTime)
	} else {
		model.ResponseTime = types.Int64Null()
	}
	if monitor.OneDayAvailability != nil {
		model.OneDayAvailability = types.Float64Value(*monitor.OneDayAvailability)
	} else {
		model.OneDayAvailability = types.Float64Null()
	}
	if monitor.SevenDaysAvailability != nil {
		model.SevenDaysAvailability = types.Float64Value(*monitor.SevenDaysAvailability)
	} else {
		model.SevenDaysAvailability = types.Float64Null()
	}

	return model
}

// monitorSchemaAttributes returns the common schema attributes for monitors
func monitorSchemaAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Computed:    true,
			Description: "Monitor ID",
		},
		"project_id": schema.Int64Attribute{
			Computed:    true,
			Description: "Parent project ID",
		},
		"name": schema.StringAttribute{
			Computed:    true,
			Description: "Monitor name",
		},
		"protocol": schema.StringAttribute{
			Computed:    true,
			Description: "Monitoring protocol (http, tcp)",
		},
		"request": schema.SingleNestedAttribute{
			Computed:    true,
			Description: "Monitor request configuration (varies by protocol)",
			Attributes: map[string]schema.Attribute{
				// HTTP fields
				"url": schema.StringAttribute{
					Computed:    true,
					Description: "URL being monitored (for HTTP/HTTPS monitors)",
				},
				"method": schema.StringAttribute{
					Computed:    true,
					Description: "HTTP method (for HTTP monitors)",
				},
				"body": schema.StringAttribute{
					Computed:    true,
					Description: "HTTP request body (for HTTP monitors)",
				},
				"tls_skip_verify": schema.BoolAttribute{
					Computed:    true,
					Description: "Whether to skip TLS certificate verification",
				},
				"follow_redirects": schema.BoolAttribute{
					Computed:    true,
					Description: "Whether to follow HTTP redirects",
				},
				"user_agent_secret": schema.StringAttribute{
					Computed:    true,
					Description: "User agent secret for request verification",
				},
				"headers": schema.ListNestedAttribute{
					Computed:    true,
					Description: "HTTP headers to send with the request",
					NestedObject: schema.NestedAttributeObject{
						Attributes: map[string]schema.Attribute{
							"name": schema.StringAttribute{
								Computed:    true,
								Description: "Header name",
							},
							"value": schema.StringAttribute{
								Computed:    true,
								Description: "Header value",
							},
						},
					},
				},
				// TCP fields
				"host": schema.StringAttribute{
					Computed:    true,
					Description: "Host to connect to (for TCP monitors)",
				},
				"port": schema.Int64Attribute{
					Computed:    true,
					Description: "Port to connect to (for TCP monitors)",
				},
				"connection": schema.StringAttribute{
					Computed:    true,
					Description: "Connection type (for TCP monitors)",
				},
			},
		},
		"interval": schema.Int64Attribute{
			Computed:    true,
			Description: "Monitoring interval in seconds",
		},
		"timeout": schema.Int64Attribute{
			Computed:    true,
			Description: "Monitoring timeout in milliseconds",
		},
		"success_assertions": schema.StringAttribute{
			Computed:    true,
			Description: "Success assertions as JSON",
		},
		"incident_confirmations": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of confirmations before creating an incident",
		},
		"recovery_confirmations": schema.Int64Attribute{
			Computed:    true,
			Description: "Number of confirmations before marking as recovered",
		},
		"regions": schema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "Regions where monitoring is performed",
		},
		"status": schema.StringAttribute{
			Computed:    true,
			Description: "Current monitor status",
		},
		"paused": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether the monitor is paused",
		},
		"response_time": schema.Int64Attribute{
			Computed:    true,
			Description: "Latest response time in milliseconds",
		},
		"one_day_availability": schema.Float64Attribute{
			Computed:    true,
			Description: "Availability over the last 24 hours",
		},
		"seven_days_availability": schema.Float64Attribute{
			Computed:    true,
			Description: "Availability over the last 7 days",
		},
		"created_at": schema.StringAttribute{
			Computed:    true,
			Description: "Creation timestamp",
		},
		"updated_at": schema.StringAttribute{
			Computed:    true,
			Description: "Last update timestamp",
		},
	}
}

// uptimeMonitorDataSourceModel describes the data source data model.
type uptimeMonitorDataSourceModel struct {
	monitorModel
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
}

func (d *uptimeMonitorDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_uptime_monitor"
}

func (d *uptimeMonitorDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Fetches a specific uptime monitor by ID.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Monitor ID",
			},
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this data source. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
		},
	}

	// Add the common monitor attributes
	for key, attr := range monitorSchemaAttributes() {
		resp.Schema.Attributes[key] = attr
	}
}

func (d *uptimeMonitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config uptimeMonitorDataSourceModel

	// Read configuration
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get scoped client for this data source
	scopedClient := helpers.ConfigureResourceWithProjectScope(ctx, d.GetClient(), config.ProjectScope, "phare_uptime_monitor", &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	// Fetch the monitor by ID
	monitor, err := scopedClient.GetMonitor(ctx, config.ID.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Monitor",
			fmt.Sprintf("Error reading monitor ID %d: %s", config.ID.ValueInt64(), err.Error()),
		)
		return
	}

	// Map API response to state using shared helper
	mappedModel := mapMonitorToModel(ctx, monitor, resp)
	if resp.Diagnostics.HasError() {
		return
	}
	config.monitorModel = mappedModel

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
