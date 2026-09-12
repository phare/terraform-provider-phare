package resources

import (
	"context"
	"fmt"
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

func validHttpRequestModel() *HttpRequestModel {
	return &HttpRequestModel{
		Method:          types.StringValue("GET"),
		Url:             types.StringValue("https://example.com"),
		Headers:         types.ListNull(types.ObjectType{AttrTypes: HeaderModelAttrTypes}),
		Body:            types.StringNull(),
		FollowRedirects: types.BoolValue(true),
		TlsSkipVerify:   types.BoolValue(false),
	}
}

func validSuccessAssertionsModel() *SuccessAssertionsModel {
	return &SuccessAssertionsModel{
		StatusCode: types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{
			types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
				"operator": types.StringValue("in"),
				"value":    types.StringValue("2xx"),
			}),
		}),
		ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
		ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
	}
}

func TestUptimeMonitorHttpResource_Metadata(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
	req := resource.MetadataRequest{
		ProviderTypeName: "phare",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	require.Equal(t, "phare_uptime_monitor_http", resp.TypeName)
}

func TestUptimeMonitorHttpResource_Schema(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	// Verify schema is not nil
	require.NotNil(t, resp.Schema)
	require.NotNil(t, resp.Schema.Attributes)
}

func TestUptimeMonitorHttpResource_NameValidation(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
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

func TestUptimeMonitorHttpResource_SchemaValidation(t *testing.T) {
	r := NewUptimeMonitorHttpResource()
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)

	reqBlock, ok := resp.Schema.Blocks["request"].(schema.SingleNestedBlock)
	require.True(t, ok)

	urlAttr, ok := reqBlock.Attributes["url"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, urlAttr.Validators)

	bodyAttr, ok := reqBlock.Attributes["body"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, bodyAttr.Validators)

	secretAttr, ok := reqBlock.Attributes["user_agent_secret"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, secretAttr.Validators)

	headersBlock, ok := reqBlock.Blocks["headers"].(schema.ListNestedBlock)
	require.True(t, ok)

	headerNameAttr, ok := headersBlock.NestedObject.Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, headerNameAttr.Validators)

	headerValueAttr, ok := headersBlock.NestedObject.Attributes["value"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, headerValueAttr.Validators)

	assertionsBlock, ok := resp.Schema.Blocks["success_assertions"].(schema.SingleNestedBlock)
	require.True(t, ok)

	respHeaderBlock, ok := assertionsBlock.Blocks["response_header"].(schema.ListNestedBlock)
	require.True(t, ok)

	selectorAttr, ok := respHeaderBlock.NestedObject.Attributes["selector"].(schema.StringAttribute)
	require.True(t, ok)
	require.NotEmpty(t, selectorAttr.Validators)

	regionsAttr, ok := resp.Schema.Attributes["regions"].(schema.ListAttribute)
	require.True(t, ok)
	require.NotEmpty(t, regionsAttr.Validators)

	t.Run("url validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid http url",
				val:         types.StringValue("http://example.com"),
				expectError: false,
			},
			{
				name:        "valid https url",
				val:         types.StringValue("https://example.com"),
				expectError: false,
			},
			{
				name:        "valid 255 chars url",
				val:         types.StringValue("http://" + strings.Repeat("a", 248)),
				expectError: false,
			},
			{
				name:        "invalid ftp scheme",
				val:         types.StringValue("ftp://example.com"),
				expectError: true,
			},
			{
				name:        "invalid missing scheme",
				val:         types.StringValue("example.com"),
				expectError: true,
			},
			{
				name:        "invalid 256 chars url",
				val:         types.StringValue("https://" + strings.Repeat("a", 248)),
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
					Path:        path.Root("request").AtName("url"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range urlAttr.Validators {
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

	t.Run("body validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid short body",
				val:         types.StringValue("{\"query\": 1}"),
				expectError: false,
			},
			{
				name:        "valid 500 chars body",
				val:         types.StringValue(strings.Repeat("a", 500)),
				expectError: false,
			},
			{
				name:        "invalid 501 chars body",
				val:         types.StringValue(strings.Repeat("a", 501)),
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
					Path:        path.Root("request").AtName("body"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range bodyAttr.Validators {
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

	t.Run("user_agent_secret validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid 1 char secret",
				val:         types.StringValue("s"),
				expectError: false,
			},
			{
				name:        "valid 50 chars secret",
				val:         types.StringValue(strings.Repeat("s", 50)),
				expectError: false,
			},
			{
				name:        "invalid empty string",
				val:         types.StringValue(""),
				expectError: true,
			},
			{
				name:        "invalid 51 chars secret",
				val:         types.StringValue(strings.Repeat("s", 51)),
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
					Path:        path.Root("request").AtName("user_agent_secret"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range secretAttr.Validators {
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

	t.Run("headers name validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid header name",
				val:         types.StringValue("X-Custom-Header"),
				expectError: false,
			},
			{
				name:        "valid 50 chars header name",
				val:         types.StringValue(strings.Repeat("a", 50)),
				expectError: false,
			},
			{
				name:        "invalid empty header name",
				val:         types.StringValue(""),
				expectError: true,
			},
			{
				name:        "invalid 51 chars header name",
				val:         types.StringValue(strings.Repeat("a", 51)),
				expectError: true,
			},
			{
				name:        "invalid space in header name",
				val:         types.StringValue("Header Name"),
				expectError: true,
			},
			{
				name:        "invalid colon in header name",
				val:         types.StringValue("Header:Name"),
				expectError: true,
			},
			{
				name:        "invalid reserved User-Agent",
				val:         types.StringValue("User-Agent"),
				expectError: true,
			},
			{
				name:        "invalid reserved Signature",
				val:         types.StringValue("Signature"),
				expectError: true,
			},
			{
				name:        "invalid reserved Signature-Input",
				val:         types.StringValue("Signature-Input"),
				expectError: true,
			},
			{
				name:        "invalid reserved Signature-Agent",
				val:         types.StringValue("Signature-Agent"),
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
					Path:        path.Root("request").AtName("headers").AtName("name"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range headerNameAttr.Validators {
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

	t.Run("headers value validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid header value",
				val:         types.StringValue("custom-value"),
				expectError: false,
			},
			{
				name:        "valid 1024 chars header value",
				val:         types.StringValue(strings.Repeat("v", 1024)),
				expectError: false,
			},
			{
				name:        "invalid 1025 chars header value",
				val:         types.StringValue(strings.Repeat("v", 1025)),
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
					Path:        path.Root("request").AtName("headers").AtName("value"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range headerValueAttr.Validators {
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

	t.Run("response_header selector validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.String
			expectError bool
		}{
			{
				name:        "valid 1 char selector",
				val:         types.StringValue("s"),
				expectError: false,
			},
			{
				name:        "valid 100 chars selector",
				val:         types.StringValue(strings.Repeat("s", 100)),
				expectError: false,
			},
			{
				name:        "invalid empty selector",
				val:         types.StringValue(""),
				expectError: true,
			},
			{
				name:        "invalid 101 chars selector",
				val:         types.StringValue(strings.Repeat("s", 101)),
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
					Path:        path.Root("success_assertions").AtName("response_header").AtName("selector"),
					ConfigValue: tc.val,
				}
				valResp := &validator.StringResponse{}

				for _, v := range selectorAttr.Validators {
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

	t.Run("regions validation", func(t *testing.T) {
		testCases := []struct {
			name        string
			val         types.List
			expectError bool
		}{
			{
				name:        "valid single region",
				val:         types.ListValueMust(types.StringType, []attr.Value{types.StringValue("eu-fra-cdg")}),
				expectError: false,
			},
			{
				name:        "invalid empty list",
				val:         types.ListValueMust(types.StringType, []attr.Value{}),
				expectError: true,
			},
			{
				name: "invalid duplicate regions",
				val: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("eu-fra-cdg"),
					types.StringValue("eu-fra-cdg"),
				}),
				expectError: true,
			},
			{
				name: "invalid region string",
				val: types.ListValueMust(types.StringType, []attr.Value{
					types.StringValue("invalid-region"),
				}),
				expectError: true,
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
	})
}

func TestUptimeMonitorHttpResource_ModifyPlan(t *testing.T) {
	r := &uptimeMonitorHttpResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)

	t.Run("missing request", func(t *testing.T) {
		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: nil,
			SuccessAssertions: &SuccessAssertionsModel{
				StatusCode:     types.ListNull(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}),
				ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
				ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
			},
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

	t.Run("missing success_assertions", func(t *testing.T) {
		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           validHttpRequestModel(),
			SuccessAssertions: nil,
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Summary() == "Missing Success Assertions" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("empty success_assertions", func(t *testing.T) {
		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: validHttpRequestModel(),
			SuccessAssertions: &SuccessAssertionsModel{
				StatusCode:     types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{}),
				ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
				ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
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
			if diagErr.Summary() == "Missing Success Assertions" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("whitespace name trimmed under 2 chars", func(t *testing.T) {
		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("  a  "),
				Regions: types.ListNull(types.StringType),
			},
			Request: validHttpRequestModel(),
			SuccessAssertions: &SuccessAssertionsModel{
				StatusCode: types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{
					types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
						"operator": types.StringValue("in"),
						"value":    types.StringValue("2xx"),
					}),
				}),
				ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
				ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
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
			if diagErr.Summary() == "Invalid Name Length" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("valid plan", func(t *testing.T) {
		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request: validHttpRequestModel(),
			SuccessAssertions: &SuccessAssertionsModel{
				StatusCode: types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{
					types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
						"operator": types.StringValue("in"),
						"value":    types.StringValue("2xx"),
					}),
				}),
				ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
				ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
			},
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("request body prohibited with GET method", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		reqModel.Method = types.StringValue("GET")
		reqModel.Body = types.StringValue("{\"query\": 1}")

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Detail() == "Request body is prohibited unless method is POST, PUT, or PATCH" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("request body prohibited with HEAD method", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		reqModel.Method = types.StringValue("HEAD")
		reqModel.Body = types.StringValue("{\"query\": 1}")

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Detail() == "Request body is prohibited unless method is POST, PUT, or PATCH" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	for _, method := range []string{"POST", "PUT", "PATCH"} {
		t.Run("request body allowed with "+method+" method", func(t *testing.T) {
			reqModel := validHttpRequestModel()
			reqModel.Method = types.StringValue(method)
			reqModel.Body = types.StringValue("{\"data\": 1}")

			planData := uptimeMonitorHttpModel{
				UptimeMonitorBaseModel: UptimeMonitorBaseModel{
					Name:    types.StringValue("Valid Monitor"),
					Regions: types.ListNull(types.StringType),
				},
				Request:           reqModel,
				SuccessAssertions: validSuccessAssertionsModel(),
			}

			plan := tfsdk.Plan{Schema: schemaResp.Schema}
			diags := plan.Set(context.Background(), planData)
			require.False(t, diags.HasError())

			resp := &resource.ModifyPlanResponse{}
			r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

			require.False(t, resp.Diagnostics.HasError())
		})
	}

	t.Run("request body empty allowed with GET method", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		reqModel.Method = types.StringValue("GET")
		reqModel.Body = types.StringValue("")

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("request body with unknown method allowed during planning", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		reqModel.Method = types.StringUnknown()
		reqModel.Body = types.StringValue("{\"data\": 1}")

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("request headers more than 10 rejected", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		headerElements := make([]attr.Value, 11)
		for i := 0; i < 11; i++ {
			headerElements[i] = types.ObjectValueMust(HeaderModelAttrTypes, map[string]attr.Value{
				"name":  types.StringValue(fmt.Sprintf("X-Header-%d", i)),
				"value": types.StringValue("val"),
			})
		}
		reqModel.Headers = types.ListValueMust(types.ObjectType{AttrTypes: HeaderModelAttrTypes}, headerElements)

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Detail() == "Cannot configure more than 10 HTTP headers" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("request headers 10 allowed", func(t *testing.T) {
		reqModel := validHttpRequestModel()
		headerElements := make([]attr.Value, 10)
		for i := 0; i < 10; i++ {
			headerElements[i] = types.ObjectValueMust(HeaderModelAttrTypes, map[string]attr.Value{
				"name":  types.StringValue(fmt.Sprintf("X-Header-%d", i)),
				"value": types.StringValue("val"),
			})
		}
		reqModel.Headers = types.ListValueMust(types.ObjectType{AttrTypes: HeaderModelAttrTypes}, headerElements)

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           reqModel,
			SuccessAssertions: validSuccessAssertionsModel(),
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})

	t.Run("status code assertions more than 1 rejected", func(t *testing.T) {
		assertions := &SuccessAssertionsModel{
			StatusCode: types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{
				types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
					"operator": types.StringValue("in"),
					"value":    types.StringValue("2xx"),
				}),
				types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
					"operator": types.StringValue("in"),
					"value":    types.StringValue("3xx"),
				}),
			}),
			ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
			ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
		}

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           validHttpRequestModel(),
			SuccessAssertions: assertions,
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.True(t, resp.Diagnostics.HasError())
		foundErr := false
		for _, diagErr := range resp.Diagnostics.Errors() {
			if diagErr.Detail() == "Only one status code assertion is allowed" {
				foundErr = true
			}
		}
		require.True(t, foundErr)
	})

	t.Run("status code assertions 1 allowed", func(t *testing.T) {
		assertions := &SuccessAssertionsModel{
			StatusCode: types.ListValueMust(types.ObjectType{AttrTypes: StatusCodeAssertionModelAttrTypes}, []attr.Value{
				types.ObjectValueMust(StatusCodeAssertionModelAttrTypes, map[string]attr.Value{
					"operator": types.StringValue("in"),
					"value":    types.StringValue("2xx"),
				}),
			}),
			ResponseHeader: types.ListNull(types.ObjectType{AttrTypes: ResponseHeaderAssertionModelAttrTypes}),
			ResponseBody:   types.ListNull(types.ObjectType{AttrTypes: ResponseBodyAssertionModelAttrTypes}),
		}

		planData := uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           validHttpRequestModel(),
			SuccessAssertions: assertions,
		}

		plan := tfsdk.Plan{Schema: schemaResp.Schema}
		diags := plan.Set(context.Background(), planData)
		require.False(t, diags.HasError())

		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(context.Background(), resource.ModifyPlanRequest{Plan: plan}, resp)

		require.False(t, resp.Diagnostics.HasError())
	})
}

func TestUptimeMonitorHttpResource_Configure(t *testing.T) {
	// Create a real client for testing
	realClient := &client.Client{}

	r := &uptimeMonitorHttpResource{}
	r.Configure(context.Background(), resource.ConfigureRequest{
		ProviderData: realClient,
	}, &resource.ConfigureResponse{})

	// Verify resource is properly configured
	require.NotNil(t, r.GetClient())
}
