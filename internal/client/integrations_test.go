package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateIntegration_SendsSettings(t *testing.T) {
	var method, path, body string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		reqBody, _ := io.ReadAll(r.Body)
		body = string(reqBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id": 1, "app": "ntfy", "name": "Ops", "paused": false, "settings": {"hostname": "ntfy.sh", "topic": "alerts"}, "created_at": "2026-10-11T00:00:00Z", "updated_at": "2026-10-11T00:00:00Z"}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	integration, err := c.CreateIntegration(context.Background(), "ntfy", &IntegrationRequest{
		Name:     "Ops",
		Settings: NtfyIntegrationSettings{Hostname: "ntfy.sh", Topic: "alerts"},
	})
	require.NoError(t, err)

	require.Equal(t, http.MethodPost, method)
	require.Equal(t, "/apps/ntfy/integrations", path)
	require.JSONEq(t, `{"name":"Ops","settings":{"hostname":"ntfy.sh","topic":"alerts"}}`, body)

	require.Equal(t, int64(1), integration.ID)
	require.Equal(t, "ntfy", integration.App)
	require.JSONEq(t, `{"hostname": "ntfy.sh", "topic": "alerts"}`, string(integration.Settings))
}

func TestUpdateIntegration_SendsSettings(t *testing.T) {
	var method, path, body string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		reqBody, _ := io.ReadAll(r.Body)
		body = string(reqBody)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 5, "app": "pushover", "name": "Phone", "paused": false, "settings": {}, "created_at": "2026-10-11T00:00:00Z", "updated_at": "2026-10-11T00:00:00Z"}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	integration, err := c.UpdateIntegration(context.Background(), "pushover", 5, &IntegrationRequest{
		Name:     "Phone",
		Settings: PushoverIntegrationSettings{UserKey: "user", APIKey: "api"},
	})
	require.NoError(t, err)

	require.Equal(t, http.MethodPost, method)
	require.Equal(t, "/apps/pushover/integrations/5", path)
	require.JSONEq(t, `{"name":"Phone","settings":{"user_key":"user","api_key":"api"}}`, body)
	require.JSONEq(t, `{}`, string(integration.Settings))
}

func TestNtfyIntegrationSettings_MarshalJSON(t *testing.T) {
	token := "tk_00000000000000000000000000000"

	testCases := []struct {
		name     string
		settings NtfyIntegrationSettings
		expected string
	}{
		{"access token", NtfyIntegrationSettings{Hostname: "ntfy.sh", Topic: "alerts", AccessToken: &token}, `{"hostname":"ntfy.sh","topic":"alerts","access_token":"` + token + `"}`},
		{"no access token is omitted to keep the current one", NtfyIntegrationSettings{Hostname: "ntfy.sh", Topic: "alerts"}, `{"hostname":"ntfy.sh","topic":"alerts"}`},
		{"removed access token is null", NtfyIntegrationSettings{Hostname: "ntfy.sh", Topic: "alerts", RemoveAccessToken: true}, `{"hostname":"ntfy.sh","topic":"alerts","access_token":null}`},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.settings)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(body))
		})
	}
}

func TestDeleteIntegration(t *testing.T) {
	var method, path string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	require.NoError(t, c.DeleteIntegration(context.Background(), "outgoing_webhook", 9))
	require.Equal(t, http.MethodDelete, method)
	require.Equal(t, "/apps/outgoing_webhook/integrations/9", path)
}

func TestDeleteIntegration_Conflict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte(`{"message": "This integration is used by at least one alert rule and cannot be deleted."}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	err = c.DeleteIntegration(context.Background(), "email", 9)

	var apiErr *APIError
	require.True(t, errors.As(err, &apiErr))
	require.Equal(t, http.StatusConflict, apiErr.StatusCode)
	require.Equal(t, "This integration is used by at least one alert rule and cannot be deleted.", apiErr.Message)
}
