package provider

import (
	"context"
	"os"
	"strconv"
	"strings"
	"time"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/datasources"
	"terraform-provider-phare/internal/provider/resources"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ provider.Provider = (*phareProvider)(nil)

// New returns a function that creates a new Phare provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &phareProvider{
			version: version,
		}
	}
}

// phareProvider is the provider implementation.
type phareProvider struct {
	version          string
	terraformVersion string
}

// PhareModel describes the provider configuration data model
type PhareModel struct {
	ApiKey       types.String `tfsdk:"api_key"`
	BaseUrl      types.String `tfsdk:"base_url"`
	ProjectScope types.String `tfsdk:"project_scope"`
	Timeout      types.Int64  `tfsdk:"timeout"`
}

// phareProviderModel describes the provider configuration data.
type phareProviderModel = PhareModel

func PhareProviderSchema(ctx context.Context) schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				Description:         "Phare API key for authentication. Can also be set via PHARE_API_KEY environment variable.",
				MarkdownDescription: "Phare API key for authentication. Can also be set via PHARE_API_KEY environment variable.",
			},
			"base_url": schema.StringAttribute{
				Optional:            true,
				Description:         "Phare API base URL. Defaults to https://api.phare.io. Can be overridden via PHARE_BASE_URL environment variable for testing.",
				MarkdownDescription: "Phare API base URL. Defaults to https://api.phare.io. Can be overridden via PHARE_BASE_URL environment variable for testing.",
			},
			"project_scope": schema.StringAttribute{
				Optional:            true,
				Description:         "Optional. Project scope for API requests when using an organization-scoped API key. Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). Can also be set via PHARE_PROJECT_ID or PHARE_PROJECT_SLUG environment variables.",
				MarkdownDescription: "Optional. Project scope for API requests when using an organization-scoped API key. Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). Can also be set via PHARE_PROJECT_ID or PHARE_PROJECT_SLUG environment variables.",
			},
			"timeout": schema.Int64Attribute{
				Optional:            true,
				Description:         "HTTP client timeout in seconds. Defaults to 30.",
				MarkdownDescription: "HTTP client timeout in seconds. Defaults to 30.",
			},
		},
	}
}

func (p *phareProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = PhareProviderSchema(ctx)
	resp.Schema.Description = "Terraform provider for managing Phare uptime monitoring resources."
}

func (p *phareProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config phareProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	p.terraformVersion = req.TerraformVersion

	apiKey := os.Getenv("PHARE_API_KEY")
	if !config.ApiKey.IsNull() {
		apiKey = config.ApiKey.ValueString()
	}

	if apiKey == "" {
		resp.Diagnostics.AddError(
			"Missing API Key",
			"The provider requires an API key. Set the api_key attribute in the provider configuration "+
				"or set the PHARE_API_KEY environment variable.",
		)
		return
	}

	baseURL := "https://api.phare.io"
	if envURL := os.Getenv("PHARE_BASE_URL"); envURL != "" {
		baseURL = envURL
	}
	if !config.BaseUrl.IsNull() {
		baseURL = config.BaseUrl.ValueString()
	}

	timeout := 30 * time.Second
	if !config.Timeout.IsNull() {
		timeout = time.Duration(config.Timeout.ValueInt64()) * time.Second
	}

	// Environment variables take precedence: PHARE_PROJECT_ID or PHARE_PROJECT_SLUG
	var projectID, projectSlug string

	if envProjectID := os.Getenv("PHARE_PROJECT_ID"); envProjectID != "" {
		projectID = envProjectID
	}
	if envProjectSlug := os.Getenv("PHARE_PROJECT_SLUG"); envProjectSlug != "" {
		projectSlug = envProjectSlug
	}

	// If project_scope is set in config, it overrides env vars
	if !config.ProjectScope.IsNull() && !config.ProjectScope.IsUnknown() {
		projectScopeValue := config.ProjectScope.ValueString()

		// Try to parse as integer first (project ID), then treat as string (project slug)
		if _, parseErr := strconv.Atoi(projectScopeValue); parseErr == nil {
			projectID = projectScopeValue
			projectSlug = "" // Clear slug if ID is set
		} else {
			projectSlug = projectScopeValue
			projectID = "" // Clear ID if slug is set
		}
	}

	// Validate that only one project identifier is provided
	if projectID != "" && projectSlug != "" {
		resp.Diagnostics.AddError(
			"Invalid Project Configuration",
			"Cannot specify both PHARE_PROJECT_ID and PHARE_PROJECT_SLUG environment variables. Please provide only one.",
		)
		return
	}

	// Determine API key type for logging (don't log actual key)
	apiKeyType := "unknown"
	if strings.HasPrefix(apiKey, "pha_org_") {
		apiKeyType = "organization-scoped"
	} else if strings.HasPrefix(apiKey, "pha_") {
		apiKeyType = "project-scoped"
	}

	// Log provider configuration (sanitized)
	tflog.Debug(ctx, "Configuring Phare API client", map[string]interface{}{
		"base_url":          baseURL,
		"timeout_seconds":   timeout.Seconds(),
		"api_key_type":      apiKeyType,
		"provider_version":  p.version,
		"terraform_version": p.terraformVersion,
		"has_project_id":    projectID != "",
		"has_project_slug":  projectSlug != "",
	})

	// Create API client
	apiClient, err := client.NewClient(baseURL, apiKey, timeout, projectID, projectSlug, p.version, p.terraformVersion)
	if err != nil {
		tflog.Error(ctx, "Failed to create Phare API client", map[string]interface{}{
			"error": err.Error(),
		})
		resp.Diagnostics.AddError(
			"Unable to Create Phare API Client",
			"An error occurred when creating the Phare API client:\n\n"+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Phare API client configured successfully")

	// Make client available to resources and data sources
	resp.DataSourceData = apiClient
	resp.ResourceData = apiClient
}

func (p *phareProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "phare"
	resp.Version = p.version
}

func (p *phareProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewAlertRuleDataSource,
		datasources.NewAlertRulesDataSource,
		datasources.NewUptimeMonitorsDataSource,
		datasources.NewUptimeMonitorDataSource,
		datasources.NewUptimeStatusPagesDataSource,
		datasources.NewUptimeStatusPageDataSource,
		datasources.NewProjectDataSource,
		datasources.NewProjectsDataSource,
		datasources.NewUserDataSource,
		datasources.NewUsersDataSource,
		datasources.NewIntegrationDataSource,
		datasources.NewIntegrationsDataSource,
	}
}

func (p *phareProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewAlertRuleResource,
		resources.NewUptimeMonitorHttpResource,
		resources.NewUptimeMonitorTcpResource,
		resources.NewUptimeStatusPageResource,
		resources.NewProjectResource,
	}
}
