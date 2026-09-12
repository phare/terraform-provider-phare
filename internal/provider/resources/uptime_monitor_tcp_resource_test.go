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
)

func validTcpRequestModel() *TcpRequestModel {
	return &TcpRequestModel{
		Host:          types.StringValue("example.com"),
		Port:          types.Int64Value(443),
		Connection:    types.StringValue("tls"),
		TlsSkipVerify: types.BoolValue(false),
	}
}

func TestUptimeMonitorTcpResource_Metadata(t *testing.T) {
	r := NewUptimeMonitorTcpResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_monitor_tcp", resp.TypeName)
}

func TestUptimeMonitorTcpResource_Schema(t *testing.T) {
	r := NewUptimeMonitorTcpResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestUptimeMonitorTcpResource_NameValidation(t *testing.T) {
	r := NewUptimeMonitorTcpResource()
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

func TestUptimeMonitorTcpResource_RegionsValidation(t *testing.T) {
	r := NewUptimeMonitorTcpResource()
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

func TestUptimeMonitorTcpResource_RequestValidation(t *testing.T) {
	r := NewUptimeMonitorTcpResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	reqBlock, ok := resp.Schema.Blocks["request"].(schema.SingleNestedBlock)
	require.True(t, ok)

	hostAttr, ok := reqBlock.Attributes["host"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, hostAttr.Validators)

	portAttr, ok := reqBlock.Attributes["port"].(schema.Int64Attribute)
	require.True(t, ok)
	require.NotEmpty(t, portAttr.Validators)

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

	t.Run("port validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.Int64
			expectError bool
		}{
			{
				name:        "valid min port 1",
				val:         types.Int64Value(1),
				expectError: false,
			},
			{
				name:        "valid port 443",
				val:         types.Int64Value(443),
				expectError: false,
			},
			{
				name:        "valid max port 65535",
				val:         types.Int64Value(65535),
				expectError: false,
			},
			{
				name:        "invalid port 0",
				val:         types.Int64Value(0),
				expectError: true,
			},
			{
				name:        "invalid port negative",
				val:         types.Int64Value(-1),
				expectError: true,
			},
			{
				name:        "invalid port 65536",
				val:         types.Int64Value(65536),
				expectError: true,
			},
			{
				name:        "null value skipped",
				val:         types.Int64Null(),
				expectError: false,
			},
			{
				name:        "unknown value skipped",
				val:         types.Int64Unknown(),
				expectError: false,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				valReq := validator.Int64Request{
					Path:        path.Root("request").AtName("port"),
					ConfigValue: tc.val,
				}
				valResp := &validator.Int64Response{}

				for _, v := range portAttr.Validators {
					v.ValidateInt64(context.Background(), valReq, valResp)
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

func TestUptimeMonitorTcpResource_ModifyPlan(t *testing.T) {
	r := &uptimeMonitorTcpResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	t.Run("missing request", func(t *testing.T) {
		planData := uptimeMonitorTcpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
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

	t.Run("whitespace name trimmed under 2 chars", func(t *testing.T) {
		planData := uptimeMonitorTcpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("  a  "),
				Regions: types.ListNull(types.StringType),
			},
			Request: validTcpRequestModel(),
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

	t.Run("valid plan", func(t *testing.T) {
		planData := uptimeMonitorTcpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: validTcpRequestModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})
}

func TestClientConfigToTcpRequestModel(t *testing.T) {
	ctx := context.Background()

	t.Run("valid port conversion", func(t *testing.T) {
		port := "443"
		host := "example.com"
		cfg := client.MonitorRequestConfig{
			Host: &host,
			Port: &port,
		}
		model, err := clientConfigToTcpRequestModel(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, model)
		require.Equal(t, int64(443), model.Port.ValueInt64())
		require.Equal(t, "example.com", model.Host.ValueString())
	})

	t.Run("invalid port conversion returns error", func(t *testing.T) {
		port := "not-a-port"
		cfg := client.MonitorRequestConfig{
			Port: &port,
		}
		model, err := clientConfigToTcpRequestModel(ctx, cfg)
		require.Error(t, err)
		require.Nil(t, model)
		require.Contains(t, err.Error(), "invalid port value")
	})

	t.Run("nil port sets null", func(t *testing.T) {
		cfg := client.MonitorRequestConfig{}
		model, err := clientConfigToTcpRequestModel(ctx, cfg)
		require.NoError(t, err)
		require.NotNil(t, model)
		require.True(t, model.Port.IsNull())
	})
}

func TestUptimeMonitorTcpResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &uptimeMonitorTcpResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}
