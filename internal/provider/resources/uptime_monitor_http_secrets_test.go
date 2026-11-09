package resources

import (
	"context"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

func testHeader(name, value string, secret types.Bool) HeaderModel {
	return HeaderModel{Name: types.StringValue(name), Value: types.StringValue(value), Secret: secret}
}

func testHeaderList(t *testing.T, headers ...HeaderModel) types.List {
	t.Helper()
	list, diags := types.ListValueFrom(context.Background(), types.ObjectType{AttrTypes: HeaderModelAttrTypes}, headers)
	require.False(t, diags.HasError(), diags)
	return list
}

func testHeaders(t *testing.T, list types.List) []HeaderModel {
	t.Helper()
	var headers []HeaderModel
	diags := list.ElementsAs(context.Background(), &headers, false)
	require.False(t, diags.HasError(), diags)
	return headers
}

func TestClientConfigToHttpRequestModel_Secrets(t *testing.T) {
	ctx := context.Background()
	method := "GET"
	url := "https://example.com"

	t.Run("secret header values are kept from the prior model by name", func(t *testing.T) {
		prior := validHttpRequestModel()
		prior.Headers = testHeaderList(t,
			testHeader("X-Public", "old", types.BoolValue(false)),
			testHeader("authorization", "Bearer token", types.BoolValue(true)),
		)

		config := client.MonitorRequestConfig{
			Method: &method,
			URL:    &url,
			// The API normalizes the case of header names and keeps their order
			Headers: []client.MonitorRequestHeader{
				{Name: "Authorization", Secret: new(true)},
				{Name: "X-Public", Value: new("new"), Secret: new(false)},
				{Name: "X-Api-Key", Secret: new(true)},
			},
		}

		model, err := clientConfigToHttpRequestModel(ctx, config, prior)
		require.NoError(t, err)

		headers := testHeaders(t, model.Headers)
		require.Len(t, headers, 3)

		require.Equal(t, "authorization", headers[0].Name.ValueString(), "the configured case is kept")
		require.Equal(t, "Bearer token", headers[0].Value.ValueString())
		require.True(t, headers[0].Secret.ValueBool())

		require.Equal(t, "X-Public", headers[1].Name.ValueString())
		require.Equal(t, "new", headers[1].Value.ValueString(), "public values come from the API")
		require.False(t, headers[1].Secret.ValueBool())

		require.Equal(t, "X-Api-Key", headers[2].Name.ValueString())
		require.True(t, headers[2].Value.IsNull(), "a secret header unknown to Terraform has no value")
		require.True(t, headers[2].Secret.ValueBool())
	})

	t.Run("imported secret headers have no value", func(t *testing.T) {
		config := client.MonitorRequestConfig{
			Headers: []client.MonitorRequestHeader{{Name: "Authorization", Secret: new(true)}},
		}

		model, err := clientConfigToHttpRequestModel(ctx, config, nil)
		require.NoError(t, err)

		headers := testHeaders(t, model.Headers)
		require.Equal(t, "Authorization", headers[0].Name.ValueString())
		require.True(t, headers[0].Value.IsNull())
		require.True(t, headers[0].Secret.ValueBool())
	})

	t.Run("headers of API versions without secrets are public", func(t *testing.T) {
		config := client.MonitorRequestConfig{
			Headers: []client.MonitorRequestHeader{{Name: "X-Public", Value: new("hello")}},
		}

		model, err := clientConfigToHttpRequestModel(ctx, config, nil)
		require.NoError(t, err)

		headers := testHeaders(t, model.Headers)
		require.Equal(t, "hello", headers[0].Value.ValueString())
		require.False(t, headers[0].Secret.ValueBool())
	})

	t.Run("no headers", func(t *testing.T) {
		model, err := clientConfigToHttpRequestModel(ctx, client.MonitorRequestConfig{}, validHttpRequestModel())
		require.NoError(t, err)
		require.True(t, model.Headers.IsNull())
	})

	testCases := []struct {
		name     string
		config   client.MonitorRequestConfig
		prior    types.String
		expected types.String
	}{
		{
			name:     "user agent secret is kept from the prior model while one is set",
			config:   client.MonitorRequestConfig{UserAgentSecretEnabled: new(true)},
			prior:    types.StringValue("definitely-not-a-bot"),
			expected: types.StringValue("definitely-not-a-bot"),
		},
		{
			name:     "user agent secret removed outside of Terraform is null",
			config:   client.MonitorRequestConfig{UserAgentSecretEnabled: new(false)},
			prior:    types.StringValue("definitely-not-a-bot"),
			expected: types.StringNull(),
		},
		{
			name:     "user agent secret set outside of Terraform is not detected",
			config:   client.MonitorRequestConfig{UserAgentSecretEnabled: new(true)},
			prior:    types.StringNull(),
			expected: types.StringNull(),
		},
		{
			name:     "user agent secret returned by API versions before secrets were hidden",
			config:   client.MonitorRequestConfig{UserAgentSecret: new("returned")},
			prior:    types.StringValue("definitely-not-a-bot"),
			expected: types.StringValue("returned"),
		},
		{
			name:     "no user agent secret",
			config:   client.MonitorRequestConfig{},
			prior:    types.StringValue("definitely-not-a-bot"),
			expected: types.StringNull(),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			prior := validHttpRequestModel()
			prior.UserAgentSecret = tc.prior

			model, err := clientConfigToHttpRequestModel(ctx, tc.config, prior)
			require.NoError(t, err)
			require.Equal(t, tc.expected, model.UserAgentSecret)
		})
	}

	t.Run("imported user agent secret is null", func(t *testing.T) {
		model, err := clientConfigToHttpRequestModel(ctx, client.MonitorRequestConfig{UserAgentSecretEnabled: new(true)}, nil)
		require.NoError(t, err)
		require.True(t, model.UserAgentSecret.IsNull())
	})
}

func TestHttpRequestModelToClientConfig_Secrets(t *testing.T) {
	ctx := context.Background()

	withSecret := func(secret types.String) *HttpRequestModel {
		model := validHttpRequestModel()
		model.UserAgentSecret = secret
		return model
	}

	t.Run("user agent secret is sent when set", func(t *testing.T) {
		config, err := httpRequestModelToClientConfig(ctx, withSecret(types.StringValue("new")), withSecret(types.StringValue("old")))
		require.NoError(t, err)
		require.Equal(t, "new", *config.UserAgentSecret)
	})

	t.Run("user agent secret removed from the configuration is removed", func(t *testing.T) {
		config, err := httpRequestModelToClientConfig(ctx, withSecret(types.StringNull()), withSecret(types.StringValue("old")))
		require.NoError(t, err)
		require.Nil(t, config.UserAgentSecret)
		require.True(t, config.RemoveUserAgentSecret)
	})

	t.Run("user agent secret unknown to Terraform is kept", func(t *testing.T) {
		config, err := httpRequestModelToClientConfig(ctx, withSecret(types.StringNull()), withSecret(types.StringNull()))
		require.NoError(t, err)
		require.Nil(t, config.UserAgentSecret)
		require.False(t, config.RemoveUserAgentSecret)

		config, err = httpRequestModelToClientConfig(ctx, withSecret(types.StringNull()), nil)
		require.NoError(t, err)
		require.False(t, config.RemoveUserAgentSecret)
	})

	t.Run("headers are sent with their value and known secret flag", func(t *testing.T) {
		model := validHttpRequestModel()
		model.Headers = testHeaderList(t,
			testHeader("Authorization", "Bearer token", types.BoolValue(true)),
			testHeader("X-Public", "hello", types.BoolValue(false)),
			testHeader("X-Kept", "hello", types.BoolNull()),
		)

		config, err := httpRequestModelToClientConfig(ctx, model, nil)
		require.NoError(t, err)
		require.Len(t, config.Headers, 3)

		require.Equal(t, "Bearer token", *config.Headers[0].Value)
		require.True(t, *config.Headers[0].Secret)
		require.Equal(t, "hello", *config.Headers[1].Value)
		require.False(t, *config.Headers[1].Secret)
		require.Nil(t, config.Headers[2].Secret, "an unknown flag is left out to keep the current one")
	})
}

func TestPlanHeaderSecrets(t *testing.T) {
	ctx := context.Background()

	withHeaders := func(headers types.List) *HttpRequestModel {
		model := validHttpRequestModel()
		model.Headers = headers
		return model
	}

	t.Run("unset flags are resolved by name from the prior state", func(t *testing.T) {
		state := withHeaders(testHeaderList(t,
			testHeader("X-Public", "hello", types.BoolValue(false)),
			testHeader("Authorization", "Bearer token", types.BoolValue(true)),
		))
		// Reordered, a header added and the case of a name changed
		config := withHeaders(testHeaderList(t,
			testHeader("authorization", "Bearer token", types.BoolNull()),
			testHeader("X-New", "new", types.BoolNull()),
			testHeader("X-Public", "hello", types.BoolNull()),
		))
		plan := withHeaders(testHeaderList(t,
			testHeader("authorization", "Bearer token", types.BoolUnknown()),
			testHeader("X-New", "new", types.BoolUnknown()),
			testHeader("X-Public", "hello", types.BoolUnknown()),
		))

		list, diags := planHeaderSecrets(ctx, plan, config, state)
		require.False(t, diags.HasError(), diags)

		headers := testHeaders(t, list)
		require.Equal(t, types.BoolValue(true), headers[0].Secret)
		require.Equal(t, types.BoolValue(false), headers[1].Secret)
		require.Equal(t, types.BoolValue(false), headers[2].Secret)
	})

	t.Run("new monitors have public headers by default", func(t *testing.T) {
		config := withHeaders(testHeaderList(t, testHeader("Authorization", "Bearer token", types.BoolNull())))
		plan := withHeaders(testHeaderList(t, testHeader("Authorization", "Bearer token", types.BoolUnknown())))

		list, diags := planHeaderSecrets(ctx, plan, config, nil)
		require.False(t, diags.HasError(), diags)
		require.Equal(t, types.BoolValue(false), testHeaders(t, list)[0].Secret)
	})

	t.Run("configured flags are kept", func(t *testing.T) {
		state := withHeaders(testHeaderList(t,
			testHeader("Authorization", "Bearer token", types.BoolValue(true)),
			testHeader("X-Api-Key", "key", types.BoolValue(false)),
			testHeader("X-Computed", "value", types.BoolValue(false)),
		))
		config := withHeaders(testHeaderList(t,
			testHeader("Authorization", "Bearer token", types.BoolValue(false)),
			testHeader("X-Api-Key", "key", types.BoolValue(true)),
			testHeader("X-Computed", "value", types.BoolUnknown()),
		))
		plan := withHeaders(testHeaderList(t,
			testHeader("Authorization", "Bearer token", types.BoolValue(false)),
			testHeader("X-Api-Key", "key", types.BoolValue(true)),
			testHeader("X-Computed", "value", types.BoolUnknown()),
		))

		list, diags := planHeaderSecrets(ctx, plan, config, state)
		require.False(t, diags.HasError(), diags)

		headers := testHeaders(t, list)
		require.Equal(t, types.BoolValue(false), headers[0].Secret)
		require.Equal(t, types.BoolValue(true), headers[1].Secret)
		require.True(t, headers[2].Secret.IsUnknown(), "a configured flag known after apply stays unknown")
	})

	t.Run("known flags of an unchanged monitor are kept", func(t *testing.T) {
		headers := testHeaderList(t, testHeader("Authorization", "Bearer token", types.BoolValue(true)))
		config := withHeaders(testHeaderList(t, testHeader("Authorization", "Bearer token", types.BoolNull())))

		list, diags := planHeaderSecrets(ctx, withHeaders(headers), config, withHeaders(headers))
		require.False(t, diags.HasError(), diags)
		require.True(t, list.Equal(headers))
	})

	t.Run("unknown header names stay unknown", func(t *testing.T) {
		header := HeaderModel{Name: types.StringUnknown(), Value: types.StringValue("v"), Secret: types.BoolUnknown()}
		plan := withHeaders(testHeaderList(t, header))
		config := withHeaders(testHeaderList(t, HeaderModel{Name: types.StringUnknown(), Value: types.StringValue("v"), Secret: types.BoolNull()}))

		list, diags := planHeaderSecrets(ctx, plan, config, nil)
		require.False(t, diags.HasError(), diags)
		require.True(t, testHeaders(t, list)[0].Secret.IsUnknown())
	})

	t.Run("no headers", func(t *testing.T) {
		list, diags := planHeaderSecrets(ctx, validHttpRequestModel(), nil, nil)
		require.False(t, diags.HasError(), diags)
		require.True(t, list.IsNull())

		list, diags = planHeaderSecrets(ctx, nil, nil, nil)
		require.False(t, diags.HasError(), diags)
		require.True(t, list.IsNull())

		unknown := withHeaders(types.ListUnknown(types.ObjectType{AttrTypes: HeaderModelAttrTypes}))
		list, diags = planHeaderSecrets(ctx, unknown, nil, nil)
		require.False(t, diags.HasError(), diags)
		require.True(t, list.IsUnknown())
	})
}

func TestUptimeMonitorHttpResource_ModifyPlan_HeaderSecrets(t *testing.T) {
	ctx := context.Background()
	r := &uptimeMonitorHttpResource{}
	realClient, err := client.NewClient("https://api.phare.io", "token", 10*time.Second, "123", "", "1.0", "1.0", true)
	require.NoError(t, err)
	r.Configure(ctx, resource.ConfigureRequest{ProviderData: realClient}, &resource.ConfigureResponse{})

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	model := func(headers types.List) uptimeMonitorHttpModel {
		request := validHttpRequestModel()
		request.Headers = headers
		return uptimeMonitorHttpModel{
			UptimeMonitorBaseModel: UptimeMonitorBaseModel{
				Tags:    types.SetNull(types.StringType),
				Name:    types.StringValue("Valid Monitor"),
				Regions: types.ListNull(types.StringType),
			},
			Request:           request,
			SuccessAssertions: validSuccessAssertionsModel(),
		}
	}

	toState := func(m uptimeMonitorHttpModel) tfsdk.State {
		state := tfsdk.State{Schema: schemaResp.Schema}
		require.False(t, state.Set(ctx, m).HasError())
		return state
	}

	state := toState(model(testHeaderList(t, testHeader("Authorization", "Bearer token", types.BoolValue(true)))))
	config := toState(model(testHeaderList(t,
		testHeader("X-New", "new", types.BoolNull()),
		testHeader("Authorization", "Bearer token", types.BoolNull()),
	)))
	plan := tfsdk.Plan{Schema: schemaResp.Schema}
	require.False(t, plan.Set(ctx, model(testHeaderList(t,
		testHeader("X-New", "new", types.BoolUnknown()),
		testHeader("Authorization", "Bearer token", types.BoolUnknown()),
	))).HasError())

	resp := &resource.ModifyPlanResponse{Plan: plan}
	r.ModifyPlan(ctx, resource.ModifyPlanRequest{
		Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: config.Raw},
		State:  state,
		Plan:   plan,
	}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	var headers types.List
	require.False(t, resp.Plan.GetAttribute(ctx, path.Root("request").AtName("headers"), &headers).HasError())
	planned := testHeaders(t, headers)
	require.Equal(t, types.BoolValue(false), planned[0].Secret)
	require.Equal(t, types.BoolValue(true), planned[1].Secret)
}

func TestUptimeMonitorHttpResource_SecretsSchema(t *testing.T) {
	resp := &resource.SchemaResponse{}
	NewUptimeMonitorHttpResource().Schema(context.Background(), resource.SchemaRequest{}, resp)

	request := resp.Schema.Blocks["request"].(schema.SingleNestedBlock)

	// Not computed, so removing it from the configuration plans its removal
	userAgentSecret := request.Attributes["user_agent_secret"].(schema.StringAttribute)
	require.True(t, userAgentSecret.Optional)
	require.False(t, userAgentSecret.Computed)
	require.True(t, userAgentSecret.Sensitive)

	headers := request.Blocks["headers"].(schema.ListNestedBlock)
	require.NotEmpty(t, headers.Validators)

	value := headers.NestedObject.Attributes["value"].(schema.StringAttribute)
	require.True(t, value.Sensitive)

	// No default, a header without it keeps its current flag
	secret := headers.NestedObject.Attributes["secret"].(schema.BoolAttribute)
	require.True(t, secret.Optional)
	require.True(t, secret.Computed)
	require.Nil(t, secret.Default)
}
