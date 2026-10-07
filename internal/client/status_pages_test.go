package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListStatusPages_BuildsRepeatedTagQueryParams(t *testing.T) {
	var requestedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data": [], "meta": {"current_page": 1, "from": 0, "to": 0, "per_page": 100}, "links": {}}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	_, err = c.ListStatusPages(context.Background(), 1, 100, []string{"environment:production", "team:backend"})
	require.NoError(t, err)
	require.Equal(t, "page=1&per_page=100&tag=environment%3Aproduction&tag=team%3Abackend", requestedQuery)
}

func TestUpdateStatusPageWithFiles_SendsCommaJoinedTagsField(t *testing.T) {
	var receivedTags string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(10<<20))
		receivedTags = r.FormValue("tags")
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1, "project_id": 1, "name": "sp", "title": "t", "description": "d", "search_engine_indexed": false, "website_url": "https://example.com", "components": [], "access_password_enabled": false, "access_token_enabled": false, "created_at": "2026-01-01", "updated_at": "2026-01-01"}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	req := &StatusPageRequest{
		Name:        "sp",
		Title:       "t",
		Description: "d",
		WebsiteURL:  "https://example.com",
		Components:  []StatusPageComponent{},
		Tags:        []string{"environment:production", "team:backend"},
	}

	_, err = c.UpdateStatusPageWithFiles(context.Background(), 1, req, nil, nil)
	require.NoError(t, err)
	require.Equal(t, "environment:production,team:backend", receivedTags)
}

func TestUpdateStatusPageWithFiles_SendsEmptyTagsFieldWhenUnset(t *testing.T) {
	var receivedTags string
	var fieldPresent bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, r.ParseMultipartForm(10<<20))
		receivedTags = r.FormValue("tags")
		_, fieldPresent = r.MultipartForm.Value["tags"]
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id": 1, "project_id": 1, "name": "sp", "title": "t", "description": "d", "search_engine_indexed": false, "website_url": "https://example.com", "components": [], "access_password_enabled": false, "access_token_enabled": false, "created_at": "2026-01-01", "updated_at": "2026-01-01"}`))
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10_000_000_000, "", "", "1.0", "1.0", true)
	require.NoError(t, err)

	req := &StatusPageRequest{
		Name:        "sp",
		Title:       "t",
		Description: "d",
		WebsiteURL:  "https://example.com",
		Components:  []StatusPageComponent{},
	}

	_, err = c.UpdateStatusPageWithFiles(context.Background(), 1, req, nil, nil)
	require.NoError(t, err)
	require.True(t, fieldPresent)
	require.Equal(t, "", receivedTags)
}
