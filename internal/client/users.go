package client

import (
	"context"
	"fmt"
	"net/url"
)

type UserResponse struct {
	ID        int64  `json:"id"`
	Role      string `json:"role"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// GetUser retrieves a user by ID.
func (c *Client) GetUser(ctx context.Context, userID int64) (*UserResponse, error) {
	var resp UserResponse
	path := fmt.Sprintf("/users/%d", userID)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ListUsers retrieves a paginated list of users, optionally filtered by emails.
// emails: comma-separated list of email addresses (optional, empty string for all)
// page: page number (1-based)
// perPage: number of results per page (max 100)
func (c *Client) ListUsers(ctx context.Context, emails string, page, perPage int) ([]*UserResponse, error) {
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

	if emails != "" {
		params.Set("email", emails)
	}

	path := fmt.Sprintf("/users?%s", params.Encode())

	var paginatedResp struct {
		Data  []UserResponse  `json:"data"`
		Meta  PaginationMeta  `json:"meta"`
		Links PaginationLinks `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	result := make([]*UserResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
