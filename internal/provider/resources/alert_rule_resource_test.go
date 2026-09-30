package resources

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
	"terraform-provider-phare/internal/provider/helpers"
)

func TestAlertRuleResource_Metadata(t *testing.T) {
	r := NewAlertRuleResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_alert_rule", resp.TypeName)
}

func TestAlertRuleResource_Schema(t *testing.T) {
	r := NewAlertRuleResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestAlertRuleResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &alertRuleResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}

func TestAlertRuleResource_EventValidation(t *testing.T) {
	r := NewAlertRuleResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	eventAttr, ok := resp.Schema.Attributes["event"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, eventAttr.Validators)

	validEvents := []string{
		"uptime.monitor.created",
		"uptime.monitor.deleted",
		"uptime.monitor_certificate.discovered",
		"uptime.monitor_certificate.expiring",
		"uptime.incident.created",
		"uptime.incident.propagated",
		"uptime.incident.partially_recovered",
		"uptime.incident.recovered",
		"uptime.incident_comment.created",
		"uptime.incident_update.published",
		"uptime.maintenance_window.in_progress",
		"uptime.maintenance_window.completed",
		"uptime.maintenance_window.cancelled",
	}

	for _, event := range validEvents {
		t.Run("valid event: "+event, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("event"),
				ConfigValue: types.StringValue(event),
			}
			valResp := &validator.StringResponse{}

			for _, v := range eventAttr.Validators {
				v.ValidateString(context.Background(), valReq, valResp)
			}

			require.False(t, valResp.Diagnostics.HasError())
		})
	}

	invalidTestCases := []struct {
		name        string
		val         types.String
		expectError bool
	}{
		{
			name:        "invalid unknown event",
			val:         types.StringValue("uptime.unknown"),
			expectError: true,
		},
		{
			name:        "invalid empty event",
			val:         types.StringValue(""),
			expectError: true,
		},
		{
			name:        "invalid random string",
			val:         types.StringValue("something_else"),
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

	for _, tc := range invalidTestCases {
		t.Run(tc.name, func(t *testing.T) {
			valReq := validator.StringRequest{
				Path:        path.Root("event"),
				ConfigValue: tc.val,
			}
			valResp := &validator.StringResponse{}

			for _, v := range eventAttr.Validators {
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

func TestAlertRuleResource_ModifyPlan(t *testing.T) {
	r := &alertRuleResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

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

	t.Run("project scope change requires replace", func(t *testing.T) {
		stateData := alertRuleModel{
			UptimeAlertRuleModel: UptimeAlertRuleModel{
				Id:            types.Int64Value(100),
				Event:         types.StringValue("uptime.monitor.created"),
				IntegrationId: types.Int64Value(1),
				RateLimit:     types.Int64Value(60),
			},
			ProjectScope: types.DynamicValue(types.StringValue("project-a")),
			Scope:        types.StringValue("project"),
		}

		planData := alertRuleModel{
			UptimeAlertRuleModel: UptimeAlertRuleModel{
				Id:            types.Int64Value(100),
				Event:         types.StringValue("uptime.monitor.created"),
				IntegrationId: types.Int64Value(1),
				RateLimit:     types.Int64Value(60),
			},
			ProjectScope: types.DynamicValue(types.StringValue("project-b")),
			Scope:        types.StringValue("project"),
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

	testCases := []struct {
		name                           string
		eventSettings                  types.String
		integrationSettings            types.String
		expectEventSettingsError       bool
		expectIntegrationSettingsError bool
	}{
		{
			name:                           "valid JSON settings",
			eventSettings:                  types.StringValue(`{"threshold": 5}`),
			integrationSettings:            types.StringValue(`{"channel": "#alerts"}`),
			expectEventSettingsError:       false,
			expectIntegrationSettingsError: false,
		},
		{
			name:                           "null settings skipped",
			eventSettings:                  types.StringNull(),
			integrationSettings:            types.StringNull(),
			expectEventSettingsError:       false,
			expectIntegrationSettingsError: false,
		},
		{
			name:                           "unknown settings skipped",
			eventSettings:                  types.StringUnknown(),
			integrationSettings:            types.StringUnknown(),
			expectEventSettingsError:       false,
			expectIntegrationSettingsError: false,
		},
		{
			name:                           "invalid event_settings JSON",
			eventSettings:                  types.StringValue(`{invalid-json`),
			integrationSettings:            types.StringValue(`{"channel": "#alerts"}`),
			expectEventSettingsError:       true,
			expectIntegrationSettingsError: false,
		},
		{
			name:                           "invalid integration_settings JSON",
			eventSettings:                  types.StringValue(`{"threshold": 5}`),
			integrationSettings:            types.StringValue(`not-json`),
			expectEventSettingsError:       false,
			expectIntegrationSettingsError: true,
		},
		{
			name:                           "both invalid JSON settings",
			eventSettings:                  types.StringValue(`bad json 1`),
			integrationSettings:            types.StringValue(`bad json 2`),
			expectEventSettingsError:       true,
			expectIntegrationSettingsError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			planData := alertRuleModel{
				UptimeAlertRuleModel: UptimeAlertRuleModel{
					CreatedAt:           types.StringNull(),
					Event:               types.StringValue("uptime.monitor.created"),
					EventSettings:       tc.eventSettings,
					Id:                  types.Int64Null(),
					IntegrationId:       types.Int64Value(1),
					IntegrationSettings: tc.integrationSettings,
					ProjectId:           types.Int64Null(),
					RateLimit:           types.Int64Value(60),
					UpdatedAt:           types.StringNull(),
				},
				ProjectScope: types.DynamicNull(),
				Scope:        types.StringValue("project"),
			}

			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			diags := plan.Set(context.Background(), planData)
			require.False(t, diags.HasError())

			resp := &resource.ModifyPlanResponse{}
			r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

			if tc.expectEventSettingsError {
				require.True(t, resp.Diagnostics.HasError())
				foundErr := false
				for _, diagErr := range resp.Diagnostics.Errors() {
					if diagErr.Detail() == "event_settings must be a valid JSON string" {
						foundErr = true
					}
				}
				require.True(t, foundErr, "expected error 'event_settings must be a valid JSON string'")
			}

			if tc.expectIntegrationSettingsError {
				require.True(t, resp.Diagnostics.HasError())
				foundErr := false
				for _, diagErr := range resp.Diagnostics.Errors() {
					if diagErr.Detail() == "integration_settings must be a valid JSON string" {
						foundErr = true
					}
				}
				require.True(t, foundErr, "expected error 'integration_settings must be a valid JSON string'")
			}

			if !tc.expectEventSettingsError && !tc.expectIntegrationSettingsError {
				require.False(t, resp.Diagnostics.HasError())
			}
		})
	}
}

func TestAlertRuleResource_ImportState(t *testing.T) {
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r := NewAlertRuleResource()
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	newTestState := func() tfsdk.State {
		state := tfsdk.State{Schema: schemaResp.Schema}
		diags := state.Set(ctx, &alertRuleModel{})
		require.False(t, diags.HasError())
		return state
	}

	t.Run("import with integer ID only on project-scoped client", func(t *testing.T) {
		projectClient, err := client.NewClient("https://api.phare.io", "pha_proj_123", 10*time.Second, "123", "", "1.0", "1.0", true)
		require.NoError(t, err)

		res := &alertRuleResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: projectClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "456"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model alertRuleModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(456), model.Id.ValueInt64())
	})

	t.Run("import with project_scope/id slug on org-scoped client", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		res := &alertRuleResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "my-project/789"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model alertRuleModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(789), model.Id.ValueInt64())
		require.Equal(t, "my-project", helpers.GetDynamicStringValue(model.ProjectScope))
	})

	t.Run("import with ID only on org-scoped client succeeds without scope (requireProjectScope=false)", func(t *testing.T) {
		orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
		require.NoError(t, err)

		res := &alertRuleResource{}
		res.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

		state := newTestState()
		resp := &resource.ImportStateResponse{State: state}
		res.ImportState(ctx, resource.ImportStateRequest{ID: "999"}, resp)

		require.False(t, resp.Diagnostics.HasError())
		var model alertRuleModel
		resp.Diagnostics.Append(resp.State.Get(ctx, &model)...)
		require.False(t, resp.Diagnostics.HasError())
		require.Equal(t, int64(999), model.Id.ValueInt64())
	})
}
