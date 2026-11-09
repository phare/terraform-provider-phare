package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListMonitors_BuildsRepeatedTagQueryParams(t *testing.T) {
	var requestedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [], "meta": {"current_page": 1, "from": 0, "to": 0, "per_page": 100}, "links": {}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	_, err = c.ListMonitors(context.Background(), 1, 100, []string{"environment:production", "team:café-1"})
	require.NoError(t, err)
	require.Equal(t, "page=1&per_page=100&tag=environment%3Aproduction&tag=team%3Acaf%C3%A9-1", requestedQuery)
}

func TestListMonitors_NoTagsOmitsTagParams(t *testing.T) {
	var requestedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [], "meta": {"current_page": 1, "from": 0, "to": 0, "per_page": 100}, "links": {}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	_, err = c.ListMonitors(context.Background(), 1, 100, nil)
	require.NoError(t, err)
	require.Equal(t, "page=1&per_page=100", requestedQuery)
}

func TestMonitorResponse_DecodesTags(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1, "project_id": 1, "name": "mon", "protocol": "http", "request": {}, "interval": 60, "timeout": 1000, "incident_confirmations": 1, "recovery_confirmations": 1, "region_threshold": 1, "regions": ["eu-fra-cdg"], "tags": ["environment:production", "team:backend"], "status": "up", "paused": false, "created_at": "2026-01-01", "updated_at": "2026-01-01"}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	monitor, err := c.GetMonitor(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, []string{"environment:production", "team:backend"}, monitor.Tags)
}

func TestMonitorResponse_DecodesDisplayName(t *testing.T) {
	var withDisplayName MonitorResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id": 1, "name": "mon", "display_name": "Website"}`), &withDisplayName))
	require.NotNil(t, withDisplayName.DisplayName)
	require.Equal(t, "Website", *withDisplayName.DisplayName)

	var withoutDisplayName MonitorResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id": 1, "name": "mon", "display_name": null}`), &withoutDisplayName))
	require.Nil(t, withoutDisplayName.DisplayName)
}

func TestMonitorRequest_MarshalsNilTagsAsNull(t *testing.T) {
	req := &MonitorRequest{Name: "mon", Protocol: "http"}
	body, err := json.Marshal(req)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"mon","display_name":null,"protocol":"http","request":{},"interval":0,"timeout":0,"incident_confirmations":0,"recovery_confirmations":0,"region_threshold":0,"regions":null,"tags":null}`, string(body))
}

func TestMonitorRequest_MarshalsDisplayName(t *testing.T) {
	displayName := "Website"
	body, err := json.Marshal(&MonitorRequest{Name: "mon", DisplayName: &displayName})
	require.NoError(t, err)
	require.Contains(t, string(body), `"display_name":"Website"`)
}

func TestMonitorRequestConfig_MarshalsUserAgentSecret(t *testing.T) {
	secret := "definitely-not-a-bot"

	testCases := []struct {
		name     string
		config   MonitorRequestConfig
		expected string
	}{
		{
			name:     "omitted keeps the current secret",
			config:   MonitorRequestConfig{},
			expected: `{}`,
		},
		{
			name:     "null removes the secret",
			config:   MonitorRequestConfig{RemoveUserAgentSecret: true},
			expected: `{"user_agent_secret":null}`,
		},
		{
			name:     "value replaces the secret",
			config:   MonitorRequestConfig{UserAgentSecret: &secret, RemoveUserAgentSecret: true},
			expected: `{"user_agent_secret":"definitely-not-a-bot"}`,
		},
		{
			name:     "read only flag is never sent",
			config:   MonitorRequestConfig{UserAgentSecretEnabled: new(true)},
			expected: `{}`,
		},
		{
			name:     "read only flag is never sent with a null secret",
			config:   MonitorRequestConfig{UserAgentSecretEnabled: new(true), RemoveUserAgentSecret: true},
			expected: `{"user_agent_secret":null}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.config)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(body))
		})
	}

	t.Run("nested in a monitor request", func(t *testing.T) {
		body, err := json.Marshal(&MonitorRequest{Name: "mon", Request: MonitorRequestConfig{RemoveUserAgentSecret: true}})
		require.NoError(t, err)
		require.Contains(t, string(body), `"request":{"user_agent_secret":null}`)
	})
}

func TestMonitorRequestHeader_Marshals(t *testing.T) {
	value := "Bearer token"

	testCases := []struct {
		name     string
		header   MonitorRequestHeader
		expected string
	}{
		{
			name:     "secret header",
			header:   MonitorRequestHeader{Name: "Authorization", Value: &value, Secret: new(true)},
			expected: `{"name":"Authorization","value":"Bearer token","secret":true}`,
		},
		{
			name:     "explicit public header",
			header:   MonitorRequestHeader{Name: "Authorization", Value: &value, Secret: new(false)},
			expected: `{"name":"Authorization","value":"Bearer token","secret":false}`,
		},
		{
			name:     "flag omitted keeps the current one",
			header:   MonitorRequestHeader{Name: "Authorization", Value: &value},
			expected: `{"name":"Authorization","value":"Bearer token"}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, err := json.Marshal(tc.header)
			require.NoError(t, err)
			require.JSONEq(t, tc.expected, string(body))
		})
	}
}

func TestMonitorResponse_DecodesSecrets(t *testing.T) {
	var monitor MonitorResponse
	require.NoError(t, json.Unmarshal([]byte(`{"id":1,"request":{"method":"GET","user_agent_secret_enabled":true,"headers":[{"name":"Authorization","secret":true},{"name":"X-Public","value":"hello","secret":false}]}}`), &monitor))

	require.Nil(t, monitor.Request.UserAgentSecret)
	require.NotNil(t, monitor.Request.UserAgentSecretEnabled)
	require.True(t, *monitor.Request.UserAgentSecretEnabled)

	require.Len(t, monitor.Request.Headers, 2)
	require.Equal(t, "Authorization", monitor.Request.Headers[0].Name)
	require.Nil(t, monitor.Request.Headers[0].Value)
	require.True(t, *monitor.Request.Headers[0].Secret)
	require.Equal(t, "hello", *monitor.Request.Headers[1].Value)
	require.False(t, *monitor.Request.Headers[1].Secret)
}
