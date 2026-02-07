package client

import (
	"context"
	"fmt"
)

// ProjectRequest represents the request body for creating/updating a project.
type ProjectRequest struct {
	Name     string           `json:"name"`
	Members  []int64          `json:"members"`
	Settings *ProjectSettings `json:"settings,omitempty"`
}

// ProjectSettings represents project configuration settings.
type ProjectSettings struct {
	UseIncidentAI             *bool  `json:"use_incident_ai,omitempty"`
	UseIncidentMerging        *bool  `json:"use_incident_merging,omitempty"`
	IncidentMergingTimeWindow *int64 `json:"incident_merging_time_window,omitempty"`
}

// ProjectResponse represents a project response from the API.
type ProjectResponse struct {
	ID        int64            `json:"id"`
	Slug      string           `json:"slug"`
	Name      string           `json:"name"`
	Members   []int64          `json:"members"`
	Settings  *ProjectSettings `json:"settings,omitempty"`
	CreatedAt string           `json:"created_at"`
	UpdatedAt string           `json:"updated_at"`
}

// CreateProject creates a new project.
func (c *Client) CreateProject(ctx context.Context, req *ProjectRequest) (*ProjectResponse, error) {
	var resp ProjectResponse
	if err := c.doRequest(ctx, "POST", "/projects", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetProject retrieves a specific project by ID.
func (c *Client) GetProject(ctx context.Context, id int64) (*ProjectResponse, error) {
	var resp ProjectResponse
	path := fmt.Sprintf("/projects/%d", id)
	if err := c.doRequest(ctx, "GET", path, nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// UpdateProject updates an existing project.
func (c *Client) UpdateProject(ctx context.Context, id int64, req *ProjectRequest) (*ProjectResponse, error) {
	var resp ProjectResponse
	path := fmt.Sprintf("/projects/%d", id)
	if err := c.doRequest(ctx, "POST", path, req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteProject deletes a project by ID.
func (c *Client) DeleteProject(ctx context.Context, id int64) error {
	path := fmt.Sprintf("/projects/%d", id)
	return c.doRequest(ctx, "DELETE", path, nil, nil)
}

// ListProjects retrieves a paginated list of projects.
func (c *Client) ListProjects(ctx context.Context, page, perPage int) ([]*ProjectResponse, error) {
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	path := fmt.Sprintf("/projects?page=%d&per_page=%d", page, perPage)

	var paginatedResp struct {
		Data  []ProjectResponse `json:"data"`
		Meta  PaginationMeta    `json:"meta"`
		Links PaginationLinks   `json:"links"`
	}

	if err := c.doRequest(ctx, "GET", path, nil, &paginatedResp); err != nil {
		return nil, err
	}

	// Convert to pointer slice
	result := make([]*ProjectResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}
