package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// AlertRuleRequest represents the request body for creating/updating an alert rule.
type AlertRuleRequest struct {
	Event               string          `json:"event"`
	Scope               *string         `json:"scope,omitempty"`
	IntegrationID       int64           `json:"integration_id"`
	RateLimit           int64           `json:"rate_limit"`
	EventSettings       json.RawMessage `json:"event_settings,omitempty"`
	IntegrationSettings json.RawMessage `json:"integration_settings,omitempty"`
}

// AlertRuleResponse represents an alert rule response from the API.
type AlertRuleResponse struct {
	ID                  int64           `json:"id"`
	Event               string          `json:"event"`
	ProjectID           *int64          `json:"project_id,omitempty"`
	IntegrationID       int64           `json:"integration_id"`
	RateLimit           int64           `json:"rate_limit"`
	EventSettings       json.RawMessage `json:"event_settings,omitempty"`
	IntegrationSettings json.RawMessage `json:"integration_settings,omitempty"`
	CreatedAt           string          `json:"created_at"`
	UpdatedAt           string          `json:"updated_at"`
}

// CreateAlertRule creates a new alert rule.
func (c *Client) CreateAlertRule(ctx context.Context, req *AlertRuleRequest) (*AlertRuleResponse, error) {
	var resp AlertRuleResponse
	if err := c.doRequest(ctx, "POST", "/alert-rules", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAlertRule retrieves an alert rule by ID.
func (c *Client) GetAlertRule(ctx context.Context, id int64) (*AlertRuleResponse, error) {
	var resp AlertRuleResponse
	path := fmt.Sprintf("/alert-rules/%d", id)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateAlertRule updates an existing alert rule.
func (c *Client) UpdateAlertRule(ctx context.Context, id int64, req *AlertRuleRequest) (*AlertRuleResponse, error) {
	var resp AlertRuleResponse
	path := fmt.Sprintf("/alert-rules/%d", id)
	if err := c.doRequest(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAlertRule deletes an alert rule by ID.
func (c *Client) DeleteAlertRule(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/alert-rules/%d", id)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ListAlertRules retrieves a paginated list of alert rules.
func (c *Client) ListAlertRules(ctx context.Context, page, perPage int) ([]*AlertRuleResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	path := fmt.Sprintf("/alert-rules?page=%d&per_page=%d", page, perPage)

	var paginatedResp struct {
		Data  []AlertRuleResponse `json:"data"`
		Meta  PaginationMeta      `json:"meta"`
		Links PaginationLinks     `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	result := make([]*AlertRuleResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
