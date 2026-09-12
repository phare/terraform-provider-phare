package resources

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func TestProjectResource_Metadata(t *testing.T) {
	r := NewProjectResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_project", resp.TypeName)
}

func TestProjectResource_Schema(t *testing.T) {
	r := NewProjectResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestProjectResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &projectResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.client)
}

func TestProjectResource_NameValidation(t *testing.T) {
	r := NewProjectResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	nameAttr, ok := resp.Schema.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, nameAttr.Validators)

	testCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{
			name:        "valid 1 char",
			val:         types.StringValue("a"),
			expectError: false,
		},
		{
			name:        "valid 25 chars",
			val:         types.StringValue(strings.Repeat("a", 25)),
			expectError: false,
		},
		{
			name:        "valid normal name",
			val:         types.StringValue("my-project"),
			expectError: false,
		},
		{
			name:        "valid name with whitespace padding within limit",
			val:         types.StringValue("   " + strings.Repeat("a", 25) + "   "),
			expectError: false,
		},
		{
			name:        "invalid whitespace only",
			val:         types.StringValue("   "),
			expectError: true,
		},
		{
			name:        "invalid empty string",
			val:         types.StringValue(""),
			expectError: true,
		},
		{
			name:        "invalid 26 chars",
			val:         types.StringValue(strings.Repeat("a", 26)),
			expectError: true,
		},
		{
			name:        "null value skipped",
			val:         types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown value skipped",
			val:         types.StringUnknown(),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("name"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range nameAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestProjectResource_MembersValidation(t *testing.T) {
	r := NewProjectResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	membersAttr, ok := resp.Schema.Attributes["members"].(schema.ListAttribute)
	require.True(t, ok)
	require.NotEmpty(t, membersAttr.Validators)

	makeMembers := func(n int) []attr.Value {
		vals := make([]attr.Value, n)
		for i := 0; i < n; i++ {
			vals[i] = types.Int64Value(int64(i + 1))
		}
		return vals
	}

	testCases := []struct {
		name        string
		val         types.List
		expectError bool
	}{
		{
			name:        "valid 1 member",
			val:         types.ListValueMust(types.Int64Type, makeMembers(1)),
			expectError: false,
		},
		{
			name:        "valid 100 members",
			val:         types.ListValueMust(types.Int64Type, makeMembers(100)),
			expectError: false,
		},
		{
			name:        "invalid empty members",
			val:         types.ListValueMust(types.Int64Type, []attr.Value{}),
			expectError: true,
		},
		{
			name:        "invalid 101 members",
			val:         types.ListValueMust(types.Int64Type, makeMembers(101)),
			expectError: true,
		},
		{
			name: "invalid duplicate members",
			val: types.ListValueMust(types.Int64Type, []attr.Value{
				types.Int64Value(1),
				types.Int64Value(1),
			}),
			expectError: true,
		},
		{
			name:        "null value skipped",
			val:         types.ListNull(types.Int64Type),
			expectError: false,
		},
		{
			name:        "unknown value skipped",
			val:         types.ListUnknown(types.Int64Type),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.ListRequest{
				Path:        path.Root("members"),
				ConfigValue: tc.val,
			}
			valResp := &validator.ListResponse{}

			for _, v := range membersAttr.Validators {
				v.ValidateList(context.Background(), valReq, valResp)
			}

			if tc.expectError {
				require.True(t, valResp.Diagnostics.HasError())
			} else {
				require.False(t, valResp.Diagnostics.HasError())
			}
		})
	}
}

func TestProjectResource_ModifyPlan(t *testing.T) {
	r := NewProjectResource().(*projectResource)

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	t.Run("skip when plan is null (destroy)", func(t *testing.T) {
		req := resource.ModifyPlanRequest{
			Plan: tfsdk.Plan{
				Raw: tftypes.NewValue(tftypes.Object{}, nil),
			},
		}
		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), req, resp)
		require.False(t, resp.Diagnostics.HasError())
	})

	settingsAttrTypes := map[string]attr.Type{
		"incident_merging_time_window": types.Int64Type,
		"use_incident_ai":              types.BoolType,
		"use_incident_merging":         types.BoolType,
	}

	testCases := []struct {
		name        string
		planName    types.String
		expectError bool
	}{
		{
			name:        "valid name",
			planName:    types.StringValue("my-project"),
			expectError: false,
		},
		{
			name:        "whitespace name trimmed to valid length",
			planName:    types.StringValue("  my-project  "),
			expectError: false,
		},
		{
			name:        "unicode name trimmed to valid length",
			planName:    types.StringValue("  Projet Éducation 🚀  "),
			expectError: false,
		},
		{
			name:        "whitespace-only name trimmed to 0 length",
			planName:    types.StringValue("   "),
			expectError: true,
		},
		{
			name:        "empty name",
			planName:    types.StringValue(""),
			expectError: true,
		},
		{
			name:        "name trimmed to over 25 characters",
			planName:    types.StringValue("  " + strings.Repeat("a", 26) + "  "),
			expectError: true,
		},
		{
			name:        "null name skipped",
			planName:    types.StringNull(),
			expectError: false,
		},
		{
			name:        "unknown name skipped",
			planName:    types.StringUnknown(),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			planData := ProjectModel{
				CreatedAt: types.StringNull(),
				Id:        types.Int64Null(),
				Members:   types.ListValueMust(types.Int64Type, []attr.Value{types.Int64Value(1)}),
				Name:      tc.planName,
				Settings:  types.ObjectNull(settingsAttrTypes),
				Slug:      types.StringNull(),
				UpdatedAt: types.StringNull(),
			}

			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			diags := plan.Set(context.Background(), planData)
			require.False(t, diags.HasError())

			resp := &resource.ModifyPlanResponse{}
			r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

			if tc.expectError {
				require.True(t, resp.Diagnostics.HasError())
				foundNameErr := false
				for _, diagErr := range resp.Diagnostics.Errors() {
					if diagErr.Summary() == "Invalid Name Length" {
						foundNameErr = true
					}
				}
				require.True(t, foundNameErr)
			} else {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}
