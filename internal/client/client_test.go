package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"testing/synctest"
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

// handlerTransport serves requests in memory with a handler. Unlike a real
// httptest server it has no network I/O, so the tests can run in a synctest
// bubble, where the fake clock skips the retry waits instantly.
type handlerTransport struct {
	handler http.Handler
}

func (h handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	if r.Body == nil {
		r.Body = http.NoBody
	}
	rec := httptest.NewRecorder()
	h.handler.ServeHTTP(rec, r)
	return rec.Result(), nil
}

// newTestClient returns a client whose requests are served by handler.
func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	c, err := NewClient("https://api.phare.test", "token", 10*time.Second, "", "", "1.0", "1.0", true)
	if err != nil {
		t.Fatal(err)
	}
	c.httpClient.Transport = handlerTransport{handler}
	return c
}

func TestDoRequest_RetriesRateLimit(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		attempts := 0
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		if err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, nil); err != nil {
			t.Fatalf("doRequest() error = %v", err)
		}
		if attempts != 2 {
			t.Errorf("attempts = %d, want 2", attempts)
		}
	})
}

func TestDoRequest_RetryResendsBody(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var bodies []string
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			b, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("read body: %v", err)
			}
			bodies = append(bodies, string(b))
			if len(bodies) == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		body := map[string]string{"name": "hello"}
		if err := c.doRequest(context.Background(), http.MethodPost, "/test", body, nil); err != nil {
			t.Fatalf("doRequest() error = %v", err)
		}

		want := `{"name":"hello"}`
		if len(bodies) != 2 {
			t.Fatalf("attempts = %d, want 2", len(bodies))
		}
		for i, got := range bodies {
			if got != want {
				t.Errorf("attempt %d body = %q, want %q", i+1, got, want)
			}
		}
	})
}

func TestDoMultipartRequest_RetryResendsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "logo.png")
	if err := os.WriteFile(path, []byte("PNGDATA"), 0o600); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	synctest.Test(t, func(t *testing.T) {
		type attempt struct{ field, file string }
		var attempts []attempt
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Errorf("parse multipart: %v", err)
			}
			var content string
			if f, _, err := r.FormFile("logo"); err == nil {
				b, _ := io.ReadAll(f)
				content = string(b)
				f.Close()
			}
			attempts = append(attempts, attempt{r.FormValue("name"), content})
			if len(attempts) == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		fields := []FormField{{"name", "Status"}}
		files := []FileUpload{{FieldName: "logo", FileName: path, Content: file}}
		if err := c.doMultipartRequest(context.Background(), http.MethodPost, "/test", fields, files, nil); err != nil {
			t.Fatalf("doMultipartRequest() error = %v", err)
		}

		if len(attempts) != 2 {
			t.Fatalf("attempts = %d, want 2", len(attempts))
		}
		for i, got := range attempts {
			if got.field != "Status" || got.file != "PNGDATA" {
				t.Errorf("attempt %d sent name=%q logo=%q, want name=%q logo=%q", i+1, got.field, got.file, "Status", "PNGDATA")
			}
		}
	})
}

func TestDoRequest_RateLimitWaitsPastZeroRetryAfter(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var times []time.Time
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			times = append(times, time.Now())
			if len(times) == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		})

		if err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, nil); err != nil {
			t.Fatalf("doRequest() error = %v", err)
		}

		if len(times) != 2 {
			t.Fatalf("attempts = %d, want 2", len(times))
		}
		// The bubble's fake clock only advances by the retry wait: 1s plus up to 1s of jitter
		if gap := times[1].Sub(times[0]); gap < time.Second || gap >= 2*time.Second {
			t.Errorf("retried after %v, want between 1s and 2s after Retry-After: 0", gap)
		}
	})
}

func TestDoRequest_ServerErrorBackoffDoubles(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var times []time.Time
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			times = append(times, time.Now())
			w.WriteHeader(http.StatusServiceUnavailable)
		})

		if err := c.doRequest(context.Background(), http.MethodGet, "/test", nil, nil); err == nil {
			t.Fatal("doRequest() error = nil, want the 503 after the last retry")
		}

		if len(times) != maxRetries+1 {
			t.Fatalf("attempts = %d, want %d", len(times), maxRetries+1)
		}
		// Each wait is the backoff plus up to 1s of jitter
		want := []time.Duration{1 * time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second, 16 * time.Second}
		jittered := false
		for i, base := range want {
			gap := times[i+1].Sub(times[i])
			if gap < base || gap >= base+time.Second {
				t.Errorf("wait before attempt %d = %v, want between %v and %v", i+2, gap, base, base+time.Second)
			}
			jittered = jittered || gap != base
		}
		if !jittered {
			t.Error("no wait had jitter added")
		}
	})
}

func TestRetryWait(t *testing.T) {
	backoff := 3 * time.Second
	tests := []struct {
		name       string
		status     int
		retryAfter string
		want       time.Duration
	}{
		{"rate limit adds one second", http.StatusTooManyRequests, "5", 6 * time.Second},
		{"rate limit at window end", http.StatusTooManyRequests, "0", time.Second},
		{"rate limit without header", http.StatusTooManyRequests, "", backoff},
		{"rate limit with invalid header", http.StatusTooManyRequests, "soon", backoff},
		{"rate limit with negative header", http.StatusTooManyRequests, "-1", backoff},
		{"server error ignores header", http.StatusServiceUnavailable, "5", backoff},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tt.status, Header: http.Header{}}
			if tt.retryAfter != "" {
				resp.Header.Set("Retry-After", tt.retryAfter)
			}
			if got := retryWait(resp, backoff); got != tt.want {
				t.Errorf("retryWait() = %v, want %v", got, tt.want)
			}
		})
	}
}
