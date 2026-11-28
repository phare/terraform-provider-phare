package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeartbeats_CRUDRequests(t *testing.T) {
	const heartbeatJSON = `{"id": 12, "project_id": 3, "status": "waiting", "paused": false, "ping_url": "https://beacon.phare.io/heartbeats/token", "name": "Nightly backup", "display_name": null, "interval": 86400, "grace": 300, "incident_confirmations": 1, "recovery_confirmations": 2, "tags": ["environment:production"], "last_pinged_at": null, "created_at": "2026-11-16T10:00:00.000000Z", "updated_at": "2026-11-16T10:00:00.000000Z"}`

	type request struct {
		method, path, body string
	}
	var requests []request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		requests = append(requests, request{r.Method, r.URL.Path, string(body)})

		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(heartbeatJSON))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)
	ctx := context.Background()

	req := &HeartbeatRequest{Name: "Nightly backup", Interval: 86400, Grace: 300, IncidentConfirmations: 1, RecoveryConfirmations: 2}

	created, err := c.CreateHeartbeat(ctx, req)
	require.NoError(t, err)
	require.Equal(t, int64(12), created.ID)
	require.Equal(t, int64(300), created.Grace)
	require.Equal(t, "waiting", created.Status)
	require.Equal(t, "https://beacon.phare.io/heartbeats/token", created.PingURL)
	require.Nil(t, created.DisplayName)
	require.Nil(t, created.LastPingedAt)
	require.Equal(t, []string{"environment:production"}, created.Tags)

	_, err = c.GetHeartbeat(ctx, 12)
	require.NoError(t, err)
	_, err = c.UpdateHeartbeat(ctx, 12, req)
	require.NoError(t, err)
	require.NoError(t, c.DeleteHeartbeat(ctx, 12))

	require.Len(t, requests, 4)
	require.Equal(t, request{"POST", "/uptime/heartbeats", requests[0].body}, requests[0])
	require.JSONEq(t, `{"name":"Nightly backup","display_name":null,"interval":86400,"grace":300,"incident_confirmations":1,"recovery_confirmations":2,"tags":null}`, requests[0].body)
	require.Equal(t, request{"GET", "/uptime/heartbeats/12", ""}, requests[1])
	require.Equal(t, "POST", requests[2].method)
	require.Equal(t, "/uptime/heartbeats/12", requests[2].path)
	require.Equal(t, request{"DELETE", "/uptime/heartbeats/12", ""}, requests[3])
}

func TestListHeartbeats_BuildsTagQueryParams(t *testing.T) {
	var requestedPath, requestedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedPath = r.URL.Path
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [{"id": 1, "name": "a"}, {"id": 2, "name": "b"}], "meta": {"current_page": 1, "per_page": 100}, "links": {}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	heartbeats, err := c.ListHeartbeats(context.Background(), 1, 100, []string{"environment:production", "team:backend"})
	require.NoError(t, err)
	require.Equal(t, "/uptime/heartbeats", requestedPath)
	require.Equal(t, "page=1&per_page=100&tag=environment%3Aproduction&tag=team%3Abackend", requestedQuery)
	require.Len(t, heartbeats, 2)
	require.Equal(t, int64(2), heartbeats[1].ID)
}

func TestHeartbeatRequest_MarshalsDisplayNameAndTags(t *testing.T) {
	body, err := json.Marshal(&HeartbeatRequest{Name: "hb", DisplayName: new("Backups"), Tags: []string{"a"}})
	require.NoError(t, err)
	require.Contains(t, string(body), `"display_name":"Backups"`)
	require.Contains(t, string(body), `"tags":["a"]`)
}

func TestRedactBody_HeartbeatPingURL(t *testing.T) {
	body, err := json.Marshal(&HeartbeatResponse{ID: 1, Name: "hb", PingURL: "https://beacon.phare.io/heartbeats/" + testSecret})
	require.NoError(t, err)
	require.Contains(t, string(body), testSecret)
	require.NotContains(t, redactBody(body), testSecret)
}
