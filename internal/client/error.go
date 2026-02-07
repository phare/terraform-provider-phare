package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// APIError represents an error returned by the Phare API.
// It includes the HTTP status code and error message(s) from the API.
type APIError struct {
	StatusCode int
	Message    string
	Errors     map[string][]string // Field-level validation errors (422)
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		// Format validation errors as user-friendly messages
		var errMsgs string
		for field, messages := range e.Errors {
			for _, msg := range messages {
				if errMsgs != "" {
					errMsgs += "; "
				}
				errMsgs += fmt.Sprintf("%s: %s", field, msg)
			}
		}
		if e.Message != "" {
			return fmt.Sprintf("API error %d: %s (%s)", e.StatusCode, e.Message, errMsgs)
		}
		return fmt.Sprintf("API error %d: %s", e.StatusCode, errMsgs)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

// IsNotFoundError returns true if the error is a 404 Not Found error.
// This is useful for determining if a resource has been deleted outside of Terraform.
func IsNotFoundError(err error) bool {
	if apiErr, ok := err.(*APIError); ok {
		return apiErr.StatusCode == http.StatusNotFound
	}
	return false
}

// parseAPIError parses an API error response into an APIError struct.
func parseAPIError(statusCode int, body []byte) error {
	var errorResp struct {
		Message string              `json:"message"`
		Errors  map[string][]string `json:"errors,omitempty"`
	}

	if err := json.Unmarshal(body, &errorResp); err != nil {
		// If we can't parse the error response, return the raw body
		return &APIError{
			StatusCode: statusCode,
			Message:    string(body),
		}
	}

	return &APIError{
		StatusCode: statusCode,
		Message:    errorResp.Message,
		Errors:     errorResp.Errors,
	}
}
