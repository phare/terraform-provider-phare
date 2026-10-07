package helpers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/dynamicplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// BaseConfig provides common configuration for resources and data sources.
type BaseConfig struct {
	client *client.Client
}

// Configure sets up the base configuration with the provided API client.
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

// GetClient returns the configured API client.
func (b *BaseConfig) GetClient() *client.Client {
	return b.client
}

// ValidateExactlyOneOf validates that exactly one of two boolean conditions is true.
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

// ValidateProjectScopeAtPlanTime validates that a project scope is configured either on the resource or provider level.
func ValidateProjectScopeAtPlanTime(
	ctx context.Context,
	client *client.Client,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) {
	if client.IsProjectScoped() {
		return
	}

	if projectScope.IsNull() || projectScope.IsUnknown() {
		projectID, projectSlug := client.GetProjectScope()
		if projectID == "" && projectSlug == "" {
			diagnostics.AddError(
				"Missing Project Scope",
				fmt.Sprintf("Project scope must be specified for %s either at resource level or provider level.", resourceType),
			)
		}
	}
}

// ConfigureResourceWithProjectScope returns a client configured with the appropriate project scope.
func ConfigureResourceWithProjectScope(
	ctx context.Context,
	baseClient *client.Client,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) *client.Client {
	if !projectScope.IsNull() && !projectScope.IsUnknown() {
		scopeValue := GetDynamicStringValue(projectScope)
		if scopeValue != "" {
			if projectID, err := strconv.Atoi(scopeValue); err == nil {
				return createScopedClient(baseClient, strconv.Itoa(projectID), "", resourceType, diagnostics)
			}
			return createScopedClient(baseClient, "", scopeValue, resourceType, diagnostics)
		}
	}

	return baseClient
}

// ProjectScopeAttribute returns the schema definition for project_scope with RequiresReplace attached.
func ProjectScopeAttribute() schema.DynamicAttribute {
	return schema.DynamicAttribute{
		Description: "Optional. Project scope for this resource. " +
			"Accepts either a numeric project ID (e.g., 123) or a string project slug (e.g., \"my-project\"). " +
			"Overrides the provider-level project_scope if set. " +
			"Required when using an organization-scoped API key (starting with pha_org_).",
		Optional: true,
		PlanModifiers: []planmodifier.Dynamic{
			dynamicplanmodifier.RequiresReplace(),
		},
	}
}

// CheckProjectScopeRequiresReplace compares the prior state and planned project_scope values
// and appends path.Root("project_scope") to resp.RequiresReplace if they differ.
func CheckProjectScopeRequiresReplace(
	ctx context.Context,
	req resource.ModifyPlanRequest,
	resp *resource.ModifyPlanResponse,
) {
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var stateScope types.Dynamic
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("project_scope"), &stateScope)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var planScope types.Dynamic
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("project_scope"), &planScope)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if GetDynamicStringValue(stateScope) != GetDynamicStringValue(planScope) {
		resp.RequiresReplace = append(resp.RequiresReplace, path.Root("project_scope"))
	}
}

// ValidateNoProjectScopeChange ensures project_scope was not modified during Update.
func ValidateNoProjectScopeChange(
	ctx context.Context,
	reqState tfsdk.State,
	reqPlan tfsdk.Plan,
	diagnostics *diag.Diagnostics,
) bool {
	var stateScope types.Dynamic
	diagnostics.Append(reqState.GetAttribute(ctx, path.Root("project_scope"), &stateScope)...)
	if diagnostics.HasError() {
		return false
	}

	var planScope types.Dynamic
	diagnostics.Append(reqPlan.GetAttribute(ctx, path.Root("project_scope"), &planScope)...)
	if diagnostics.HasError() {
		return false
	}

	if GetDynamicStringValue(stateScope) != GetDynamicStringValue(planScope) {
		diagnostics.AddError(
			"Cannot Change Project Scope",
			"Changing project_scope is not supported and requires replacing the resource.",
		)
		return false
	}

	return true
}

// ImportStateWithProjectScope imports a resource using either "<id>" or "<project_scope>/<id>".
func ImportStateWithProjectScope(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
	apiClient *client.Client,
	resourceTypeName string,
	requireProjectScope bool,
) {
	parts := strings.Split(req.ID, "/")

	var idStr, projectScopeStr string
	switch len(parts) {
	case 1:
		idStr = parts[0]
	case 2:
		projectScopeStr = strings.TrimSpace(parts[0])
		idStr = parts[1]
	default:
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Invalid import format %q. Expected either \"<id>\" or \"<project_scope>/<id>\".", req.ID),
		)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("%s ID %q must be a valid integer: %s", resourceTypeName, idStr, err.Error()),
		)
		return
	}

	if requireProjectScope && apiClient != nil && !apiClient.IsProjectScoped() {
		projectID, projectSlug := apiClient.GetProjectScope()
		hasProviderScope := projectID != "" || projectSlug != ""
		if !hasProviderScope && projectScopeStr == "" {
			resp.Diagnostics.AddError(
				"Missing Project Scope for Import",
				fmt.Sprintf("When using an organization-scoped API key without provider-level project_scope, "+
					"the import ID must include the project scope: \"<project_scope>/<%s_id>\" "+
					"(e.g., \"my-project/123\" or \"456/123\").", strings.ToLower(resourceTypeName)),
			)
			return
		}
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), id)...)

	if projectScopeStr != "" {
		if scopeInt, err := strconv.ParseInt(projectScopeStr, 10, 64); err == nil {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_scope"), types.DynamicValue(types.Int64Value(scopeInt)))...)
		} else {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_scope"), types.DynamicValue(types.StringValue(projectScopeStr)))...)
		}
	}
}

// GetDynamicStringValue extracts a string representation from types.Dynamic (Int64, Number, or String).
func GetDynamicStringValue(dynamicVal types.Dynamic) string {
	if dynamicVal.IsNull() || dynamicVal.IsUnknown() {
		return ""
	}

	switch v := dynamicVal.UnderlyingValue().(type) {
	case types.Int64:
		if !v.IsNull() {
			return strconv.FormatInt(v.ValueInt64(), 10)
		}
	case types.Number:
		if !v.IsNull() {
			return v.ValueBigFloat().String()
		}
	case types.String:
		if !v.IsNull() {
			return v.ValueString()
		}
	}

	return ""
}

// StringSliceToList converts a []string to a types.List of StringType.
// Returns a null list if the slice is empty.
func StringSliceToList(values []string, diagnostics *diag.Diagnostics) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}

	elements := make([]attr.Value, len(values))
	for i, v := range values {
		elements[i] = types.StringValue(v)
	}

	list, diags := types.ListValue(types.StringType, elements)
	diagnostics.Append(diags...)
	return list
}

// StringSliceToSet converts a []string to a types.Set of StringType.
// An empty slice produces a null set when the previous value was null
func StringSliceToSet(values []string, previous types.Set, diagnostics *diag.Diagnostics) types.Set {
	if len(values) == 0 {
		if previous.IsNull() {
			return types.SetNull(types.StringType)
		}
		return types.SetValueMust(types.StringType, nil)
	}

	elements := make([]attr.Value, len(values))
	for i, v := range values {
		elements[i] = types.StringValue(v)
	}

	set, diags := types.SetValue(types.StringType, elements)
	diagnostics.Append(diags...)
	return set
}

func createScopedClient(
	baseClient *client.Client,
	projectID string,
	projectSlug string,
	resourceType string,
	diagnostics *diag.Diagnostics,
) *client.Client {
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
