package resources

import (
	"context"
	"regexp"
	"strconv"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// ThemeColorsModel represents color values for a theme (light or dark)
type ThemeColorsModel struct {
	Operational         types.String `tfsdk:"operational"`
	DegradedPerformance types.String `tfsdk:"degraded_performance"`
	PartialOutage       types.String `tfsdk:"partial_outage"`
	MajorOutage         types.String `tfsdk:"major_outage"`
	Maintenance         types.String `tfsdk:"maintenance"`
	Empty               types.String `tfsdk:"empty"`
	Background          types.String `tfsdk:"background"`
	Foreground          types.String `tfsdk:"foreground"`
	ForegroundMuted     types.String `tfsdk:"foreground_muted"`
	BackgroundCard      types.String `tfsdk:"background_card"`
}

// ThemeModel represents the theme configuration for status pages
type ThemeModel struct {
	Light       types.Object `tfsdk:"light"`
	Dark        types.Object `tfsdk:"dark"`
	Rounded     types.Bool   `tfsdk:"rounded"`
	BorderWidth types.Int64  `tfsdk:"border_width"`
}

// ComponentModel represents a component in the status page
type ComponentModel struct {
	ComponentableType types.String `tfsdk:"componentable_type"`
	ComponentableID   types.Int64  `tfsdk:"componentable_id"`
}

// UptimeStatusPageModel defines the data model for the status page resource
type UptimeStatusPageModel struct {
	ColorScheme          types.String  `tfsdk:"color_scheme"`
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
	Theme                types.Object  `tfsdk:"theme"`
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
			"color_scheme": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("all"),
				Description:         "Available color schemes for the status page (all, dark, or light). Defaults to all.",
				MarkdownDescription: "Available color schemes for the status page (all, dark, or light). Defaults to all.",
				Validators: []validator.String{
					stringvalidator.OneOf("all", "dark", "light"),
				},
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
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
			},
			"domain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Custom domain for the status page",
				MarkdownDescription: "Custom domain for the status page, [see docs](https://docs.phare.io/uptime/status-pages#custom-domain)",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
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
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
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
				MarkdownDescription: "Subdomain for the status page, [see docs](https://docs.phare.io/uptime/status-pages#phare-domain)",
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
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
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
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
				PlanModifiers: []planmodifier.String{
					helpers.TrimString(),
				},
			},
			"project_scope": schema.DynamicAttribute{
				Description: "Optional. Project scope for this resource. " +
					"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
					"Overrides the provider-level project_scope if set. " +
					"Required when using an organization-scoped API key (starting with pha_org_).",
				Optional: true,
			},
		},
		Blocks: map[string]schema.Block{
			"theme": schema.SingleNestedBlock{
				Attributes: map[string]schema.Attribute{
					"rounded": schema.BoolAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "Whether to use rounded corners",
						MarkdownDescription: "Whether to use rounded corners",
					},
					"border_width": schema.Int64Attribute{
						Optional:            true,
						Computed:            true,
						Description:         "Border width (0-3)",
						MarkdownDescription: "Border width (0-3)",
						Validators: []validator.Int64{
							int64validator.Between(0, 3),
						},
					},
				},
				Blocks: map[string]schema.Block{
					"light": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"operational": schema.StringAttribute{
								Required:            true,
								Description:         "Color for operational status (e.g., #16a34a)",
								MarkdownDescription: "Color for operational status (e.g., #16a34a)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #16a34a)",
									),
								},
							},
							"degraded_performance": schema.StringAttribute{
								Required:            true,
								Description:         "Color for degraded performance status (e.g., #fbbf24)",
								MarkdownDescription: "Color for degraded performance status (e.g., #fbbf24)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #fbbf24)",
									),
								},
							},
							"partial_outage": schema.StringAttribute{
								Required:            true,
								Description:         "Color for partial outage status (e.g., #f59e0b)",
								MarkdownDescription: "Color for partial outage status (e.g., #f59e0b)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #f59e0b)",
									),
								},
							},
							"major_outage": schema.StringAttribute{
								Required:            true,
								Description:         "Color for major outage status (e.g., #ef4444)",
								MarkdownDescription: "Color for major outage status (e.g., #ef4444)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #ef4444)",
									),
								},
							},
							"maintenance": schema.StringAttribute{
								Required:            true,
								Description:         "Color for maintenance status (e.g., #6366f1)",
								MarkdownDescription: "Color for maintenance status (e.g., #6366f1)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #6366f1)",
									),
								},
							},
							"empty": schema.StringAttribute{
								Required:            true,
								Description:         "Color for empty/no data status (e.g., #d3d3d3)",
								MarkdownDescription: "Color for empty/no data status (e.g., #d3d3d3)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #d3d3d3)",
									),
								},
							},
							"background": schema.StringAttribute{
								Required:            true,
								Description:         "Background color (e.g., #ffffff)",
								MarkdownDescription: "Background color (e.g., #ffffff)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #ffffff)",
									),
								},
							},
							"foreground": schema.StringAttribute{
								Required:            true,
								Description:         "Foreground/text color (e.g., #000000)",
								MarkdownDescription: "Foreground/text color (e.g., #000000)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #000000)",
									),
								},
							},
							"foreground_muted": schema.StringAttribute{
								Required:            true,
								Description:         "Muted foreground/text color (e.g., #737373)",
								MarkdownDescription: "Muted foreground/text color (e.g., #737373)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #737373)",
									),
								},
							},
							"background_card": schema.StringAttribute{
								Required:            true,
								Description:         "Card background color (e.g., #fafafa)",
								MarkdownDescription: "Card background color (e.g., #fafafa)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #fafafa)",
									),
								},
							},
						},
						Description:         "Light theme colors",
						MarkdownDescription: "Light theme colors",
					},
					"dark": schema.SingleNestedBlock{
						Attributes: map[string]schema.Attribute{
							"operational": schema.StringAttribute{
								Required:            true,
								Description:         "Color for operational status (e.g., #16a34a)",
								MarkdownDescription: "Color for operational status (e.g., #16a34a)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #16a34a)",
									),
								},
							},
							"degraded_performance": schema.StringAttribute{
								Required:            true,
								Description:         "Color for degraded performance status (e.g., #fbbf24)",
								MarkdownDescription: "Color for degraded performance status (e.g., #fbbf24)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #fbbf24)",
									),
								},
							},
							"partial_outage": schema.StringAttribute{
								Required:            true,
								Description:         "Color for partial outage status (e.g., #f59e0b)",
								MarkdownDescription: "Color for partial outage status (e.g., #f59e0b)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #f59e0b)",
									),
								},
							},
							"major_outage": schema.StringAttribute{
								Required:            true,
								Description:         "Color for major outage status (e.g., #ef4444)",
								MarkdownDescription: "Color for major outage status (e.g., #ef4444)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #ef4444)",
									),
								},
							},
							"maintenance": schema.StringAttribute{
								Required:            true,
								Description:         "Color for maintenance status (e.g., #6366f1)",
								MarkdownDescription: "Color for maintenance status (e.g., #6366f1)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #6366f1)",
									),
								},
							},
							"empty": schema.StringAttribute{
								Required:            true,
								Description:         "Color for empty/no data status (e.g., #d3d3d3)",
								MarkdownDescription: "Color for empty/no data status (e.g., #d3d3d3)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #d3d3d3)",
									),
								},
							},
							"background": schema.StringAttribute{
								Required:            true,
								Description:         "Background color (e.g., #111111)",
								MarkdownDescription: "Background color (e.g., #111111)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #111111)",
									),
								},
							},
							"foreground": schema.StringAttribute{
								Required:            true,
								Description:         "Foreground/text color (e.g., #ffffff)",
								MarkdownDescription: "Foreground/text color (e.g., #ffffff)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #ffffff)",
									),
								},
							},
							"foreground_muted": schema.StringAttribute{
								Required:            true,
								Description:         "Muted foreground/text color (e.g., #959595)",
								MarkdownDescription: "Muted foreground/text color (e.g., #959595)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #959595)",
									),
								},
							},
							"background_card": schema.StringAttribute{
								Required:            true,
								Description:         "Card background color (e.g., #1a1a1a)",
								MarkdownDescription: "Card background color (e.g., #1a1a1a)",
								Validators: []validator.String{
									stringvalidator.RegexMatches(
										regexp.MustCompile(`^#[0-9a-fA-F]{6}$`),
										"must be a valid hexadecimal color code (e.g., #1a1a1a)",
									),
								},
							},
						},
						Description:         "Dark theme colors",
						MarkdownDescription: "Dark theme colors",
					},
				},
				Description:         "Theme settings to customize the status page",
				MarkdownDescription: "Theme settings to customize the status page",
			},
		},
	}
}

// ThemeColorsModelAttrTypes defines the attribute types for ThemeColorsModel
var ThemeColorsModelAttrTypes = map[string]attr.Type{
	"operational":          types.StringType,
	"degraded_performance": types.StringType,
	"partial_outage":       types.StringType,
	"major_outage":         types.StringType,
	"maintenance":          types.StringType,
	"empty":                types.StringType,
	"background":           types.StringType,
	"foreground":           types.StringType,
	"foreground_muted":     types.StringType,
	"background_card":      types.StringType,
}

// ThemeModelAttrTypes defines the attribute types for ThemeModel
var ThemeModelAttrTypes = map[string]attr.Type{
	"light":        types.ObjectType{AttrTypes: ThemeColorsModelAttrTypes},
	"dark":         types.ObjectType{AttrTypes: ThemeColorsModelAttrTypes},
	"rounded":      types.BoolType,
	"border_width": types.Int64Type,
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

// Helper function to convert Terraform theme colors model to client theme colors
func themeColorsModelToClient(colors ThemeColorsModel) *client.ThemeColors {
	return &client.ThemeColors{
		Operational:         colors.Operational.ValueString(),
		DegradedPerformance: colors.DegradedPerformance.ValueString(),
		PartialOutage:       colors.PartialOutage.ValueString(),
		MajorOutage:         colors.MajorOutage.ValueString(),
		Maintenance:         colors.Maintenance.ValueString(),
		Empty:               colors.Empty.ValueString(),
		Background:          colors.Background.ValueString(),
		Foreground:          colors.Foreground.ValueString(),
		ForegroundMuted:     colors.ForegroundMuted.ValueString(),
		BackgroundCard:      colors.BackgroundCard.ValueString(),
	}
}

// Helper function to convert client theme colors to Terraform theme colors model
func clientToThemeColorsModel(colors *client.ThemeColors) ThemeColorsModel {
	return ThemeColorsModel{
		Operational:         types.StringValue(colors.Operational),
		DegradedPerformance: types.StringValue(colors.DegradedPerformance),
		PartialOutage:       types.StringValue(colors.PartialOutage),
		MajorOutage:         types.StringValue(colors.MajorOutage),
		Maintenance:         types.StringValue(colors.Maintenance),
		Empty:               types.StringValue(colors.Empty),
		Background:          types.StringValue(colors.Background),
		Foreground:          types.StringValue(colors.Foreground),
		ForegroundMuted:     types.StringValue(colors.ForegroundMuted),
		BackgroundCard:      types.StringValue(colors.BackgroundCard),
	}
}

// Helper function to map theme colors to attribute values
func mapThemeColorsToAttrs(colors ThemeColorsModel) map[string]attr.Value {
	return map[string]attr.Value{
		"operational":          colors.Operational,
		"degraded_performance": colors.DegradedPerformance,
		"partial_outage":       colors.PartialOutage,
		"major_outage":         colors.MajorOutage,
		"maintenance":          colors.Maintenance,
		"empty":                colors.Empty,
		"background":           colors.Background,
		"foreground":           colors.Foreground,
		"foreground_muted":     colors.ForegroundMuted,
		"background_card":      colors.BackgroundCard,
	}
}

// Helper function to map theme from API response to Terraform state
func mapThemeFromAPIResponse(ctx context.Context, apiTheme *client.StatusPageTheme, diagnostics *diag.Diagnostics) types.Object {
	if apiTheme == nil {
		return types.ObjectNull(ThemeModelAttrTypes)
	}

	themeAttrs := map[string]attr.Value{
		"light":        types.ObjectNull(ThemeColorsModelAttrTypes),
		"dark":         types.ObjectNull(ThemeColorsModelAttrTypes),
		"rounded":      types.BoolNull(),
		"border_width": types.Int64Null(),
	}

	// Map light theme colors
	if apiTheme.Light != nil {
		lightColors := clientToThemeColorsModel(apiTheme.Light)
		lightObj, objDiags := types.ObjectValue(ThemeColorsModelAttrTypes, mapThemeColorsToAttrs(lightColors))
		diagnostics.Append(objDiags...)
		if diagnostics.HasError() {
			return types.ObjectNull(ThemeModelAttrTypes)
		}
		themeAttrs["light"] = lightObj
	}

	// Map dark theme colors
	if apiTheme.Dark != nil {
		darkColors := clientToThemeColorsModel(apiTheme.Dark)
		darkObj, objDiags := types.ObjectValue(ThemeColorsModelAttrTypes, mapThemeColorsToAttrs(darkColors))
		diagnostics.Append(objDiags...)
		if diagnostics.HasError() {
			return types.ObjectNull(ThemeModelAttrTypes)
		}
		themeAttrs["dark"] = darkObj
	}

	// Map rounded and border width
	if apiTheme.Rounded != nil {
		themeAttrs["rounded"] = types.BoolValue(*apiTheme.Rounded)
	}
	if apiTheme.BorderWidth != nil {
		themeAttrs["border_width"] = types.Int64Value(*apiTheme.BorderWidth)
	}

	themeObj, objDiags := types.ObjectValue(ThemeModelAttrTypes, themeAttrs)
	diagnostics.Append(objDiags...)
	return themeObj
}

// Helper function to extract theme from plan and convert to client format
func extractThemeFromPlan(ctx context.Context, planTheme types.Object, diagnostics *diag.Diagnostics) *client.StatusPageTheme {
	if planTheme.IsNull() || planTheme.IsUnknown() {
		return nil
	}

	themeStruct := struct {
		Light       types.Object `tfsdk:"light"`
		Dark        types.Object `tfsdk:"dark"`
		Rounded     types.Bool   `tfsdk:"rounded"`
		BorderWidth types.Int64  `tfsdk:"border_width"`
	}{}

	diagsObj := planTheme.As(ctx, &themeStruct, basetypes.ObjectAsOptions{})
	diagnostics.Append(diagsObj...)
	if diagnostics.HasError() {
		return nil
	}

	theme := &client.StatusPageTheme{}

	// Extract light theme colors
	if !themeStruct.Light.IsNull() && !themeStruct.Light.IsUnknown() {
		var lightColors ThemeColorsModel
		diagsObj := themeStruct.Light.As(ctx, &lightColors, basetypes.ObjectAsOptions{})
		diagnostics.Append(diagsObj...)
		if diagnostics.HasError() {
			return nil
		}
		theme.Light = themeColorsModelToClient(lightColors)
	}

	// Extract dark theme colors
	if !themeStruct.Dark.IsNull() && !themeStruct.Dark.IsUnknown() {
		var darkColors ThemeColorsModel
		diagsObj := themeStruct.Dark.As(ctx, &darkColors, basetypes.ObjectAsOptions{})
		diagnostics.Append(diagsObj...)
		if diagnostics.HasError() {
			return nil
		}
		theme.Dark = themeColorsModelToClient(darkColors)
	}

	// Extract rounded and border width
	if !themeStruct.Rounded.IsNull() {
		rounded := themeStruct.Rounded.ValueBool()
		theme.Rounded = &rounded
	}
	if !themeStruct.BorderWidth.IsNull() {
		borderWidth := themeStruct.BorderWidth.ValueInt64()
		theme.BorderWidth = &borderWidth
	}

	return theme
}

// Helper function to map components from API response
func mapComponentsFromAPIResponse(ctx context.Context, apiComponents []client.StatusPageComponent, diagnostics *diag.Diagnostics) types.List {
	if len(apiComponents) == 0 {
		return types.ListNull(types.ObjectType{AttrTypes: ComponentModelAttrTypes})
	}

	componentModels := make([]ComponentModel, len(apiComponents))
	for i, comp := range apiComponents {
		componentModels[i] = ComponentModel{
			ComponentableType: types.StringValue(comp.ComponentableType),
			ComponentableID:   types.Int64Value(comp.ComponentableID),
		}
	}

	componentType := types.ObjectType{AttrTypes: ComponentModelAttrTypes}
	componentsList, diagsObj := types.ListValueFrom(ctx, componentType, componentModels)
	diagnostics.Append(diagsObj...)
	return componentsList
}

// Helper function to map subscription channels from API response
func mapSubscriptionChannelsFromAPIResponse(apiChannels []string, diagnostics *diag.Diagnostics) types.List {
	if len(apiChannels) == 0 {
		return types.ListNull(types.StringType)
	}

	channelsElements := make([]attr.Value, len(apiChannels))
	for i, channel := range apiChannels {
		channelsElements[i] = types.StringValue(channel)
	}

	channelsList, diagsObj := types.ListValue(types.StringType, channelsElements)
	diagnostics.Append(diagsObj...)
	return channelsList
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

	// Extract and convert theme from plan
	theme := extractThemeFromPlan(ctx, plan.Theme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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
	colorScheme := plan.ColorScheme.ValueString()
	apiReq := &client.StatusPageRequest{
		Name:                 plan.Name.ValueString(),
		Subdomain:            &subdomain,
		Title:                plan.Title.ValueString(),
		Description:          plan.Description.ValueString(),
		SearchEngineIndexed:  plan.SearchEngineIndexed.ValueBool(),
		WebsiteURL:           plan.WebsiteUrl.ValueString(),
		ColorScheme:          &colorScheme,
		Theme:                theme,
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

	// Map color_scheme from API response
	if apiResp.ColorScheme != nil {
		plan.ColorScheme = types.StringValue(*apiResp.ColorScheme)
	}

	// Map theme, components, and subscription channels from API response using helper functions
	plan.Theme = mapThemeFromAPIResponse(ctx, apiResp.Theme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Components = mapComponentsFromAPIResponse(ctx, apiResp.Components, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.SubscriptionChannels = mapSubscriptionChannelsFromAPIResponse(apiResp.SubscriptionChannels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
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

	// Map color_scheme from API response
	if apiResp.ColorScheme != nil {
		state.ColorScheme = types.StringValue(*apiResp.ColorScheme)
	}

	// Map theme, components, and subscription channels from API response using helper functions
	state.Theme = mapThemeFromAPIResponse(ctx, apiResp.Theme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state.Components = mapComponentsFromAPIResponse(ctx, apiResp.Components, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	state.SubscriptionChannels = mapSubscriptionChannelsFromAPIResponse(apiResp.SubscriptionChannels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
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

	// Extract and convert theme from plan
	theme := extractThemeFromPlan(ctx, plan.Theme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

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
	colorScheme := plan.ColorScheme.ValueString()
	apiReq := &client.StatusPageRequest{
		Name:                 plan.Name.ValueString(),
		Subdomain:            &subdomain,
		Title:                plan.Title.ValueString(),
		Description:          plan.Description.ValueString(),
		SearchEngineIndexed:  plan.SearchEngineIndexed.ValueBool(),
		WebsiteURL:           plan.WebsiteUrl.ValueString(),
		ColorScheme:          &colorScheme,
		Theme:                theme,
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

	// Map color_scheme from API response
	if apiResp.ColorScheme != nil {
		plan.ColorScheme = types.StringValue(*apiResp.ColorScheme)
	}

	// Map theme, components, and subscription channels from API response using helper functions
	plan.Theme = mapThemeFromAPIResponse(ctx, apiResp.Theme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.Components = mapComponentsFromAPIResponse(ctx, apiResp.Components, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	plan.SubscriptionChannels = mapSubscriptionChannelsFromAPIResponse(apiResp.SubscriptionChannels, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
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
