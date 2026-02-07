package helpers

import (
	"context"
	"fmt"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ResourceBase provides common functionality for project-scoped resources.
// Resources should embed this type to inherit common behavior.
type ResourceBase struct {
	BaseConfig
}

// Configure implements common configuration logic for all project-scoped resources.
func (r *ResourceBase) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	apiClient, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	r.BaseConfig.Configure(ctx, apiClient, &resp.Diagnostics)
}

// GetScopedClient returns a client with the appropriate project scope.
// If the resource has a project_scope set, it creates a new client with that scope.
// Otherwise, it returns the provider-level client.
func (r *ResourceBase) GetScopedClient(
	ctx context.Context,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) *client.Client {
	return ConfigureResourceWithProjectScope(ctx, r.GetClient(), projectScope, resourceType, diagnostics)
}

// ValidateProjectScopeAtPlanTime validates project scope during the plan phase.
// This should be called from a resource's ModifyPlan method.
func (r *ResourceBase) ValidateProjectScopeAtPlanTime(
	ctx context.Context,
	projectScope types.Dynamic,
	resourceType string,
	diagnostics *diag.Diagnostics,
) {
	ValidateProjectScopeAtPlanTime(ctx, r.GetClient(), projectScope, resourceType, diagnostics)
}
