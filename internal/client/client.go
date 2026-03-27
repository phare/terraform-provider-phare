// Package client provides an HTTP client for the Phare API.
//
// The client handles authentication, request/response marshaling,
// and error handling for all API operations.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 3
	initialBackoff = 1 * time.Second
)

// PaginatedResponse represents a paginated API response.
type PaginatedResponse struct {
	Meta  PaginationMeta  `json:"meta"`
	Links PaginationLinks `json:"links"`
}

// PaginationMeta contains pagination metadata.
type PaginationMeta struct {
	CurrentPage int    `json:"current_page"`
	From        int    `json:"from"`
	To          int    `json:"to"`
	PerPage     int    `json:"per_page"`
	Path        string `json:"path"`
}

// PaginationLinks contains pagination links.
type PaginationLinks struct {
	First *string `json:"first"`
	Last  *string `json:"last"`
	Prev  *string `json:"prev"`
	Next  *string `json:"next"`
}

// Client is the main HTTP client for the Phare API.
// It manages authentication and provides methods for all API operations.
type Client struct {
	baseURL          string
	token            string
	httpClient       *http.Client
	userAgent        string
	projectID        string
	projectSlug      string
	providerVersion  string
	terraformVersion string
	isProjectScoped  bool
}

// NewClient creates a new Phare API client.
//
// Parameters:
//   - baseURL: The base URL of the Phare API (e.g., "https://api.phare.io")
//   - token: The API authentication token
//   - timeout: HTTP client timeout duration
//   - projectID: Optional project ID for scoping requests (organization-scoped keys)
//   - projectSlug: Optional project slug for scoping requests (organization-scoped keys)
//   - providerVersion: The version of the Terraform provider
//   - terraformVersion: The version of Terraform (if available)
//   - isProjectScoped: Whether the API key is project-scoped (starts with "pha_" but not "pha_org_")
//
// Returns an error if the configuration is invalid.
func NewClient(baseURL, token string, timeout time.Duration, projectID, projectSlug, providerVersion, terraformVersion string, isProjectScoped bool) (*Client, error) {
	if baseURL == "" {
		return nil, fmt.Errorf("baseURL cannot be empty")
	}
	if token == "" {
		return nil, fmt.Errorf("token cannot be empty")
	}
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	// Validate that only one project identifier is provided
	if projectID != "" && projectSlug != "" {
		return nil, fmt.Errorf("cannot specify both project_id and project_slug")
	}

	// Build dynamic user agent
	userAgent := buildUserAgent(providerVersion, terraformVersion)

	return &Client{
		baseURL:          baseURL,
		token:            token,
		httpClient:       &http.Client{Timeout: timeout},
		userAgent:        userAgent,
		projectID:        projectID,
		projectSlug:      projectSlug,
		providerVersion:  providerVersion,
		terraformVersion: terraformVersion,
		isProjectScoped:  isProjectScoped,
	}, nil
}

// buildUserAgent constructs a user agent string with version information.
// Format: terraform-provider-phare/VERSION (Terraform VERSION; Go/VERSION; +https://phare.io)
func buildUserAgent(providerVersion, terraformVersion string) string {
	if providerVersion == "" {
		providerVersion = "dev"
	}
	if terraformVersion == "" {
		terraformVersion = "unknown"
	}

	return fmt.Sprintf("terraform-provider-phare/%s (Terraform/%s; +https://registry.terraform.io/providers/phare/phare)",
		providerVersion, terraformVersion)
}

// doRequest performs an HTTP request with authentication, marshaling, and error handling.
// It automatically retries on server errors (5xx) with exponential backoff.
//
// Parameters:
//   - ctx: Context for cancellation
//   - method: HTTP method (GET, POST, PUT, DELETE, etc.)
//   - path: API path (e.g., "/alert-rules")
//   - body: Request body (will be marshaled to JSON), can be nil
//   - result: Pointer to store the response (will be unmarshaled from JSON), can be nil
//
// Returns an APIError if the request fails.
func (c *Client) doRequest(ctx context.Context, method, path string, body, result interface{}) error {
	url := c.baseURL + path

	// Marshal request body
	var bodyReader io.Reader
	var jsonBody []byte
	if body != nil {
		var err error
		jsonBody, err = json.Marshal(body)
		if err != nil {
			tflog.Error(ctx, "Failed to marshal request body", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Log request at DEBUG level (summary only)
	tflog.Debug(ctx, "Phare API request", map[string]interface{}{
		"method": method,
		"url":    url,
	})

	// Log request body at TRACE level (full details)
	if body != nil {
		tflog.Trace(ctx, "Phare API request body", map[string]interface{}{
			"body": string(jsonBody),
		})
	}

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Log retry attempts
		if attempt > 0 {
			tflog.Debug(ctx, "Retrying Phare API request", map[string]interface{}{
				"attempt": attempt + 1,
				"max":     maxRetries + 1,
				"backoff": backoff.String(),
			})
		}

		// Create request
		req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
		if err != nil {
			tflog.Error(ctx, "Failed to create HTTP request", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		// Add project scoping headers if configured
		if c.projectID != "" {
			req.Header.Set("X-Phare-Project-Id", c.projectID)
		}
		if c.projectSlug != "" {
			req.Header.Set("X-Phare-Project-Slug", c.projectSlug)
		}

		// Perform request with timing
		startTime := time.Now()
		resp, err := c.httpClient.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			tflog.Error(ctx, "Phare API request failed", map[string]interface{}{
				"error":       err.Error(),
				"url":         url,
				"duration_ms": duration.Milliseconds(),
			})
			return fmt.Errorf("failed to perform request: %w", err)
		}
		defer resp.Body.Close()

		// Log response at DEBUG level (summary)
		tflog.Debug(ctx, "Phare API response", map[string]interface{}{
			"status_code": resp.StatusCode,
			"duration_ms": duration.Milliseconds(),
		})

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			tflog.Error(ctx, "Failed to read response body", map[string]interface{}{
				"error":       err.Error(),
				"status_code": resp.StatusCode,
			})
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Log response body at TRACE level (full details)
		if len(respBody) > 0 {
			tflog.Trace(ctx, "Phare API response body", map[string]interface{}{
				"body":        string(respBody),
				"status_code": resp.StatusCode,
			})
		}

		// Handle successful responses
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && resp.StatusCode != http.StatusNoContent {
				if len(respBody) > 0 {
					if err := json.Unmarshal(respBody, result); err != nil {
						tflog.Error(ctx, "Failed to unmarshal response", map[string]interface{}{
							"error": err.Error(),
						})
						return fmt.Errorf("failed to unmarshal response: %w", err)
					}
				}
			}
			return nil
		}

		// Parse error response
		lastErr = parseAPIError(resp.StatusCode, respBody)

		// Log HTTP error responses at WARN level
		tflog.Warn(ctx, "Phare API returned error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"error":       lastErr.Error(),
			"body":        string(respBody),
		})

		// Don't retry on client errors (4xx) - these are permanent
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}

		// Retry on server errors (5xx) if we have attempts left
		if resp.StatusCode >= 500 && attempt < maxRetries {
			tflog.Debug(ctx, "Server error, will retry", map[string]interface{}{
				"status_code":   resp.StatusCode,
				"retry_in":      backoff.String(),
				"attempts_left": maxRetries - attempt,
			})

			// Wait with exponential backoff before retrying
			select {
			case <-time.After(backoff):
				// Calculate next backoff with exponential increase
				backoff = time.Duration(float64(backoff) * math.Pow(2, float64(attempt)))
			case <-ctx.Done():
				tflog.Error(ctx, "Request cancelled during retry backoff", map[string]interface{}{
					"error": ctx.Err().Error(),
				})
				return ctx.Err()
			}
			continue
		}

		// Return error if no more retries or not a server error
		return lastErr
	}

	return lastErr
}

// GetConfig returns the client's configuration parameters.
func (c *Client) GetConfig() (string, string, time.Duration) {
	return c.baseURL, c.token, c.httpClient.Timeout
}

// GetProjectScope returns the client's project scope configuration.
func (c *Client) GetProjectScope() (string, string) {
	return c.projectID, c.projectSlug
}

// IsProjectScoped returns true if the client is using a project-scoped API key.
func (c *Client) IsProjectScoped() bool {
	return c.isProjectScoped
}

// GetVersions returns the client's version information.
func (c *Client) GetVersions() (string, string) {
	return c.providerVersion, c.terraformVersion
}

// WithProjectScope creates a new client with the specified project scope.
// This allows resources to override the provider-level project scope with a resource-level scope.
func (c *Client) WithProjectScope(projectID, projectSlug string) (*Client, error) {
	// Validate that only one project identifier is provided
	if projectID != "" && projectSlug != "" {
		return nil, fmt.Errorf("cannot specify both project_id and project_slug")
	}

	// Create a new client with the same configuration but different project scope
	return NewClient(
		c.baseURL,
		c.token,
		c.httpClient.Timeout,
		projectID,
		projectSlug,
		c.providerVersion,
		c.terraformVersion,
		c.isProjectScoped,
	)
}

// FileUpload represents a file to be uploaded in a multipart request.
type FileUpload struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

// doMultipartRequest performs an HTTP multipart/form-data request with file uploads.
// It automatically retries on server errors (5xx) with exponential backoff.
//
// Parameters:
//   - ctx: Context for cancellation
//   - method: HTTP method (typically POST)
//   - path: API path (e.g., "/uptime/status-pages/123")
//   - fields: Form fields to include (field name -> value)
//   - files: Files to upload
//   - result: Pointer to store the response (will be unmarshaled from JSON), can be nil
//
// Returns an APIError if the request fails.
func (c *Client) doMultipartRequest(ctx context.Context, method, path string, fields map[string]string, files []FileUpload, result interface{}) error {
	url := c.baseURL + path

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Log retry attempts
		if attempt > 0 {
			tflog.Debug(ctx, "Retrying Phare API multipart request", map[string]interface{}{
				"attempt": attempt + 1,
				"max":     maxRetries + 1,
				"backoff": backoff.String(),
			})
		}

		// Create multipart body
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)

		// Add form fields
		for fieldName, fieldValue := range fields {
			if err := writer.WriteField(fieldName, fieldValue); err != nil {
				tflog.Error(ctx, "Failed to write multipart field", map[string]interface{}{
					"field": fieldName,
					"error": err.Error(),
				})
				return fmt.Errorf("failed to write multipart field %s: %w", fieldName, err)
			}
		}

		// Add files
		for _, file := range files {
			part, err := writer.CreateFormFile(file.FieldName, filepath.Base(file.FileName))
			if err != nil {
				tflog.Error(ctx, "Failed to create form file", map[string]interface{}{
					"field": file.FieldName,
					"error": err.Error(),
				})
				return fmt.Errorf("failed to create form file %s: %w", file.FieldName, err)
			}
			if _, err := io.Copy(part, file.Content); err != nil {
				tflog.Error(ctx, "Failed to copy file content", map[string]interface{}{
					"field": file.FieldName,
					"error": err.Error(),
				})
				return fmt.Errorf("failed to copy file content for %s: %w", file.FieldName, err)
			}
		}

		// Close the multipart writer to finalize the body
		if err := writer.Close(); err != nil {
			tflog.Error(ctx, "Failed to close multipart writer", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to close multipart writer: %w", err)
		}

		// Log request at DEBUG level (summary only)
		tflog.Debug(ctx, "Phare API multipart request", map[string]interface{}{
			"method":      method,
			"url":         url,
			"field_count": len(fields),
			"file_count":  len(files),
		})

		// Create request
		req, err := http.NewRequestWithContext(ctx, method, url, &body)
		if err != nil {
			tflog.Error(ctx, "Failed to create HTTP request", map[string]interface{}{
				"error": err.Error(),
			})
			return fmt.Errorf("failed to create request: %w", err)
		}

		// Set headers
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)

		// Add project scoping headers if configured
		if c.projectID != "" {
			req.Header.Set("X-Phare-Project-Id", c.projectID)
		}
		if c.projectSlug != "" {
			req.Header.Set("X-Phare-Project-Slug", c.projectSlug)
		}

		// Perform request with timing
		startTime := time.Now()
		resp, err := c.httpClient.Do(req)
		duration := time.Since(startTime)

		if err != nil {
			tflog.Error(ctx, "Phare API multipart request failed", map[string]interface{}{
				"error":       err.Error(),
				"url":         url,
				"duration_ms": duration.Milliseconds(),
			})
			return fmt.Errorf("failed to perform request: %w", err)
		}
		defer resp.Body.Close()

		// Log response at DEBUG level (summary)
		tflog.Debug(ctx, "Phare API multipart response", map[string]interface{}{
			"status_code": resp.StatusCode,
			"duration_ms": duration.Milliseconds(),
		})

		// Read response body
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			tflog.Error(ctx, "Failed to read response body", map[string]interface{}{
				"error":       err.Error(),
				"status_code": resp.StatusCode,
			})
			return fmt.Errorf("failed to read response body: %w", err)
		}

		// Log response body at TRACE level (full details)
		if len(respBody) > 0 {
			tflog.Trace(ctx, "Phare API multipart response body", map[string]interface{}{
				"body":        string(respBody),
				"status_code": resp.StatusCode,
			})
		}

		// Handle successful responses
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if result != nil && resp.StatusCode != http.StatusNoContent {
				if len(respBody) > 0 {
					if err := json.Unmarshal(respBody, result); err != nil {
						tflog.Error(ctx, "Failed to unmarshal response", map[string]interface{}{
							"error": err.Error(),
						})
						return fmt.Errorf("failed to unmarshal response: %w", err)
					}
				}
			}
			return nil
		}

		// Parse error response
		lastErr = parseAPIError(resp.StatusCode, respBody)

		// Log HTTP error responses at WARN level
		tflog.Warn(ctx, "Phare API multipart returned error", map[string]interface{}{
			"status_code": resp.StatusCode,
			"error":       lastErr.Error(),
			"body":        string(respBody),
		})

		// Don't retry on client errors (4xx) - these are permanent
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			return lastErr
		}

		// Retry on server errors (5xx) if we have attempts left
		if resp.StatusCode >= 500 && attempt < maxRetries {
			tflog.Debug(ctx, "Server error, will retry", map[string]interface{}{
				"status_code":   resp.StatusCode,
				"retry_in":      backoff.String(),
				"attempts_left": maxRetries - attempt,
			})

			// Wait with exponential backoff before retrying
			select {
			case <-time.After(backoff):
				// Calculate next backoff with exponential increase
				backoff = time.Duration(float64(backoff) * math.Pow(2, float64(attempt)))
			case <-ctx.Done():
				tflog.Error(ctx, "Request cancelled during retry backoff", map[string]interface{}{
					"error": ctx.Err().Error(),
				})
				return ctx.Err()
			}
			continue
		}

		// Return error if no more retries or not a server error
		return lastErr
	}

	return lastErr
}
