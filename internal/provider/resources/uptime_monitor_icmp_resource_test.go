package resources

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"
)

func validIcmpRequestModel() *IcmpRequestModel {
	return &IcmpRequestModel{
		Host: types.StringValue("8.8.8.8"),
	}
}

func TestUptimeMonitorIcmpResource_Metadata(t *testing.T) {
	r := NewUptimeMonitorIcmpResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_monitor_icmp", resp.TypeName)
}

func TestUptimeMonitorIcmpResource_Schema(t *testing.T) {
	r := NewUptimeMonitorIcmpResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
	require.NotNil(t, resp.Schema.Blocks["request"])

	// Verify project_scope has RequiresReplace plan modifier
	projectScopeAttr, ok := resp.Schema.Attributes["project_scope"].(schema.DynamicAttribute)
	require.True(t, ok)
	require.NotEmpty(t, projectScopeAttr.PlanModifiers)
}

func TestUptimeMonitorIcmpResource_NameValidation(t *testing.T) {
	r := NewUptimeMonitorIcmpResource()
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
			name:        "valid 2 chars",
			val:         types.StringValue("ab"),
			expectError: false,
		},
		{
			name:        "valid 45 chars",
			val:         types.StringValue(strings.Repeat("a", 45)),
			expectError: false,
		},
		{
			name:        "valid 45 chars with surrounding whitespace",
			val:         types.StringValue("   " + strings.Repeat("a", 45) + "   "),
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
			name:        "invalid 1 char",
			val:         types.StringValue("a"),
			expectError: true,
		},
		{
			name:        "invalid 46 chars",
			val:         types.StringValue(strings.Repeat("a", 46)),
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

func TestUptimeMonitorIcmpResource_RegionsValidation(t *testing.T) {
	r := NewUptimeMonitorIcmpResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	regionsAttr, ok := resp.Schema.Attributes["regions"].(schema.ListAttribute)
	require.True(t, ok)
	require.NotEmpty(t, regionsAttr.Validators)

	testCases := []struct {
		name        string
		val         types.List
		expectError bool
	}{
		{
			name:        "valid single region",
			val:         types.ListValueMust(types.StringType, []attr.Value{types.StringValue("eu-deu-fra")}),
			expectError: false,
		},
		{
			name: "valid 10 regions",
			val: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("as-jpn-hnd"),
				types.StringValue("as-sgp-sin"),
				types.StringValue("as-tha-bkk"),
				types.StringValue("eu-deu-fra"),
				types.StringValue("eu-fra-cdg"),
				types.StringValue("eu-gbr-lhr"),
				types.StringValue("eu-swe-arn"),
				types.StringValue("ng-nld-ams"),
				types.StringValue("na-mex-mex"),
				types.StringValue("na-usa-iad"),
			}),
			expectError: false,
		},
		{
			name:        "invalid empty list",
			val:         types.ListValueMust(types.StringType, []attr.Value{}),
			expectError: true,
		},
		{
			name: "invalid 11 regions",
			val: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("as-jpn-hnd"),
				types.StringValue("as-sgp-sin"),
				types.StringValue("as-tha-bkk"),
				types.StringValue("eu-deu-fra"),
				types.StringValue("eu-fra-cdg"),
				types.StringValue("eu-gbr-lhr"),
				types.StringValue("eu-swe-arn"),
				types.StringValue("ng-nld-ams"),
				types.StringValue("na-mex-mex"),
				types.StringValue("na-usa-iad"),
				types.StringValue("na-usa-sea"),
			}),
			expectError: true,
		},
		{
			name: "invalid duplicate regions",
			val: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("eu-deu-fra"),
				types.StringValue("eu-deu-fra"),
			}),
			expectError: true,
		},
		{
			name: "invalid region value",
			val: types.ListValueMust(types.StringType, []attr.Value{
				types.StringValue("us-west-invalid"),
			}),
			expectError: true,
		},
		{
			name:        "null value skipped",
			val:         types.ListNull(types.StringType),
			expectError: false,
		},
		{
			name:        "unknown value skipped",
			val:         types.ListUnknown(types.StringType),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.ListRequest{
				Path:        path.Root("regions"),
				ConfigValue: tc.val,
			}
			valResp := &validator.ListResponse{}

			for _, v := range regionsAttr.Validators {
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

func TestUptimeMonitorIcmpResource_RequestValidation(t *testing.T) {
	r := NewUptimeMonitorIcmpResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	reqBlock, ok := resp.Schema.Blocks["request"].(schema.SingleNestedBlock)
	require.True(t, ok)

	hostAttr, ok := reqBlock.Attributes["host"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, hostAttr.Validators)

	t.Run("host validation", func(t *testing.T) {
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
				name:        "valid 255 chars",
				val:         types.StringValue(strings.Repeat("a", 255)),
				expectError: false,
			},
			{
				name:        "valid with surrounding whitespace",
				val:         types.StringValue("   8.8.8.8   "),
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
				name:        "invalid 256 chars",
				val:         types.StringValue(strings.Repeat("a", 256)),
				expectError: true,
			},
			{
				name:        "valid 128 multibyte chars (256 bytes)",
				val:         types.StringValue(strings.Repeat("é", 128)),
				expectError: false,
			},
			{
				name:        "valid 255 multibyte chars (510 bytes)",
				val:         types.StringValue(strings.Repeat("é", 255)),
				expectError: false,
			},
			{
				name:        "invalid 256 multibyte chars (512 bytes)",
				val:         types.StringValue(strings.Repeat("é", 256)),
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
					Path:        path.Root("request").AtName("host"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range hostAttr.Validators {
					v.ValidateString(context.Background(), valReq, valResp)
				}

				if tc.expectError {
					require.True(t, valResp.Diagnostics.HasError())
				} else {
					require.False(t, valResp.Diagnostics.HasError())
				}
			})
		}
	})
}

func TestUptimeMonitorIcmpResource_ModifyPlan(t *testing.T) {
	r := &uptimeMonitorIcmpResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	t.Run("missing request", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: nil,
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundReqErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Missing Request Configuration" {
				foundReqErr = true
			}
		}
		require.True(t, foundReqErr)
	})

	t.Run("whitespace host rejected", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: &IcmpRequestModel{
				Host: types.StringValue("    "),
			},
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Host Length" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("multibyte host with 128 chars (256 bytes) allowed in modify plan", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: &IcmpRequestModel{
				Host: types.StringValue(strings.Repeat("é", 128)),
			},
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("multibyte host with 256 chars rejected in modify plan", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: &IcmpRequestModel{
				Host: types.StringValue(strings.Repeat("é", 256)),
			},
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Host Length" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("whitespace name trimmed under 2 chars", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Name:    types.StringValue("  a  "),
				Regions: types.ListNull(types.StringType),
			},
			Request: validIcmpRequestModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid Name Length" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("region threshold exceeds regions length", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:            types.ListNull(types.StringType),
				Name:            types.StringValue("Valid Monitor"),
				RegionThreshold: types.Int64Value(3),
				Regions: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("eu-fra-cdg"),
				}),
			},
			Request: validIcmpRequestModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Invalid region_threshold" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("project scope change requires replace", func(t *testing.T) {
		stateData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:         types.ListNull(types.StringType),
				Id:           types.Int64Value(100),
				Name:         types.StringValue("Valid Monitor"),
				ProjectScope: types.DynamicValue(types.StringValue("project-a")),
				Regions:      types.ListNull(types.StringType),
			},
			Request: validIcmpRequestModel(),
		}

		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:         types.ListNull(types.StringType),
				Id:           types.Int64Value(100),
				Name:         types.StringValue("Valid Monitor"),
				ProjectScope: types.DynamicValue(types.StringValue("project-b")),
				Regions:      types.ListNull(types.StringType),
			},
			Request: validIcmpRequestModel(),
		}

		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(context.Background(), stateData)
		require.False(t, diags.HasError())

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags = plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{
			State: state,
			Plan:  plan,
		}, resp)

		require.False(t, resp.Diagnostics.HasError())
		require.True(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
	})

	t.Run("valid plan", func(t *testing.T) {
		planData := uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:            types.ListNull(types.StringType),
				Name:            types.StringValue("Valid Monitor"),
				RegionThreshold: types.Int64Value(1),
				Regions: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("eu-fra-cdg"),
				}),
			},
			Request: validIcmpRequestModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})
}

func TestIcmpRequestModelToClientConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("nil request", func(t *testing.T) {
		cfg, err := icmpRequestModelToClientConfig(ctx, nil)
		require.NoError(t, err)
		require.Nil(t, cfg.Host)
	})

	t.Run("valid host", func(t *testing.T) {
		req := &IcmpRequestModel{
			Host: types.StringValue("1.1.1.1"),
		}
		cfg, err := icmpRequestModelToClientConfig(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, cfg.Host)
		require.Equal(t, "1.1.1.1", *cfg.Host)
	})

	t.Run("whitespace trimmed host", func(t *testing.T) {
		req := &IcmpRequestModel{
			Host: types.StringValue("   1.1.1.1   "),
		}
		cfg, err := icmpRequestModelToClientConfig(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, cfg.Host)
		require.Equal(t, "1.1.1.1", *cfg.Host)
	})

	t.Run("whitespace only host sets nil", func(t *testing.T) {
		req := &IcmpRequestModel{
			Host: types.StringValue("    "),
		}
		cfg, err := icmpRequestModelToClientConfig(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cfg.Host)
	})

	t.Run("null host", func(t *testing.T) {
		req := &IcmpRequestModel{
			Host: types.StringNull(),
		}
		cfg, err := icmpRequestModelToClientConfig(ctx, req)
		require.NoError(t, err)
		require.Nil(t, cfg.Host)
	})
}

func TestClientConfigToIcmpRequestModel(t *testing.T) {
	ctx := context.Background()

	t.Run("valid host", func(t *testing.T) {
		host := "8.8.8.8"
		cfg := client.MonitorRequestConfig{
			Host: &host,
		}
		model, err := clientConfigToIcmpRequestModel(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, model)
		require.Equal(t, "8.8.8.8", model.Host.ValueString())
	})

	t.Run("nil host sets null", func(t *testing.T) {
		cfg := client.MonitorRequestConfig{}
		model, err := clientConfigToIcmpRequestModel(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, model)
		require.True(t, model.Host.IsNull())
	})
}

func TestUptimeMonitorIcmpResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &uptimeMonitorIcmpResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}

func TestUptimeMonitorIcmpResource_ImportState(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r := &uptimeMonitorIcmpResource{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	newTestState := func() tfsdk.State {
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, &uptimeMonitorIcmpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.ListNull(types.StringType),
				Regions: types.ListNull(types.StringType),
			},
		})
		require.False(t, diags.HasError())
		return state
	}

	t.Run("import with integer ID only on project-scoped client", func(t *testing.T) {
		projectClient, err := client.NewClient("https://api.phare.io", "pha_proj_123", 10*time.Second, "123", "", "1.0", "1.0", true)
		require.NoError(t, err)

		res := &uptimeMonitorIcmpResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: projectClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "456"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model uptimeMonitorIcmpModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(456), model.Id.ValueInt64())
	})

	t.Run("import with project_scope/id slug on org-scoped client", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		res := &uptimeMonitorIcmpResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "my-project/789"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model uptimeMonitorIcmpModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
		require.Equal(t, "my-project", helpers.GetDynamicStringValue(model.ProjectScope))
	})

	t.Run("import with project_scope/id integer on org-scoped client", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		res := &uptimeMonitorIcmpResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "123/789"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model uptimeMonitorIcmpModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
		require.Equal(t, "123", helpers.GetDynamicStringValue(model.ProjectScope))
	})

	t.Run("import without project scope on org-scoped client with no provider scope returns error", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		res := &uptimeMonitorIcmpResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "789"}, resp)

		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Missing Project Scope for Import", resp.Diagnostics.Errors()[0].Summary())
	})

	t.Run("invalid import ID format", func(t *testing.T) {
		res := &uptimeMonitorIcmpResource{}
		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "a/b/c"}, resp)

		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Invalid Import ID", resp.Diagnostics.Errors()[0].Summary())
	})
}
