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

func TestMonitorRequest_MarshalsNilTagsAsNull(t *testing.T) {
	req := &MonitorRequest{Name: "mon", Protocol: "http"}
	body, err := json.Marshal(req)
	require.NoError(t, err)
	require.JSONEq(t, `{"name":"mon","protocol":"http","request":{},"interval":0,"timeout":0,"incident_confirmations":0,"recovery_confirmations":0,"region_threshold":0,"regions":null,"tags":null}`, string(body))
}
