package helpers

import (
	"context"
	"testing"
	"time"

	"terraform-provider-phare/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

type testResourceModel struct {
	Id           types.Int64   `tfsdk:"id"`
	ProjectScope types.Dynamic `tfsdk:"project_scope"`
}

func testResourceSchema() schema.Schema {
	return schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed: true,
			},
			"project_scope": ProjectScopeAttribute(),
		},
	}
}

func TestProjectScopeAttribute(t *testing.T) {
	attr := ProjectScopeAttribute()
	require.True(t, attr.Optional)
	require.NotEmpty(t, attr.PlanModifiers)
	require.Contains(t, attr.Description, "Project scope for this resource")
}

func TestCheckProjectScopeRequiresReplace(t *testing.T) {
	ctx := context.Background()
	sch := testResourceSchema()

	t.Run("state is null (resource creation) does not require replace", func(t *testing.T) {
		plan := tfsdk.Plan{Schema: sch}
		diags := plan.Set(ctx, &testResourceModel{
			ProjectScope: types.DynamicValue(types.StringValue("my-proj")),
		})
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		CheckProjectScopeRequiresReplace(ctx, resource.ModifyPlanRequest{
			Plan: plan,
		}, resp)

		require.False(t, resp.Diagnostics.HasError())
		require.False(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
	})

	t.Run("state and plan project_scope match does not require replace", func(t *testing.T) {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("my-proj")),
		})
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: sch}
		diags = plan.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("my-proj")),
		})
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		CheckProjectScopeRequiresReplace(ctx, resource.ModifyPlanRequest{
			State: state,
			Plan:  plan,
		}, resp)

		require.False(t, resp.Diagnostics.HasError())
		require.False(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
	})

	t.Run("state and plan project_scope differ triggers requires replace", func(t *testing.T) {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("old-proj")),
		})
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: sch}
		diags = plan.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("new-proj")),
		})
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		CheckProjectScopeRequiresReplace(ctx, resource.ModifyPlanRequest{
			State: state,
			Plan:  plan,
		}, resp)

		require.False(t, resp.Diagnostics.HasError())
		require.True(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
	})

	t.Run("state numeric and plan string project_scope differ triggers requires replace", func(t *testing.T) {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.Int64Value(123)),
		})
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: sch}
		diags = plan.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.Int64Value(456)),
		})
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		CheckProjectScopeRequiresReplace(ctx, resource.ModifyPlanRequest{
			State: state,
			Plan:  plan,
		}, resp)

		require.False(t, resp.Diagnostics.HasError())
		require.True(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
	})
}

func TestValidateNoProjectScopeChange(t *testing.T) {
	ctx := context.Background()
	sch := testResourceSchema()

	t.Run("scopes match returns true", func(t *testing.T) {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("proj-a")),
		})
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: sch}
		diags = plan.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("proj-a")),
		})
		require.False(t, diags.HasError())

		resp := &resource.UpdateResponse{}
		ok := ValidateNoProjectScopeChange(ctx, state, plan, &resp.Diagnostics)
		require.True(t, ok)
		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("scopes differ returns false with error diagnostic", func(t *testing.T) {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("proj-a")),
		})
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: sch}
		diags = plan.Set(ctx, &testResourceModel{
			Id:           types.Int64Value(1),
			ProjectScope: types.DynamicValue(types.StringValue("proj-b")),
		})
		require.False(t, diags.HasError())

		resp := &resource.UpdateResponse{}
		ok := ValidateNoProjectScopeChange(ctx, state, plan, &resp.Diagnostics)
		require.False(t, ok)
		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Cannot Change Project Scope", resp.Diagnostics.Errors()[0].Summary())
	})
}

func TestImportStateWithProjectScope(t *testing.T) {
	ctx := context.Background()
	sch := testResourceSchema()

	newTestState := func() tfsdk.State {
		state := tfsdk.State{Schema: sch}
		diags := state.Set(ctx, &testResourceModel{})
		require.False(t, diags.HasError())
		return state
	}

	t.Run("import with integer ID only on project-scoped client", func(t *testing.T) {
		projectClient, err := client.NewClient("https://api.phare.io", "pha_proj_123", 10*time.Second, "123", "", "1.0", "1.0", true)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "456"}, resp, projectClient, "TestResource", true)

		require.False(t, resp.Diagnostics.HasError())
		var model testResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(456), model.Id.ValueInt64())
	})

	t.Run("import with project_scope/id slug on org-scoped client", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "my-project/789"}, resp, orgClient, "TestResource", true)

		require.False(t, resp.Diagnostics.HasError())
		var model testResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
		require.Equal(t, "my-project", GetDynamicStringValue(model.ProjectScope))
	})

	t.Run("import with project_scope/id integer on org-scoped client", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "123/789"}, resp, orgClient, "TestResource", true)

		require.False(t, resp.Diagnostics.HasError())
		var model testResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
		require.Equal(t, "123", GetDynamicStringValue(model.ProjectScope))
	})

	t.Run("import without project scope on org-scoped client with no provider scope returns error when required", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "789"}, resp, orgClient, "TestResource", true)

		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Missing Project Scope for Import", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("import without project scope on org-scoped client when requireProjectScope is false succeeds", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "789"}, resp, orgClient, "Alert rule", false)

		require.False(t, resp.Diagnostics.HasError())
		var model testResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
	})

	t.Run("import with provider-level scope on org-scoped client succeeds without scope in ID", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "123", "", "1.0", "1.0", false)
		require.NoError(t, err)

		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "789"}, resp, orgClient, "TestResource", true)

		require.False(t, resp.Diagnostics.HasError())
		var model testResourceModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
	})

	t.Run("invalid import format returns error", func(t *testing.T) {
		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "a/b/c"}, resp, nil, "TestResource", true)

		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Invalid Import ID", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("non-integer ID returns error", func(t *testing.T) {
		resp := &resource.ImportStateResponse{State: newTestState()}
		ImportStateWithProjectScope(ctx, resource.ImportStateRequest{ID: "not-an-id"}, resp, nil, "TestResource", true)

		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Invalid Import ID", resp.Diagnostics.Errors()[0].Summary())
	})
}

func TestStringSliceToSet(t *testing.T) {
	newSet := func(t *testing.T, values ...string) types.Set {
		s, diags := types.SetValueFrom(context.Background(), types.StringType, values)
		require.False(t, diags.HasError())
		return s
	}
	toSet := func(values []string, previous types.Set) types.Set {
		var diags diag.Diagnostics
		s := StringSliceToSet(values, previous, &diags)
		require.False(t, diags.HasError())
		return s
	}

	t.Run("values produce an order-independent set", func(t *testing.T) {
		set := toSet([]string{"tfacc:tcp-tags-test", "environment:production"}, types.SetNull(types.StringType))
		require.True(t, set.Equal(newSet(t, "environment:production", "tfacc:tcp-tags-test")))
	})

	t.Run("empty values preserve a null previous value", func(t *testing.T) {
		require.True(t, toSet(nil, types.SetNull(types.StringType)).IsNull())
	})

	t.Run("empty values preserve an empty previous value", func(t *testing.T) {
		set := toSet(nil, types.SetValueMust(types.StringType, nil))
		require.False(t, set.IsNull())
		require.Equal(t, 0, len(set.Elements()))
	})

	t.Run("empty values with a previously populated value", func(t *testing.T) {
		set := toSet(nil, newSet(t, "environment:production"))
		require.False(t, set.IsNull())
		require.Equal(t, 0, len(set.Elements()))
	})
}
