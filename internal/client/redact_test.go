package client

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"github.com/stretchr/testify/require"
)

const testSecret = "s3cr3t-value-that-must-not-be-logged"

func TestRedactBody(t *testing.T) {
	testCases := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "top level secret",
			body:     `{"name":"Ops","secret":"` + testSecret + `"}`,
			expected: `{"name":"Ops","secret":"***"}`,
		},
		{
			name:     "nested secrets",
			body:     `{"name":"Ops","settings":{"user_key":"` + testSecret + `","api_key":"` + testSecret + `"}}`,
			expected: `{"name":"Ops","settings":{"user_key":"***","api_key":"***"}}`,
		},
		{
			name:     "secrets in arrays",
			body:     `{"data":[{"request":{"user_agent_secret":"` + testSecret + `"}},{"request":{}}]}`,
			expected: `{"data":[{"request":{"user_agent_secret":"***"}},{"request":{}}]}`,
		},
		{
			name:     "null secret is kept",
			body:     `{"hostname":"ntfy.sh","access_token":null}`,
			expected: `{"hostname":"ntfy.sh","access_token":null}`,
		},
		{
			name:     "numbers are preserved",
			body:     `{"id":12345678901234567890,"ratio":0.5,"secret":"` + testSecret + `"}`,
			expected: `{"id":12345678901234567890,"ratio":0.5,"secret":"***"}`,
		},
		{
			name:     "similar keys are not redacted",
			body:     `{"access_token_enabled":true,"access_password_enabled":false}`,
			expected: `{"access_token_enabled":true,"access_password_enabled":false}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			redacted := redactBody([]byte(tc.body))
			require.JSONEq(t, tc.expected, redacted)
			require.NotContains(t, redacted, testSecret)
		})
	}

	t.Run("body without secrets is unchanged", func(t *testing.T) {
		body := `{ "name": "Ops",  "members": [1, 2] }`
		require.Equal(t, body, redactBody([]byte(body)))
	})

	t.Run("non JSON body is unchanged", func(t *testing.T) {
		body := `<html>Bad Gateway</html>`
		require.Equal(t, body, redactBody([]byte(body)))
	})
}

// TestRedactBody_SensitiveRequestFields verifies that every secret sent by the
// client is redacted, a new secret field must be added to sensitiveBodyKeys
func TestRedactBody_SensitiveRequestFields(t *testing.T) {
	secret := testSecret

	requests := map[string]any{
		"ilert integration":            &IntegrationRequest{Name: "Ops", Settings: IlertIntegrationSettings{IntegrationKey: secret}},
		"ntfy integration":             &IntegrationRequest{Name: "Ops", Settings: NtfyIntegrationSettings{Hostname: "ntfy.sh", Topic: "alerts", AccessToken: &secret}},
		"outgoing webhook integration": &IntegrationRequest{Name: "Ops", Settings: OutgoingWebhookIntegrationSettings{URL: "https://example.com", Secret: secret}},
		"pushover integration":         &IntegrationRequest{Name: "Ops", Settings: PushoverIntegrationSettings{UserKey: secret, APIKey: secret}},
		"http monitor":                 &MonitorRequest{Name: "Monitor", Request: MonitorRequestConfig{UserAgentSecret: &secret}},
		"status page":                  &StatusPageRequest{Name: "Status", AccessToken: &secret, AccessPassword: &secret},
	}

	for name, req := range requests {
		t.Run(name, func(t *testing.T) {
			body, err := json.Marshal(req)
			require.NoError(t, err)
			require.Contains(t, string(body), secret)

			require.NotContains(t, redactBody(body), secret)
		})
	}
}

func TestDoRequest_DoesNotLogSecrets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/apps/outgoing_webhook/integrations" {
			w.WriteHeader(http.StatusUnprocessableEntity)
			w.Write([]byte(`{"message":"The given data was invalid.","errors":{"settings.url":["The url is invalid."]},"secret":"` + testSecret + `"}`))
			return
		}
		w.Write([]byte(`{"id":1,"name":"Monitor","request":{"user_agent_secret":"` + testSecret + `"}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	var logs bytes.Buffer
	ctx := tflogtest.RootLogger(context.Background(), &logs)

	// Secrets in the request body, logged at TRACE level
	_, err = c.CreateIntegration(ctx, "pushover", &IntegrationRequest{
		Name:     "Ops",
		Settings: PushoverIntegrationSettings{UserKey: testSecret, APIKey: testSecret},
	})
	require.NoError(t, err)

	// Secrets in the response body, logged at TRACE level
	_, err = c.GetMonitor(ctx, 1)
	require.NoError(t, err)

	// Secrets in an error response body, logged at WARN level
	_, err = c.CreateIntegration(ctx, "outgoing_webhook", &IntegrationRequest{
		Name:     "Ops",
		Settings: OutgoingWebhookIntegrationSettings{URL: "invalid", Secret: testSecret},
	})
	require.Error(t, err)

	require.Contains(t, logs.String(), "Phare API request body")
	require.Contains(t, logs.String(), "Phare API response body")
	require.Contains(t, logs.String(), "Phare API returned error")
	require.NotContains(t, logs.String(), testSecret)
}
