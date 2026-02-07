package client

import (
	"context"
	"fmt"
	"net/url"
)

type IntegrationResponse struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Paused    bool   `json:"paused"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetIntegration retrieves an integration by app and ID.
func (c *Client) GetIntegration(ctx context.Context, app string, integrationID int64) (*IntegrationResponse, error) {
	var resp IntegrationResponse
	path := fmt.Sprintf("/apps/%s/integrations/%d", app, integrationID)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetIntegrationByName retrieves an integration by app and name.
// Returns an error if no integration with the given name is found, or if multiple integrations match.
func (c *Client) GetIntegrationByName(ctx context.Context, app, name string) (*IntegrationResponse, error) {
	// Use ListIntegrations with name filter
	integrations, err := c.ListIntegrations(ctx, app, name, 1, 100)
	if err != nil {
		return nil, err
	}

	if len(integrations) == 0 {
		return nil, fmt.Errorf("no integration found with name %q for app %s", name, app)
	}

	// Find exact match (API might return partial matches)
	for _, integration := range integrations {
		if integration.Name == name {
			return integration, nil
		}
	}

	// If no exact match found, return error
	return nil, fmt.Errorf("no integration found with exact name %q for app %s", name, app)
}

// ListIntegrations retrieves a paginated list of integrations for a specific app, optionally filtered by names.
// app: the app key to filter integrations
// names: comma-separated list of integration names (optional, empty string for all)
// page: page number (1-based)
// perPage: number of results per page (max 100)
func (c *Client) ListIntegrations(ctx context.Context, app, names string, page, perPage int) ([]*IntegrationResponse, error) {
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

	if names != "" {
		params.Set("name", names)
	}

	path := fmt.Sprintf("/apps/%s/integrations?%s", app, params.Encode())

	var paginatedResp struct {
		Data  []IntegrationResponse `json:"data"`
		Meta  PaginationMeta        `json:"meta"`
		Links PaginationLinks       `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	result := make([]*IntegrationResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
