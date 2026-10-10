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
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultTimeout = 30 * time.Second
	maxRetries     = 5
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

// doRequest performs a JSON HTTP request with authentication, marshaling, and error handling.
// It retries rate limits (429) and server errors (5xx), see send.
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

	return c.send(ctx, method, url, "application/json", jsonBody, result)
}

// send performs an HTTP request with authentication and error handling.
// It retries rate limits (429) after their Retry-After delay plus one second,
// and server errors (5xx) with a backoff that doubles on each attempt. Every
// wait adds a random jitter of up to one second, see retryJitter.
//
// Parameters:
//   - ctx: Context for cancellation
//   - method: HTTP method
//   - url: Full request URL
//   - contentType: Content-Type of the request body
//   - body: Request body, resent in full on every attempt, can be nil
//   - result: Pointer to store the response (will be unmarshaled from JSON), can be nil
//
// Returns an APIError if the request fails.
func (c *Client) send(ctx context.Context, method, url, contentType string, body []byte, result interface{}) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Create a fresh body reader per attempt, a retry must resend the full body
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
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
		req.Header.Set("Content-Type", contentType)
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

		// Log response at DEBUG level (summary)
		tflog.Debug(ctx, "Phare API response", map[string]interface{}{
			"status_code": resp.StatusCode,
			"duration_ms": duration.Milliseconds(),
		})

		// Read response body, then close it so retries don't hold it open
		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
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

		// Don't retry on client errors (4xx) - these are permanent, except a rate limit (429)
		if isPermanent(resp.StatusCode) {
			return lastErr
		}

		// Retry on server errors (5xx) and rate limits (429) if we have attempts left
		if isRetryable(resp.StatusCode) && attempt < maxRetries {
			wait := retryWait(resp, backoff) + retryJitter()
			tflog.Debug(ctx, "Retrying Phare API request", map[string]interface{}{
				"status_code": resp.StatusCode,
				"retry_in":    wait.String(),
				"attempt":     attempt + 2,
				"max":         maxRetries + 1,
			})

			// Wait for the Retry-After delay or the backoff before retrying
			select {
			case <-time.After(wait):
				// Double the backoff for the next retry
				backoff *= 2
			case <-ctx.Done():
				tflog.Error(ctx, "Request cancelled while waiting to retry", map[string]interface{}{
					"error": ctx.Err().Error(),
				})
				return ctx.Err()
			}
			continue
		}

		// Return error if no more retries or the status is not retryable
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

type FormField struct {
	Key   string
	Value string
}

// FileUpload represents a file to be uploaded in a multipart request.
type FileUpload struct {
	FieldName string
	FileName  string
	Content   io.Reader
}

// doMultipartRequest performs an HTTP multipart/form-data request with file uploads.
// It retries rate limits (429) and server errors (5xx), see send.
//
// Parameters:
//   - ctx: Context for cancellation
//   - method: HTTP method (typically POST)
//   - path: API path (e.g., "/uptime/status-pages/123")
//   - fields: Form fields to include
//   - files: Files to upload
//   - result: Pointer to store the response (will be unmarshaled from JSON), can be nil
//
// Returns an APIError if the request fails.
func (c *Client) doMultipartRequest(ctx context.Context, method, path string, fields []FormField, files []FileUpload, result interface{}) error {
	url := c.baseURL + path

	// Build the multipart body once: file contents are streams that can only be
	// read once, so every attempt must resend the same buffered bytes
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	// Add form fields
	for _, field := range fields {
		if err := writer.WriteField(field.Key, field.Value); err != nil {
			tflog.Error(ctx, "Failed to write multipart field", map[string]interface{}{
				"field": field.Key,
				"error": err.Error(),
			})
			return fmt.Errorf("failed to write multipart field %s: %w", field.Key, err)
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

	return c.send(ctx, method, url, writer.FormDataContentType(), body.Bytes(), result)
}

// isPermanent reports whether a status is a permanent client error (4xx).
// A rate limit (429) is not permanent.
func isPermanent(statusCode int) bool {
	return statusCode >= 400 && statusCode < 500 && statusCode != http.StatusTooManyRequests
}

// isRetryable reports whether a status is a server error (5xx) or a rate limit (429).
func isRetryable(statusCode int) bool {
	return statusCode >= 500 || statusCode == http.StatusTooManyRequests
}

// retryWait returns how long to wait before retrying a response. A rate limit
// waits its Retry-After delay plus one second: the API rounds the delay down to
// whole seconds, and reports 0 during the last second of the window, so
// retrying after exactly Retry-After can hit the same window again. Other
// responses, and rate limits without a Retry-After, wait the backoff.
func retryWait(resp *http.Response, backoff time.Duration) time.Duration {
	if resp.StatusCode == http.StatusTooManyRequests {
		if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds >= 0 {
			return time.Duration(seconds)*time.Second + time.Second
		}
	}
	return backoff
}

// retryJitter returns a random delay of up to one second, added to every retry
// wait so that parallel requests rate limited together don't all retry at the
// same instant and hit the limit again.
func retryJitter() time.Duration {
	return rand.N(time.Second)
}
