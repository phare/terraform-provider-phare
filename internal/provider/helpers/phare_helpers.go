package helpers

import (
	"context"
	"fmt"
	"strconv"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BaseConfig provides common configuration for both resources and data sources
type BaseConfig struct {
	client *client.Client
}

// Configure sets up the base configuration
func (b *BaseConfig) Configure(ctx context.Context, client *client.Client, diagnostics *diag.Diagnostics) {
	if client == nil {
		diagnostics.AddError(
			"Client Configuration Error",
			"Provider data cannot be nil",
		)
		return
	}
	b.client = client
}

// GetClient returns the configured client
func (b *BaseConfig) GetClient() *client.Client {
	return b.client
}

// ValidateExactlyOneOf validates that exactly one of two boolean conditions is true
func ValidateExactlyOneOf(diagnostics *diag.Diagnostics, hasField1, hasField2 bool, resourceName string) bool {
	if !hasField1 && !hasField2 {
		diagnostics.AddError(
			"Missing Required Attribute",
			fmt.Sprintf("Exactly one of the attributes must be specified for %s.", resourceName),
		)
		return false
	}

	if hasField1 && hasField2 {
		diagnostics.AddError(
			"Conflicting Attributes",
			fmt.Sprintf("Only one of the attributes can be specified for %s, not both.", resourceName),
		)
		return false
	}

	return true
}

// ValidateProjectScopeAtPlanTime validates project scope during the plan phase
func ValidateProjectScopeAtPlanTime(
	ctx context.Context,
	client *client.Client,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) {
	// If projectScope is null or unknown, we need to check if the client has a project scope
	if projectScope.IsNull() || projectScope.IsUnknown() {
		// Check if the client has a project scope configured
		projectID, projectSlug := client.GetProjectScope()
		if projectID == "" && projectSlug == "" {
			diagnostics.AddError(
				"Missing Project Scope",
				fmt.Sprintf("Project scope must be specified for %s either at resource level or provider level.", resourceType),
			)
		}
		return
	}

	// For now, just check that projectScope is not null/unknown
	// The actual validation of the scope value will happen when we try to use it
	// This is a simplified approach to get things working
}

// ConfigureResourceWithProjectScope returns a client configured with the appropriate project scope
func ConfigureResourceWithProjectScope(
	ctx context.Context,
	baseClient *client.Client,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) *client.Client {
	// If projectScope is set at resource level, use it to create a new client
	if !projectScope.IsNull() && !projectScope.IsUnknown() {
		scopeValue := getDynamicStringValue(projectScope)
		if scopeValue != "" {
			// Try to parse as integer first (project ID), then treat as string (project slug)
			if projectID, parseErr := strconv.Atoi(scopeValue); parseErr == nil {
				// It's a project ID - create new client with this project ID
				return createScopedClient(baseClient, strconv.Itoa(projectID), "", resourceType, diagnostics)
			} else {
				// It's a project slug - create new client with this project slug
				return createScopedClient(baseClient, "", scopeValue, resourceType, diagnostics)
			}
		}
	}

	// Otherwise, use the base client as-is (it already has the provider-level project scope)
	return baseClient
}

// getDynamicStringValue extracts a string value from types.Dynamic
func getDynamicStringValue(dynamicVal types.Dynamic) string {
	if dynamicVal.IsNull() || dynamicVal.IsUnknown() {
		return ""
	}

	underlyingValue := dynamicVal.UnderlyingValue()

	if intVal, ok := underlyingValue.(types.Int64); ok && !intVal.IsNull() {
		return fmt.Sprintf("%d", intVal.ValueInt64())
	} else if strVal, ok := underlyingValue.(types.String); ok && !strVal.IsNull() {
		return strVal.ValueString()
	}

	return ""
}

// createScopedClient creates a new client with the specified project scope
func createScopedClient(
	baseClient *client.Client,
	projectID string,
	projectSlug string,
	resourceType string,
	diagnostics *diag.Diagnostics,
) *client.Client {
	// Use the new WithProjectScope method to create a scoped client
	scopedClient, err := baseClient.WithProjectScope(projectID, projectSlug)
	if err != nil {
		diagnostics.AddError(
			"Failed to create scoped client",
			fmt.Sprintf("Unable to create client with project scope for %s: %s", resourceType, err.Error()),
		)
		return baseClient
	}

	return scopedClient
}
