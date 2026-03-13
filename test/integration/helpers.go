package integration

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

// RequestOptions contains optional parameters for HTTP requests
type RequestOptions struct {
	Headers map[string]string
	Body    []byte
}

// MakeRequest makes an HTTP request to the test server
func MakeRequest(t *testing.T, ts *TestServer, method, path string, opts *RequestOptions) *http.Response {
	t.Helper()

	var body io.Reader
	if opts != nil && len(opts.Body) > 0 {
		body = bytes.NewReader(opts.Body)
	}

	req, err := http.NewRequest(method, ts.Server.URL+path, body)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}

	// Set default headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Apply custom headers
	if opts != nil && opts.Headers != nil {
		for key, value := range opts.Headers {
			req.Header.Set(key, value)
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to execute request: %v", err)
	}

	return resp
}

// MakeAuthenticatedRequest makes an authenticated HTTP request
func MakeAuthenticatedRequest(t *testing.T, ts *TestServer, method, path string, opts *RequestOptions) *http.Response {
	t.Helper()

	if opts == nil {
		opts = &RequestOptions{}
	}
	if opts.Headers == nil {
		opts.Headers = make(map[string]string)
	}

	opts.Headers["X-Client-ID"] = TestClientID
	opts.Headers["X-Client-Secret"] = TestClientSecret

	return MakeRequest(t, ts, method, path, opts)
}

// ParseResponse parses the JSON response into the given target
func ParseResponse(t *testing.T, resp *http.Response, target interface{}) {
	t.Helper()

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	if len(body) == 0 {
		t.Fatal("Response body is empty")
	}

	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("Failed to unmarshal response: %v\nBody: %s", err, string(body))
	}
}

// ParseResponseRaw returns the response body as bytes
func ParseResponseRaw(t *testing.T, resp *http.Response) []byte {
	t.Helper()

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read response body: %v", err)
	}

	return body
}

// AssertStatusCode checks if the response status code matches expected
func AssertStatusCode(t *testing.T, resp *http.Response, expected int) {
	t.Helper()

	if resp.StatusCode != expected {
		t.Errorf("Expected status code %d, got %d", expected, resp.StatusCode)
	}
}

// AssertContentType checks if the response Content-Type matches expected
func AssertContentType(t *testing.T, resp *http.Response, expected string) {
	t.Helper()

	contentType := resp.Header.Get("Content-Type")
	if contentType != expected {
		t.Errorf("Expected Content-Type %q, got %q", expected, contentType)
	}
}

// AssertHeader checks if a response header exists and matches expected value
func AssertHeader(t *testing.T, resp *http.Response, header, expected string) {
	t.Helper()

	value := resp.Header.Get(header)
	if value != expected {
		t.Errorf("Expected header %q to be %q, got %q", header, expected, value)
	}
}

// AssertHeaderExists checks if a response header exists
func AssertHeaderExists(t *testing.T, resp *http.Response, header string) {
	t.Helper()

	if resp.Header.Get(header) == "" {
		t.Errorf("Expected header %q to exist", header)
	}
}
