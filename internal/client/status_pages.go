package client

import (
	"context"
	"fmt"
)

// StatusPageRequest represents the request body for creating/updating a status page.
type StatusPageRequest struct {
	Name                 string                `json:"name"`
	Subdomain            *string               `json:"subdomain,omitempty"`
	Domain               *string               `json:"domain,omitempty"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	SearchEngineIndexed  bool                  `json:"search_engine_indexed"`
	WebsiteURL           string                `json:"website_url"`
	Colors               *StatusPageColors     `json:"colors,omitempty"`
	Components           []StatusPageComponent `json:"components"`
	Timeframe            *int64                `json:"timeframe,omitempty"`
	SubscriptionChannels []string              `json:"subscription_channels,omitempty"`
}

// StatusPageColors represents color customization for a status page.
type StatusPageColors struct {
	Operational         string `json:"operational"`
	DegradedPerformance string `json:"degradedPerformance"`
	PartialOutage       string `json:"partialOutage"`
	MajorOutage         string `json:"majorOutage"`
	Maintenance         string `json:"maintenance"`
	Empty               string `json:"empty"`
}

// StatusPageComponent represents a component on a status page.
type StatusPageComponent struct {
	ComponentableType string `json:"componentable_type"`
	ComponentableID   int64  `json:"componentable_id"`
}

// StatusPageResponse represents a status page response from the API.
type StatusPageResponse struct {
	ID                   int64                 `json:"id"`
	ProjectID            int64                 `json:"project_id"`
	Name                 string                `json:"name"`
	Subdomain            *string               `json:"subdomain,omitempty"`
	Domain               *string               `json:"domain,omitempty"`
	Title                string                `json:"title"`
	Description          string                `json:"description"`
	SearchEngineIndexed  bool                  `json:"search_engine_indexed"`
	WebsiteURL           string                `json:"website_url"`
	Colors               *StatusPageColors     `json:"colors,omitempty"`
	Components           []StatusPageComponent `json:"components"`
	Timeframe            *int64                `json:"timeframe,omitempty"`
	SubscriptionChannels []string              `json:"subscription_channels,omitempty"`
	CreatedAt            string                `json:"created_at"`
	UpdatedAt            string                `json:"updated_at"`
}

// CreateStatusPage creates a new status page.
func (c *Client) CreateStatusPage(ctx context.Context, req *StatusPageRequest) (*StatusPageResponse, error) {
	var resp StatusPageResponse
	if err := c.doRequest(ctx, "POST", "/uptime/status-pages", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetStatusPage retrieves a status page by ID.
func (c *Client) GetStatusPage(ctx context.Context, id int64) (*StatusPageResponse, error) {
	var resp StatusPageResponse
	path := fmt.Sprintf("/uptime/status-pages/%d", id)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateStatusPage updates an existing status page.
// Note: The API uses POST (not PUT) for updates.
func (c *Client) UpdateStatusPage(ctx context.Context, id int64, req *StatusPageRequest) (*StatusPageResponse, error) {
	var resp StatusPageResponse
	path := fmt.Sprintf("/uptime/status-pages/%d", id)
	if err := c.doRequest(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteStatusPage deletes a status page by ID.
func (c *Client) DeleteStatusPage(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/uptime/status-pages/%d", id)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ListStatusPages retrieves a paginated list of status pages.
func (c *Client) ListStatusPages(ctx context.Context, page, perPage int) ([]*StatusPageResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	path := fmt.Sprintf("/uptime/status-pages?page=%d&per_page=%d", page, perPage)

	var paginatedResp struct {
		Data  []StatusPageResponse `json:"data"`
		Meta  PaginationMeta       `json:"meta"`
		Links PaginationLinks      `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*StatusPageResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
