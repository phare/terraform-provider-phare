package client

import (
	"context"
	"fmt"
)

// MonitorRequest represents the request body for creating/updating a monitor.
type MonitorRequest struct {
	Name                  string                   `json:"name"`
	Protocol              string                   `json:"protocol"`
	Request               MonitorRequestConfig     `json:"request"`
	Interval              int64                    `json:"interval"`
	Timeout               int64                    `json:"timeout"`
	SuccessAssertions     []map[string]interface{} `json:"success_assertions,omitempty"`
	IncidentConfirmations int64                    `json:"incident_confirmations"`
	RecoveryConfirmations int64                    `json:"recovery_confirmations"`
	Regions               []string                 `json:"regions"`
}

// MonitorRequestConfig represents the request configuration for a monitor.
// This is a flexible structure that can represent HTTP or TCP requests.
type MonitorRequestConfig struct {
	// HTTP fields
	Method          *string                `json:"method,omitempty"`
	URL             *string                `json:"url,omitempty"`
	TLSSkipVerify   *bool                  `json:"tls_skip_verify,omitempty"`
	Body            *string                `json:"body,omitempty"`
	FollowRedirects *bool                  `json:"follow_redirects,omitempty"`
	UserAgentSecret *string                `json:"user_agent_secret,omitempty"`
	Headers         []MonitorRequestHeader `json:"headers,omitempty"`

	// TCP fields
	Host       *string `json:"host,omitempty"`
	Port       *string `json:"port,omitempty"`
	Connection *string `json:"connection,omitempty"`
}

// MonitorRequestHeader represents a custom HTTP header.
type MonitorRequestHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// MonitorResponse represents a monitor response from the API.
type MonitorResponse struct {
	ID                    int64                    `json:"id"`
	ProjectID             int64                    `json:"project_id"`
	Name                  string                   `json:"name"`
	Protocol              string                   `json:"protocol"`
	Request               MonitorRequestConfig     `json:"request"`
	Interval              int64                    `json:"interval"`
	Timeout               int64                    `json:"timeout"`
	SuccessAssertions     []map[string]interface{} `json:"success_assertions,omitempty"`
	IncidentConfirmations int64                    `json:"incident_confirmations"`
	RecoveryConfirmations int64                    `json:"recovery_confirmations"`
	Regions               []string                 `json:"regions"`
	Status                string                   `json:"status"`
	Paused                bool                     `json:"paused"`
	ResponseTime          *int64                   `json:"response_time"`
	OneDayAvailability    *float64                 `json:"one_day_availability"`
	SevenDaysAvailability *float64                 `json:"seven_days_availability"`
	CreatedAt             string                   `json:"created_at"`
	UpdatedAt             string                   `json:"updated_at"`
}

// CreateMonitor creates a new uptime monitor.
func (c *Client) CreateMonitor(ctx context.Context, req *MonitorRequest) (*MonitorResponse, error) {
	var resp MonitorResponse
	if err := c.doRequest(ctx, "POST", "/uptime/monitors", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetMonitor retrieves an uptime monitor by ID.
func (c *Client) GetMonitor(ctx context.Context, id int64) (*MonitorResponse, error) {
	var resp MonitorResponse
	path := fmt.Sprintf("/uptime/monitors/%d", id)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateMonitor updates an existing uptime monitor.
func (c *Client) UpdateMonitor(ctx context.Context, id int64, req *MonitorRequest) (*MonitorResponse, error) {
	var resp MonitorResponse
	path := fmt.Sprintf("/uptime/monitors/%d", id)
	if err := c.doRequest(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteMonitor deletes an uptime monitor by ID.
func (c *Client) DeleteMonitor(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/uptime/monitors/%d", id)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ListMonitors retrieves a paginated list of uptime monitors.
func (c *Client) ListMonitors(ctx context.Context, page, perPage int) ([]*MonitorResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	path := fmt.Sprintf("/uptime/monitors?page=%d&per_page=%d", page, perPage)

	var paginatedResp struct {
		Data  []MonitorResponse `json:"data"`
		Meta  PaginationMeta    `json:"meta"`
		Links PaginationLinks   `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	result := make([]*MonitorResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
