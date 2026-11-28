package client

import (
	"context"
	"fmt"
	"net/url"
)

// HeartbeatRequest represents the request body for creating/updating a heartbeat.
// DisplayName is always sent, as a null clears it while a missing field keeps the current value.
type HeartbeatRequest struct {
	Name                  string   `json:"name"`
	DisplayName           *string  `json:"display_name"`
	Interval              int64    `json:"interval"`
	Grace                 int64    `json:"grace"`
	IncidentConfirmations int64    `json:"incident_confirmations"`
	RecoveryConfirmations int64    `json:"recovery_confirmations"`
	Tags                  []string `json:"tags"`
}

// HeartbeatResponse represents a heartbeat response from the API.
type HeartbeatResponse struct {
	ID                    int64    `json:"id"`
	ProjectID             int64    `json:"project_id"`
	Name                  string   `json:"name"`
	DisplayName           *string  `json:"display_name"`
	Interval              int64    `json:"interval"`
	Grace                 int64    `json:"grace"`
	IncidentConfirmations int64    `json:"incident_confirmations"`
	RecoveryConfirmations int64    `json:"recovery_confirmations"`
	Tags                  []string `json:"tags"`
	Status                string   `json:"status"`
	Paused                bool     `json:"paused"`
	PingURL               string   `json:"ping_url"`
	LastPingedAt          *string  `json:"last_pinged_at"`
	CreatedAt             string   `json:"created_at"`
	UpdatedAt             string   `json:"updated_at"`
}

// CreateHeartbeat creates a new uptime heartbeat.
func (c *Client) CreateHeartbeat(ctx context.Context, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	var resp HeartbeatResponse
	if err := c.doRequest(ctx, "POST", "/uptime/heartbeats", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetHeartbeat retrieves an uptime heartbeat by ID.
func (c *Client) GetHeartbeat(ctx context.Context, id int64) (*HeartbeatResponse, error) {
	var resp HeartbeatResponse
	path := fmt.Sprintf("/uptime/heartbeats/%d", id)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateHeartbeat updates an existing uptime heartbeat. The ping URL stays the same.
func (c *Client) UpdateHeartbeat(ctx context.Context, id int64, req *HeartbeatRequest) (*HeartbeatResponse, error) {
	var resp HeartbeatResponse
	path := fmt.Sprintf("/uptime/heartbeats/%d", id)
	if err := c.doRequest(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteHeartbeat deletes an uptime heartbeat by ID.
func (c *Client) DeleteHeartbeat(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/uptime/heartbeats/%d", id)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ListHeartbeats retrieves a paginated list of uptime heartbeats, only the
// ones having all the given tags when tags are set.
func (c *Client) ListHeartbeats(ctx context.Context, page, perPage int, tags []string) ([]*HeartbeatResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	params := url.Values{}
	params.Set("page", fmt.Sprintf("%d", page))
	params.Set("per_page", fmt.Sprintf("%d", perPage))

	for _, tag := range tags {
		params.Add("tag", tag)
	}

	path := fmt.Sprintf("/uptime/heartbeats?%s", params.Encode())

	var paginatedResp struct {
		Data  []HeartbeatResponse `json:"data"`
		Meta  PaginationMeta      `json:"meta"`
		Links PaginationLinks     `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	result := make([]*HeartbeatResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
