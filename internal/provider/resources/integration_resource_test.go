package resources

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"terraform-provider-phare/internal/client"
)

const testNtfyAccessToken = "tk_00000000000000000000000000000"

var integrationResourceConstructors = map[string]func() resource.Resource{
	"phare_integration_email":            NewIntegrationEmailResource,
	"phare_integration_ilert":            NewIntegrationIlertResource,
	"phare_integration_ntfy":             NewIntegrationNtfyResource,
	"phare_integration_outgoing_webhook": NewIntegrationOutgoingWebhookResource,
	"phare_integration_pushover":         NewIntegrationPushoverResource,
	"phare_integration_sms":              NewIntegrationSmsResource,
}

// integrationResourceSchema returns the schema of an integration resource
func integrationResourceSchema(t *testing.T, r resource.Resource) schema.Schema {
	t.Helper()

	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())

	return resp.Schema
}

// integrationSettingsAttributes returns the nested settings attributes of an integration resource
func integrationSettingsAttributes(t *testing.T, r resource.Resource) map[string]schema.Attribute {
	t.Helper()

	settings, ok := integrationResourceSchema(t, r).Attributes["settings"].(schema.SingleNestedAttribute)
	require.True(t, ok)

	return settings.Attributes
}

// newIntegrationTestResource returns an integration resource configured with a client of the given test server
func newIntegrationTestResource(t *testing.T, constructor func() resource.Resource, serverURL string) resource.Resource {
	t.Helper()

	c, err := client.NewClient(serverURL, "token", 10*time.Second, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	r := constructor()
	resp := &resource.ConfigureResponse{}
	r.(resource.ResourceWithConfigure).Configure(context.Background(), resource.ConfigureRequest{ProviderData: c}, resp)
	require.False(t, resp.Diagnostics.HasError())

	return r
}

// emptyIntegrationState returns a null state for the given schema
func emptyIntegrationState(s schema.Schema) tfsdk.State {
	return tfsdk.State{
		Schema: s,
		Raw:    tftypes.NewValue(s.Type().TerraformType(context.Background()), nil),
	}
}

// recordedRequest is a request received by the integration test server
type recordedRequest struct {
	Method string
	Path   string
	Body   string
}

// newIntegrationTestServer returns a test server answering every request with the given status and body,
// the received requests are appended to requests
func newIntegrationTestServer(t *testing.T, requests *[]recordedRequest, status int, body string) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqBody, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		*requests = append(*requests, recordedRequest{Method: r.Method, Path: r.URL.Path, Body: string(reqBody)})

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)

	return server
}

func TestIntegrationResources_Metadata(t *testing.T) {
	for typeName, constructor := range integrationResourceConstructors {
		t.Run(typeName, func(t *testing.T) {
			resp := &resource.MetadataResponse{}
			constructor().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "phare"}, resp)

			require.Equal(t, typeName, resp.TypeName)
		})
	}
}

func TestIntegrationResources_Schema(t *testing.T) {
	for typeName, constructor := range integrationResourceConstructors {
		t.Run(typeName, func(t *testing.T) {
			s := integrationResourceSchema(t, constructor())

			require.NotEmpty(t, s.Description)
			require.NotEmpty(t, s.MarkdownDescription)

			require.True(t, s.Attributes["name"].IsRequired())
			for _, name := range []string{"id", "paused", "created_at", "updated_at"} {
				require.True(t, s.Attributes[name].IsComputed(), name)
				require.False(t, s.Attributes[name].IsRequired(), name)
				require.False(t, s.Attributes[name].IsOptional(), name)
			}

			settings, ok := s.Attributes["settings"].(schema.SingleNestedAttribute)
			require.True(t, ok)
			require.True(t, settings.IsRequired())
			require.NotEmpty(t, settings.Attributes)
		})
	}
}

func TestIntegrationResources_SettingsAttributes(t *testing.T) {
	testCases := []struct {
		resource  func() resource.Resource
		attribute string
		required  bool
		sensitive bool
	}{
		{NewIntegrationEmailResource, "members", true, false},
		{NewIntegrationSmsResource, "members", true, false},
		{NewIntegrationIlertResource, "integration_key", true, true},
		{NewIntegrationNtfyResource, "hostname", true, false},
		{NewIntegrationNtfyResource, "topic", true, false},
		{NewIntegrationNtfyResource, "access_token", false, true},
		{NewIntegrationOutgoingWebhookResource, "url", true, false},
		{NewIntegrationOutgoingWebhookResource, "secret", true, true},
		{NewIntegrationPushoverResource, "user_key", true, true},
		{NewIntegrationPushoverResource, "api_key", true, true},
	}

	attributeCounts := map[string]int{}
	for _, tc := range testCases {
		r := tc.resource()
		metadata := &resource.MetadataResponse{}
		r.Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "phare"}, metadata)
		attributeCounts[metadata.TypeName]++

		t.Run(metadata.TypeName+"/"+tc.attribute, func(t *testing.T) {
			attribute, ok := integrationSettingsAttributes(t, r)[tc.attribute]
			require.True(t, ok)

			require.Equal(t, tc.required, attribute.IsRequired())
			require.Equal(t, !tc.required, attribute.IsOptional())
			require.Equal(t, tc.sensitive, attribute.IsSensitive())
			require.NotEmpty(t, attribute.GetDescription())
		})
	}

	// Every settings attribute must be covered so that new secrets can not miss the sensitive flag
	for typeName, constructor := range integrationResourceConstructors {
		require.Len(t, integrationSettingsAttributes(t, constructor()), attributeCounts[typeName], typeName)
	}
}

func TestIntegrationResources_NameValidation(t *testing.T) {
	nameAttr, ok := integrationResourceSchema(t, NewIntegrationNtfyResource()).Attributes["name"].(schema.StringAttribute)
	require.True(t, ok)

	testCases := []struct {
		name        string
		value       string
		expectError bool
	}{
		{"valid name", "DevOps Team", false},
		{"min length", "ab", false},
		{"max length", strings.Repeat("a", 50), false},
		{"too short", "a", true},
		{"too short after trim", "  a  ", true},
		{"too long", strings.Repeat("a", 51), true},
		{"emoji", "DevOps 🚀", true},
		{"forbidden symbol", "DevOps/Infra", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			for _, v := range nameAttr.Validators {
				v.ValidateString(context.Background(), validator.StringRequest{
					Path:        path.Root("name"),
					ConfigValue: types.StringValue(tc.value),
				}, resp)
			}

			require.Equal(t, tc.expectError, resp.Diagnostics.HasError())
		})
	}
}

func TestIntegrationResources_SettingsValidation(t *testing.T) {
	testCases := []struct {
		name        string
		resource    func() resource.Resource
		attribute   string
		value       string
		expectError bool
	}{
		{"ilert key min length", NewIntegrationIlertResource, "integration_key", strings.Repeat("a", 40), false},
		{"ilert key max length", NewIntegrationIlertResource, "integration_key", strings.Repeat("a", 70), false},
		{"ilert key too short", NewIntegrationIlertResource, "integration_key", strings.Repeat("a", 39), true},
		{"ilert key too long", NewIntegrationIlertResource, "integration_key", strings.Repeat("a", 71), true},

		{"ntfy hostname", NewIntegrationNtfyResource, "hostname", "ntfy.sh", false},
		{"ntfy custom hostname", NewIntegrationNtfyResource, "hostname", "ntfy.example.com", false},
		{"ntfy hostname with scheme", NewIntegrationNtfyResource, "hostname", "https://ntfy.sh", true},
		{"ntfy hostname ip", NewIntegrationNtfyResource, "hostname", "10.20.30.40", true},
		{"ntfy hostname localhost", NewIntegrationNtfyResource, "hostname", "localhost", true},
		{"ntfy hostname too long", NewIntegrationNtfyResource, "hostname", strings.Repeat("a.", 50) + "sh", true},
		{"ntfy topic", NewIntegrationNtfyResource, "topic", "phare-alerts", false},
		{"ntfy empty topic", NewIntegrationNtfyResource, "topic", "", true},
		{"ntfy topic too long", NewIntegrationNtfyResource, "topic", strings.Repeat("a", 256), true},
		{"ntfy access token", NewIntegrationNtfyResource, "access_token", testNtfyAccessToken, false},
		{"ntfy access token without prefix", NewIntegrationNtfyResource, "access_token", strings.Repeat("a", 32), true},
		{"ntfy access token too short", NewIntegrationNtfyResource, "access_token", "tk_" + strings.Repeat("a", 28), true},
		{"ntfy access token too long", NewIntegrationNtfyResource, "access_token", "tk_" + strings.Repeat("a", 30), true},

		{"webhook https url", NewIntegrationOutgoingWebhookResource, "url", "https://example.com/webhooks/phare", false},
		{"webhook http url", NewIntegrationOutgoingWebhookResource, "url", "http://example.com/webhooks/phare", false},
		{"webhook url without scheme", NewIntegrationOutgoingWebhookResource, "url", "example.com/webhooks/phare", true},
		{"webhook url with other scheme", NewIntegrationOutgoingWebhookResource, "url", "ftp://example.com", true},
		{"webhook url too long", NewIntegrationOutgoingWebhookResource, "url", "https://example.com/" + strings.Repeat("a", 231), true},
		{"webhook secret min length", NewIntegrationOutgoingWebhookResource, "secret", strings.Repeat("a", 10), false},
		{"webhook secret too short", NewIntegrationOutgoingWebhookResource, "secret", strings.Repeat("a", 9), true},
		{"webhook secret too long", NewIntegrationOutgoingWebhookResource, "secret", strings.Repeat("a", 251), true},

		{"pushover user key", NewIntegrationPushoverResource, "user_key", strings.Repeat("u", 30), false},
		{"pushover user key too short", NewIntegrationPushoverResource, "user_key", strings.Repeat("u", 29), true},
		{"pushover api key", NewIntegrationPushoverResource, "api_key", strings.Repeat("a", 30), false},
		{"pushover api key too long", NewIntegrationPushoverResource, "api_key", strings.Repeat("a", 31), true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stringAttr, ok := integrationSettingsAttributes(t, tc.resource())[tc.attribute].(schema.StringAttribute)
			require.True(t, ok)

			resp := &validator.StringResponse{}
			for _, v := range stringAttr.Validators {
				v.ValidateString(context.Background(), validator.StringRequest{
					Path:        path.Root("settings").AtName(tc.attribute),
					ConfigValue: types.StringValue(tc.value),
				}, resp)
			}

			require.Equal(t, tc.expectError, resp.Diagnostics.HasError())
		})
	}
}

func TestIntegrationResources_MembersValidation(t *testing.T) {
	members := func(ids ...int64) types.Set {
		elements := make([]attr.Value, len(ids))
		for i, id := range ids {
			elements[i] = types.Int64Value(id)
		}
		return types.SetValueMust(types.Int64Type, elements)
	}

	manyMembers := make([]int64, 101)
	for i := range manyMembers {
		manyMembers[i] = int64(i + 1)
	}

	testCases := []struct {
		name        string
		value       types.Set
		expectError bool
	}{
		{"single member", members(1), false},
		{"multiple members", members(1, 2, 3), false},
		{"max members", members(manyMembers[:100]...), false},
		{"no member", members(), true},
		{"too many members", members(manyMembers...), true},
		{"invalid member ID", members(0), true},
	}

	for _, constructor := range []func() resource.Resource{NewIntegrationEmailResource, NewIntegrationSmsResource} {
		setAttr, ok := integrationSettingsAttributes(t, constructor())["members"].(schema.SetAttribute)
		require.True(t, ok)

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				resp := &validator.SetResponse{}
				for _, v := range setAttr.Validators {
					v.ValidateSet(context.Background(), validator.SetRequest{
						Path:        path.Root("settings").AtName("members"),
						ConfigValue: tc.value,
					}, resp)
				}

				require.Equal(t, tc.expectError, resp.Diagnostics.HasError())
			})
		}
	}
}

func TestIntegrationResources_BuildSettings(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name     string
		model    integrationModel
		prior    integrationModel
		expected string
	}{
		{
			name: "members",
			model: &integrationMembersModel{Settings: &integrationMembersSettingsModel{
				Members: types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(1)}),
			}},
			expected: `{"members":[1]}`,
		},
		{
			name: "ilert",
			model: &integrationIlertModel{Settings: &integrationIlertSettingsModel{
				IntegrationKey: types.StringValue("il1api"),
			}},
			expected: `{"integration_key":"il1api"}`,
		},
		{
			name: "ntfy with access token",
			model: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("alerts"),
				AccessToken: types.StringValue(testNtfyAccessToken),
			}},
			expected: `{"hostname":"ntfy.sh","topic":"alerts","access_token":"` + testNtfyAccessToken + `"}`,
		},
		{
			name: "ntfy without access token omits it",
			model: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("alerts"),
				AccessToken: types.StringNull(),
			}},
			expected: `{"hostname":"ntfy.sh","topic":"alerts"}`,
		},
		{
			name: "ntfy without access token in the state omits it to keep the current one",
			model: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("alerts"),
				AccessToken: types.StringNull(),
			}},
			prior: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("old"),
				AccessToken: types.StringNull(),
			}},
			expected: `{"hostname":"ntfy.sh","topic":"alerts"}`,
		},
		{
			name: "ntfy access token removed from the configuration sends null to remove it",
			model: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("alerts"),
				AccessToken: types.StringNull(),
			}},
			prior: &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue("alerts"),
				AccessToken: types.StringValue(testNtfyAccessToken),
			}},
			expected: `{"hostname":"ntfy.sh","topic":"alerts","access_token":null}`,
		},
		{
			name: "outgoing webhook",
			model: &integrationOutgoingWebhookModel{Settings: &integrationOutgoingWebhookSettingsModel{
				Url:    types.StringValue("https://example.com/webhooks/phare"),
				Secret: types.StringValue("a-very-secure-secret"),
			}},
			expected: `{"url":"https://example.com/webhooks/phare","secret":"a-very-secure-secret"}`,
		},
		{
			name: "pushover",
			model: &integrationPushoverModel{Settings: &integrationPushoverSettingsModel{
				UserKey: types.StringValue("user"),
				ApiKey:  types.StringValue("api"),
			}},
			expected: `{"user_key":"user","api_key":"api"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var diags diag.Diagnostics
			settings := tc.model.buildSettings(ctx, tc.prior, &diags)
			require.False(t, diags.HasError())

			body, err := json.Marshal(settings)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(body))
		})
	}
}

func TestIntegrationResources_ReadSettings(t *testing.T) {
	ctx := context.Background()

	t.Run("members are refreshed", func(t *testing.T) {
		model := &integrationMembersModel{}
		var diags diag.Diagnostics
		model.readSettings(ctx, json.RawMessage(`{"members":[3,1]}`), &diags)
		require.False(t, diags.HasError())

		var members []int64
		require.False(t, model.Settings.Members.ElementsAs(ctx, &members, false).HasError())
		require.ElementsMatch(t, []int64{1, 3}, members)
	})

	t.Run("ntfy keeps the access token", func(t *testing.T) {
		model := &integrationNtfyModel{Settings: &integrationNtfySettingsModel{
			Hostname:    types.StringValue("ntfy.sh"),
			Topic:       types.StringValue("alerts"),
			AccessToken: types.StringValue(testNtfyAccessToken),
		}}
		var diags diag.Diagnostics
		model.readSettings(ctx, json.RawMessage(`{"hostname":"ntfy.example.com","topic":"renamed"}`), &diags)
		require.False(t, diags.HasError())

		require.Equal(t, "ntfy.example.com", model.Settings.Hostname.ValueString())
		require.Equal(t, "renamed", model.Settings.Topic.ValueString())
		require.Equal(t, testNtfyAccessToken, model.Settings.AccessToken.ValueString())
	})

	t.Run("outgoing webhook keeps the secret", func(t *testing.T) {
		model := &integrationOutgoingWebhookModel{Settings: &integrationOutgoingWebhookSettingsModel{
			Url:    types.StringValue("https://example.com/old"),
			Secret: types.StringValue("a-very-secure-secret"),
		}}
		var diags diag.Diagnostics
		model.readSettings(ctx, json.RawMessage(`{"url":"https://example.com/new"}`), &diags)
		require.False(t, diags.HasError())

		require.Equal(t, "https://example.com/new", model.Settings.Url.ValueString())
		require.Equal(t, "a-very-secure-secret", model.Settings.Secret.ValueString())
	})

	t.Run("imported integrations have null secrets", func(t *testing.T) {
		models := []integrationModel{
			&integrationIlertModel{},
			&integrationNtfyModel{},
			&integrationOutgoingWebhookModel{},
			&integrationPushoverModel{},
		}

		for _, model := range models {
			var diags diag.Diagnostics
			model.readSettings(ctx, json.RawMessage(`{}`), &diags)
			require.False(t, diags.HasError())
		}

		require.True(t, models[0].(*integrationIlertModel).Settings.IntegrationKey.IsNull())
		require.True(t, models[1].(*integrationNtfyModel).Settings.AccessToken.IsNull())
		require.True(t, models[2].(*integrationOutgoingWebhookModel).Settings.Secret.IsNull())
		require.True(t, models[3].(*integrationPushoverModel).Settings.UserKey.IsNull())
		require.True(t, models[3].(*integrationPushoverModel).Settings.ApiKey.IsNull())
	})

	t.Run("invalid settings", func(t *testing.T) {
		var diags diag.Diagnostics
		(&integrationNtfyModel{}).readSettings(ctx, json.RawMessage(`[]`), &diags)
		require.True(t, diags.HasError())
	})
}

func TestIntegrationResource_CRUD(t *testing.T) {
	ctx := context.Background()
	s := integrationResourceSchema(t, NewIntegrationNtfyResource())

	ntfyModel := func(id types.Int64, topic string, accessToken types.String) *integrationNtfyModel {
		return &integrationNtfyModel{
			IntegrationBaseModel: IntegrationBaseModel{
				Id:        id,
				Name:      types.StringValue("Ops"),
				Paused:    types.BoolUnknown(),
				CreatedAt: types.StringUnknown(),
				UpdatedAt: types.StringUnknown(),
			},
			Settings: &integrationNtfySettingsModel{
				Hostname:    types.StringValue("ntfy.sh"),
				Topic:       types.StringValue(topic),
				AccessToken: accessToken,
			},
		}
	}

	ntfyResponse := func(topic string) string {
		return `{"id":42,"app":"ntfy","name":"Ops","paused":false,"settings":{"hostname":"ntfy.sh","topic":"` + topic + `"},"created_at":"2026-10-11T00:00:00Z","updated_at":"2026-10-11T00:00:00Z"}`
	}

	readState := func(t *testing.T, state tfsdk.State) *integrationNtfyModel {
		t.Helper()
		model := &integrationNtfyModel{}
		require.False(t, state.Get(ctx, model).HasError())
		return model
	}

	t.Run("create stores the response and keeps the access token", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusCreated, ntfyResponse("alerts"))
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		plan := tfsdk.Plan{Schema: s}
		require.False(t, plan.Set(ctx, ntfyModel(types.Int64Unknown(), "alerts", types.StringValue(testNtfyAccessToken))).HasError())

		resp := &resource.CreateResponse{State: emptyIntegrationState(s)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		require.Len(t, requests, 1)
		require.Equal(t, http.MethodPost, requests[0].Method)
		require.Equal(t, "/apps/ntfy/integrations", requests[0].Path)
		require.JSONEq(t, `{"name":"Ops","settings":{"hostname":"ntfy.sh","topic":"alerts","access_token":"`+testNtfyAccessToken+`"}}`, requests[0].Body)

		state := readState(t, resp.State)
		require.Equal(t, int64(42), state.Id.ValueInt64())
		require.False(t, state.Paused.ValueBool())
		require.Equal(t, "2026-10-11T00:00:00Z", state.CreatedAt.ValueString())
		require.Equal(t, testNtfyAccessToken, state.Settings.AccessToken.ValueString())
	})

	t.Run("create stores the integration when the response can not be mapped", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusCreated, `{"id":42,"app":"ntfy","name":"Ops","paused":false,"settings":{"topic":1},"created_at":"2026-10-11T00:00:00Z","updated_at":"2026-10-11T00:00:00Z"}`)
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		plan := tfsdk.Plan{Schema: s}
		require.False(t, plan.Set(ctx, ntfyModel(types.Int64Unknown(), "alerts", types.StringNull())).HasError())

		resp := &resource.CreateResponse{State: emptyIntegrationState(s)}
		r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
		require.True(t, resp.Diagnostics.HasError())
		require.Equal(t, "Error decoding integration settings", resp.Diagnostics.Errors()[0].Summary())

		// Terraform marks the integration as tainted to replace it on the next apply
		state := readState(t, resp.State)
		require.Equal(t, int64(42), state.Id.ValueInt64())
		require.Equal(t, "alerts", state.Settings.Topic.ValueString())
	})

	t.Run("read refreshes the settings and keeps the access token", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusOK, ntfyResponse("renamed"))
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		prior := ntfyModel(types.Int64Value(42), "alerts", types.StringValue(testNtfyAccessToken))
		prior.Paused = types.BoolValue(false)
		prior.CreatedAt = types.StringValue("2026-10-11T00:00:00Z")
		prior.UpdatedAt = types.StringValue("2026-10-11T00:00:00Z")
		state := tfsdk.State{Schema: s}
		require.False(t, state.Set(ctx, prior).HasError())

		resp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		require.Len(t, requests, 1)
		require.Equal(t, http.MethodGet, requests[0].Method)
		require.Equal(t, "/apps/ntfy/integrations/42", requests[0].Path)

		refreshed := readState(t, resp.State)
		require.Equal(t, "renamed", refreshed.Settings.Topic.ValueString())
		require.Equal(t, testNtfyAccessToken, refreshed.Settings.AccessToken.ValueString())
	})

	t.Run("read of an imported integration", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusOK, ntfyResponse("alerts"))
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		importResp := &resource.ImportStateResponse{State: emptyIntegrationState(s)}
		r.(resource.ResourceWithImportState).ImportState(ctx, resource.ImportStateRequest{ID: "42"}, importResp)
		require.False(t, importResp.Diagnostics.HasError(), importResp.Diagnostics)

		resp := &resource.ReadResponse{State: importResp.State}
		r.Read(ctx, resource.ReadRequest{State: importResp.State}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		state := readState(t, resp.State)
		require.Equal(t, "Ops", state.Name.ValueString())
		require.Equal(t, "alerts", state.Settings.Topic.ValueString())
		require.True(t, state.Settings.AccessToken.IsNull())
	})

	t.Run("read removes integrations deleted outside of terraform", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusNotFound, `{"message":"Not found"}`)
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		state := tfsdk.State{Schema: s}
		require.False(t, state.Set(ctx, ntfyModel(types.Int64Value(42), "alerts", types.StringNull())).HasError())

		resp := &resource.ReadResponse{State: state}
		r.Read(ctx, resource.ReadRequest{State: state}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		require.True(t, resp.State.Raw.IsNull())
	})

	t.Run("update uses the state ID and removes the access token", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusOK, ntfyResponse("renamed"))
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		state := tfsdk.State{Schema: s}
		require.False(t, state.Set(ctx, ntfyModel(types.Int64Value(42), "alerts", types.StringValue(testNtfyAccessToken))).HasError())
		plan := tfsdk.Plan{Schema: s}
		require.False(t, plan.Set(ctx, ntfyModel(types.Int64Value(42), "renamed", types.StringNull())).HasError())

		resp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		require.Len(t, requests, 1)
		require.Equal(t, http.MethodPost, requests[0].Method)
		require.Equal(t, "/apps/ntfy/integrations/42", requests[0].Path)
		require.JSONEq(t, `{"name":"Ops","settings":{"hostname":"ntfy.sh","topic":"renamed","access_token":null}}`, requests[0].Body)

		updated := readState(t, resp.State)
		require.Equal(t, "renamed", updated.Settings.Topic.ValueString())
		require.True(t, updated.Settings.AccessToken.IsNull())
	})

	t.Run("update keeps an access token unknown to terraform", func(t *testing.T) {
		var requests []recordedRequest
		server := newIntegrationTestServer(t, &requests, http.StatusOK, ntfyResponse("renamed"))
		r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

		// Imported integrations have no access token in the state
		state := tfsdk.State{Schema: s}
		require.False(t, state.Set(ctx, ntfyModel(types.Int64Value(42), "alerts", types.StringNull())).HasError())
		plan := tfsdk.Plan{Schema: s}
		require.False(t, plan.Set(ctx, ntfyModel(types.Int64Value(42), "renamed", types.StringNull())).HasError())

		resp := &resource.UpdateResponse{State: state}
		r.Update(ctx, resource.UpdateRequest{Plan: plan, State: state}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		require.Len(t, requests, 1)
		require.JSONEq(t, `{"name":"Ops","settings":{"hostname":"ntfy.sh","topic":"renamed"}}`, requests[0].Body)
	})

	t.Run("delete", func(t *testing.T) {
		testCases := []struct {
			name         string
			status       int
			body         string
			expectError  bool
			expectDetail string
		}{
			{"deleted", http.StatusNoContent, ``, false, ""},
			{"already deleted", http.StatusNotFound, `{"message":"Not found"}`, false, ""},
			{"used by alert rules", http.StatusConflict, `{"message":"This integration is used by at least one alert rule and cannot be deleted."}`, true, "Delete or update the alert rules using this integration first."},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				var requests []recordedRequest
				server := newIntegrationTestServer(t, &requests, tc.status, tc.body)
				r := newIntegrationTestResource(t, NewIntegrationNtfyResource, server.URL)

				state := tfsdk.State{Schema: s}
				require.False(t, state.Set(ctx, ntfyModel(types.Int64Value(42), "alerts", types.StringNull())).HasError())

				resp := &resource.DeleteResponse{State: state}
				r.Delete(ctx, resource.DeleteRequest{State: state}, resp)

				require.Len(t, requests, 1)
				require.Equal(t, http.MethodDelete, requests[0].Method)
				require.Equal(t, "/apps/ntfy/integrations/42", requests[0].Path)
				require.Equal(t, tc.expectError, resp.Diagnostics.HasError())
				if tc.expectDetail != "" {
					require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tc.expectDetail)
				}
			})
		}
	})
}

func TestIntegrationResource_MembersRoundTrip(t *testing.T) {
	ctx := context.Background()
	s := integrationResourceSchema(t, NewIntegrationEmailResource())

	var requests []recordedRequest
	server := newIntegrationTestServer(t, &requests, http.StatusCreated, `{"id":7,"app":"email","name":"Team","paused":false,"settings":{"members":[2,1]},"created_at":"2026-10-11T00:00:00Z","updated_at":"2026-10-11T00:00:00Z"}`)
	r := newIntegrationTestResource(t, NewIntegrationEmailResource, server.URL)

	plan := tfsdk.Plan{Schema: s}
	require.False(t, plan.Set(ctx, &integrationMembersModel{
		IntegrationBaseModel: IntegrationBaseModel{
			Id:        types.Int64Unknown(),
			Name:      types.StringValue("Team"),
			Paused:    types.BoolUnknown(),
			CreatedAt: types.StringUnknown(),
			UpdatedAt: types.StringUnknown(),
		},
		Settings: &integrationMembersSettingsModel{
			Members: types.SetValueMust(types.Int64Type, []attr.Value{types.Int64Value(1), types.Int64Value(2)}),
		},
	}).HasError())

	resp := &resource.CreateResponse{State: emptyIntegrationState(s)}
	r.Create(ctx, resource.CreateRequest{Plan: plan}, resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

	require.Len(t, requests, 1)
	require.Equal(t, "/apps/email/integrations", requests[0].Path)
	var body client.IntegrationRequest
	body.Settings = &client.MembersIntegrationSettings{}
	require.NoError(t, json.Unmarshal([]byte(requests[0].Body), &body))
	require.ElementsMatch(t, []int64{1, 2}, body.Settings.(*client.MembersIntegrationSettings).Members)

	state := &integrationMembersModel{}
	require.False(t, resp.State.Get(ctx, state).HasError())
	var members []int64
	require.False(t, state.Settings.Members.ElementsAs(ctx, &members, false).HasError())
	require.ElementsMatch(t, []int64{1, 2}, members)
}

func TestIntegrationResource_ImportState(t *testing.T) {
	ctx := context.Background()
	r := NewIntegrationPushoverResource().(resource.ResourceWithImportState)
	s := integrationResourceSchema(t, r)

	t.Run("integration ID", func(t *testing.T) {
		resp := &resource.ImportStateResponse{State: emptyIntegrationState(s)}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "456"}, resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)

		var id types.Int64
		require.False(t, resp.State.GetAttribute(ctx, path.Root("id"), &id).HasError())
		require.Equal(t, int64(456), id.ValueInt64())
	})

	t.Run("invalid ID", func(t *testing.T) {
		for _, id := range []string{"abc", "my-project/456", ""} {
			resp := &resource.ImportStateResponse{State: emptyIntegrationState(s)}
			r.ImportState(ctx, resource.ImportStateRequest{ID: id}, resp)
			require.True(t, resp.Diagnostics.HasError(), id)
			require.Equal(t, "Invalid Import ID", resp.Diagnostics.Errors()[0].Summary())
		}
	})
}

func TestIntegrationResource_Configure(t *testing.T) {
	r := NewIntegrationOutgoingWebhookResource().(*integrationResource)
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), resource.ConfigureRequest{ProviderData: &client.Client{}}, resp)

	require.False(t, resp.Diagnostics.HasError())
	require.NotNil(t, r.GetClient())
}
