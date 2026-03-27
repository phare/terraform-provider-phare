package client

import (
	"context"
	"fmt"
	"io"
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
	ColorScheme          *string               `json:"color_scheme,omitempty"`
	Theme                *StatusPageTheme      `json:"theme,omitempty"`
	Components           []StatusPageComponent `json:"components"`
	Timeframe            *int64                `json:"timeframe,omitempty"`
	SubscriptionChannels []string              `json:"subscription_channels,omitempty"`
}

// StatusPageTheme represents theme customization for a status page.
type StatusPageTheme struct {
	Light       *ThemeColors `json:"light,omitempty"`
	Dark        *ThemeColors `json:"dark,omitempty"`
	Rounded     *bool        `json:"rounded,omitempty"`
	BorderWidth *int64       `json:"border_width,omitempty"`
}

// ThemeColors represents color values for a theme (light or dark).
type ThemeColors struct {
	Operational         string `json:"operational"`
	DegradedPerformance string `json:"degraded_performance"`
	PartialOutage       string `json:"partial_outage"`
	MajorOutage         string `json:"major_outage"`
	Maintenance         string `json:"maintenance"`
	Empty               string `json:"empty"`
	Background          string `json:"background"`
	Foreground          string `json:"foreground"`
	ForegroundMuted     string `json:"foreground_muted"`
	BackgroundCard      string `json:"background_card"`
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
	ColorScheme          *string               `json:"color_scheme,omitempty"`
	Theme                *StatusPageTheme      `json:"theme,omitempty"`
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

	result := make([]*StatusPageResponse, len(paginatedResp.Data))
	for i := range paginatedResp.Data {
		result[i] = &paginatedResp.Data[i]
	}

	return result, nil
}

// StatusPageFileUpload represents a file to upload to a status page.
type StatusPageFileUpload struct {
	FieldName string    // One of: logo_light, logo_dark, favicon_light, favicon_dark
	FileName  string    // Original filename for Content-Disposition
	Content   io.Reader // File content
}

// UpdateStatusPageWithFiles updates a status page with file uploads/removals.
// The API requires all status page fields to be sent along with file operations.
// For uploads, provide StatusPageFileUpload entries with Content.
// For removals, provide field names in the removals slice (sends "false" to API).
func (c *Client) UpdateStatusPageWithFiles(ctx context.Context, id int64, req *StatusPageRequest, uploads []StatusPageFileUpload, removals []string) (*StatusPageResponse, error) {
	path := fmt.Sprintf("/uptime/status-pages/%d", id)

	// Build fields map with all status page data
	fields := make(map[string]string)

	// Add required fields
	fields["name"] = req.Name
	fields["title"] = req.Title
	fields["description"] = req.Description
	fields["website_url"] = req.WebsiteURL
	if req.SearchEngineIndexed {
		fields["search_engine_indexed"] = "1"
	} else {
		fields["search_engine_indexed"] = "0"
	}

	// Add optional fields
	if req.Subdomain != nil {
		fields["subdomain"] = *req.Subdomain
	}
	if req.Domain != nil {
		fields["domain"] = *req.Domain
	}
	if req.ColorScheme != nil {
		fields["color_scheme"] = *req.ColorScheme
	}
	if req.Timeframe != nil {
		fields["timeframe"] = fmt.Sprintf("%d", *req.Timeframe)
	}

	// Add components as indexed fields
	for i, comp := range req.Components {
		fields[fmt.Sprintf("components[%d][componentable_type]", i)] = comp.ComponentableType
		fields[fmt.Sprintf("components[%d][componentable_id]", i)] = fmt.Sprintf("%d", comp.ComponentableID)
	}

	// Add subscription channels
	for i, channel := range req.SubscriptionChannels {
		fields[fmt.Sprintf("subscription_channels[%d]", i)] = channel
	}

	// Add theme fields if present
	if req.Theme != nil {
		if req.Theme.Rounded != nil {
			if *req.Theme.Rounded {
				fields["theme[rounded]"] = "1"
			} else {
				fields["theme[rounded]"] = "0"
			}
		}
		if req.Theme.BorderWidth != nil {
			fields["theme[border_width]"] = fmt.Sprintf("%d", *req.Theme.BorderWidth)
		}
		if req.Theme.Light != nil {
			fields["theme[light][operational]"] = req.Theme.Light.Operational
			fields["theme[light][degraded_performance]"] = req.Theme.Light.DegradedPerformance
			fields["theme[light][partial_outage]"] = req.Theme.Light.PartialOutage
			fields["theme[light][major_outage]"] = req.Theme.Light.MajorOutage
			fields["theme[light][maintenance]"] = req.Theme.Light.Maintenance
			fields["theme[light][empty]"] = req.Theme.Light.Empty
			fields["theme[light][background]"] = req.Theme.Light.Background
			fields["theme[light][foreground]"] = req.Theme.Light.Foreground
			fields["theme[light][foreground_muted]"] = req.Theme.Light.ForegroundMuted
			fields["theme[light][background_card]"] = req.Theme.Light.BackgroundCard
		}
		if req.Theme.Dark != nil {
			fields["theme[dark][operational]"] = req.Theme.Dark.Operational
			fields["theme[dark][degraded_performance]"] = req.Theme.Dark.DegradedPerformance
			fields["theme[dark][partial_outage]"] = req.Theme.Dark.PartialOutage
			fields["theme[dark][major_outage]"] = req.Theme.Dark.MajorOutage
			fields["theme[dark][maintenance]"] = req.Theme.Dark.Maintenance
			fields["theme[dark][empty]"] = req.Theme.Dark.Empty
			fields["theme[dark][background]"] = req.Theme.Dark.Background
			fields["theme[dark][foreground]"] = req.Theme.Dark.Foreground
			fields["theme[dark][foreground_muted]"] = req.Theme.Dark.ForegroundMuted
			fields["theme[dark][background_card]"] = req.Theme.Dark.BackgroundCard
		}
	}

	// Add file removals (send "false" to remove)
	for _, fieldName := range removals {
		fields[fieldName] = "false"
	}

	// Build files slice for uploads
	files := make([]FileUpload, len(uploads))
	for i, upload := range uploads {
		files[i] = FileUpload{
			FieldName: upload.FieldName,
			FileName:  upload.FileName,
			Content:   upload.Content,
		}
	}

	// Use POST for the update (API uses POST not PUT)
	var resp StatusPageResponse
	if err := c.doMultipartRequest(ctx, "POST", path, fields, files, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
