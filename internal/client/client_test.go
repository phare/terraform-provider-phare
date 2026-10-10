package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBuildUserAgent(t *testing.T) {
	tests := []struct {
		name             string
		providerVersion  string
		terraformVersion string
		expectedContains []string
	}{
		{
			name:             "with both versions",
			providerVersion:  "0.1.0",
			terraformVersion: "1.14.3",
			expectedContains: []string{
				"terraform-provider-phare/0.1.0",
				"Terraform/1.14.3",
				"+https://registry.terraform.io/providers/phare/phare",
			},
		},
		{
			name:             "with empty versions",
			providerVersion:  "",
			terraformVersion: "",
			expectedContains: []string{
				"terraform-provider-phare/dev",
				"Terraform/unknown",
			},
		},
		{
			name:             "with dev version",
			providerVersion:  "dev",
			terraformVersion: "1.5.0",
			expectedContains: []string{
				"terraform-provider-phare/dev",
				"Terraform/1.5.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userAgent := buildUserAgent(tt.providerVersion, tt.terraformVersion)

			for _, expected := range tt.expectedContains {
				if !contains(userAgent, expected) {
					t.Errorf("buildUserAgent() = %q, should contain %q", userAgent, expected)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDoRequest_RetriesRateLimit(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c, err := NewClient(server.URL, "token", 10*time.Second, "", "", "1.0", "1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, nil); err != nil {
		t.Fatalf("doRequest() error = %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}
