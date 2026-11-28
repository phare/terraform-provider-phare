package resources

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
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

func heartbeatSchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	NewUptimeHeartbeatResource().Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func TestUptimeHeartbeatResource_Metadata(t *testing.T) {
	resp := &resource.MetadataResponse{}
	NewUptimeHeartbeatResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "phare"}, resp)
	require.Equal(t, "phare_uptime_heartbeat", resp.TypeName)
}

func TestUptimeHeartbeatResource_Schema(t *testing.T) {
	s := heartbeatSchema(t)

	for _, name := range []string{"name", "interval", "grace", "incident_confirmations", "recovery_confirmations"} {
		require.True(t, s.Attributes[name].IsRequired(), name)
	}
	for _, name := range []string{"display_name", "tags", "project_scope"} {
		require.True(t, s.Attributes[name].IsOptional(), name)
	}
	for _, name := range []string{"id", "project_id", "status", "paused", "ping_url", "created_at", "updated_at"} {
		require.True(t, s.Attributes[name].IsComputed(), name)
	}

	pingURL, ok := s.Attributes["ping_url"].(schema.StringAttribute)
	require.True(t, ok)
	require.True(t, pingURL.Sensitive)
	require.NotEmpty(t, pingURL.PlanModifiers)
}

func TestUptimeHeartbeatResource_Int64Validation(t *testing.T) {
	s := heartbeatSchema(t)

	testCases := []struct {
		attribute   string
		value       int64
		expectError bool
	}{
		{"interval", 60, false},
		{"interval", 86400, false},
		{"interval", 2678400, false},
		{"interval", 30, true},
		{"interval", 90, true},
		{"interval", 2678460, true},
		{"grace", 60, false},
		{"grace", 604800, false},
		{"grace", 0, true},
		{"grace", 150, true},
		{"grace", 604860, true},
		{"incident_confirmations", 1, false},
		{"incident_confirmations", 5, false},
		{"incident_confirmations", 6, true},
		{"recovery_confirmations", 0, true},
	}

	for _, tc := range testCases {
		t.Run(tc.attribute, func(t *testing.T) {
			attr, ok := s.Attributes[tc.attribute].(schema.Int64Attribute)
			require.True(t, ok)

			req := validator.Int64Request{Path: path.Root(tc.attribute), ConfigValue: types.Int64Value(tc.value)}
			resp := &validator.Int64Response{}
			for _, v := range attr.Validators {
				v.ValidateInt64(context.Background(), req, resp)
			}

			require.Equal(t, tc.expectError, resp.Diagnostics.HasError(), "%s = %d", tc.attribute, tc.value)
		})
	}
}

func TestUptimeHeartbeatResource_NameValidation(t *testing.T) {
	s := heartbeatSchema(t)

	testCases := []struct {
		name        string
		value       types.String
		expectError bool
	}{
		{"valid", types.StringValue("Nightly backup"), false},
		{"valid with slash", types.StringValue("backups/nightly"), false},
		{"valid 45 chars", types.StringValue(strings.Repeat("a", 45)), false},
		{"invalid 1 char", types.StringValue("a"), true},
		{"invalid 46 chars", types.StringValue(strings.Repeat("a", 46)), true},
		{"invalid whitespace only", types.StringValue("   "), true},
		{"invalid emoji", types.StringValue("backup 🚀"), true},
	}

	for _, attribute := range []string{"name", "display_name"} {
		attr, ok := s.Attributes[attribute].(schema.StringAttribute)
		require.True(t, ok)

		for _, tc := range testCases {
			t.Run(attribute+" "+tc.name, func(t *testing.T) {
				req := validator.StringRequest{Path: path.Root(attribute), ConfigValue: tc.value}
				resp := &validator.StringResponse{}
				for _, v := range attr.Validators {
					v.ValidateString(context.Background(), req, resp)
				}

				require.Equal(t, tc.expectError, resp.Diagnostics.HasError())
			})
		}
	}
}

func TestBuildHeartbeatAPIRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("with display name and tags", func(t *testing.T) {
		var diags diag.Diagnostics
		req, ok := buildHeartbeatAPIRequest(ctx, &UptimeHeartbeatModel{
			Name:                  types.StringValue("Nightly backup"),
			DisplayName:           types.StringValue("Backups"),
			Interval:              types.Int64Value(86400),
			Grace:                 types.Int64Value(300),
			IncidentConfirmations: types.Int64Value(1),
			RecoveryConfirmations: types.Int64Value(2),
			Tags:                  types.SetValueMust(types.StringType, []attr.Value{types.StringValue("environment:production")}),
		}, &diags)

		require.True(t, ok)
		require.False(t, diags.HasError())
		require.Equal(t, &client.HeartbeatRequest{
			Name:                  "Nightly backup",
			DisplayName:           new("Backups"),
			Interval:              86400,
			Grace:                 300,
			IncidentConfirmations: 1,
			RecoveryConfirmations: 2,
			Tags:                  []string{"environment:production"},
		}, req)
	})

	t.Run("without display name and tags", func(t *testing.T) {
		var diags diag.Diagnostics
		req, ok := buildHeartbeatAPIRequest(ctx, &UptimeHeartbeatModel{
			Name:        types.StringValue("Nightly backup"),
			DisplayName: types.StringNull(),
			Tags:        types.SetNull(types.StringType),
		}, &diags)

		require.True(t, ok)
		require.Nil(t, req.DisplayName)
		require.Nil(t, req.Tags)
	})
}

func TestUpdateHeartbeatModelFromResponse(t *testing.T) {
	var diags diag.Diagnostics
	model := UptimeHeartbeatModel{Tags: types.SetNull(types.StringType)}

	updateHeartbeatModelFromResponse(&client.HeartbeatResponse{
		ID:                    12,
		ProjectID:             3,
		Name:                  "Nightly backup",
		Interval:              86400,
		Grace:                 300,
		IncidentConfirmations: 1,
		RecoveryConfirmations: 2,
		Status:                "waiting",
		PingURL:               "https://beacon.phare.io/heartbeats/token",
		CreatedAt:             "2026-11-16T10:00:00.000000Z",
		UpdatedAt:             "2026-11-16T10:00:00.000000Z",
	}, &model, &diags)

	require.False(t, diags.HasError())
	require.Equal(t, int64(12), model.Id.ValueInt64())
	require.Equal(t, int64(3), model.ProjectId.ValueInt64())
	require.True(t, model.DisplayName.IsNull())
	require.Equal(t, int64(300), model.Grace.ValueInt64())
	require.Equal(t, "waiting", model.Status.ValueString())
	require.False(t, model.Paused.ValueBool())
	require.Equal(t, "https://beacon.phare.io/heartbeats/token", model.PingUrl.ValueString())
	// No tags stay null when none were configured
	require.True(t, model.Tags.IsNull())
}

func TestUptimeHeartbeatResource_ModifyPlan(t *testing.T) {
	r := &uptimeHeartbeatResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: realClient}, &resource.ConfigureResponse{})

	s := heartbeatSchema(t)
	newModel := func(scope string) UptimeHeartbeatModel {
		return UptimeHeartbeatModel{
			Id:           types.Int64Value(100),
			Name:         types.StringValue("Nightly backup"),
			Tags:         types.SetNull(types.StringType),
			ProjectScope: types.DynamicValue(types.StringValue(scope)),
		}
	}

	state := tfsdk.State{Schema: s}
	require.False(t, state.Set(context.Background(), newModel("project-a")).HasError())
	plan := tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(context.Background(), newModel("project-b")).HasError())

	resp := &resource.ModifyPlanResponse{}
	r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{State: state, Plan: plan}, resp)

	require.False(t, resp.Diagnostics.HasError())
	require.True(t, resp.RequiresReplace.Contains(path.Root("project_scope")))
}

func TestUptimeHeartbeatResource_ImportState(t *testing.T) {
	ctx := context.Background()
	s := heartbeatSchema(t)

	orgClient, err := client.NewClient("https://api.phare.io", "pha_org_123", 10*time.Second, "", "", "1.0", "1.0", false)
	require.NoError(t, err)

	r := &uptimeHeartbeatResource{}
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: orgClient}, &resource.ConfigureResponse{})

	state := tfsdk.State{Schema: s}
	require.False(t, state.Set(ctx, &UptimeHeartbeatModel{Tags: types.SetNull(types.StringType)}).HasError())

	resp := &resource.ImportStateResponse{State: state}
	r.ImportState(ctx, resource.ImportStateRequest{ID: "my-project/789"}, resp)
	require.False(t, resp.Diagnostics.HasError())

	var model UptimeHeartbeatModel
	require.False(t, resp.State.Get(ctx, &model).HasError())
	require.Equal(t, int64(789), model.Id.ValueInt64())
	require.Equal(t, "my-project", helpers.GetDynamicStringValue(model.ProjectScope))
}
