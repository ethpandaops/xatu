package blockprint

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		headers  map[string]string
	}{
		{
			name:     "creates_client_with_endpoint_and_no_headers",
			endpoint: "https://api.blockprint.sigp.io",
			headers:  nil,
		},
		{
			name:     "creates_client_with_endpoint_and_empty_headers",
			endpoint: "https://api.blockprint.sigp.io",
			headers:  map[string]string{},
		},
		{
			name:     "creates_client_with_endpoint_and_headers",
			endpoint: "https://api.blockprint.sigp.io",
			headers: map[string]string{
				"User-Agent":    "xatu-cannon/1.0",
				"Authorization": "Bearer token",
			},
		},
		{
			name:     "creates_client_with_localhost_endpoint",
			endpoint: "http://localhost:8080",
			headers:  map[string]string{"X-Test": "test-value"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.endpoint, tt.headers)

			assert.NotNil(t, client)
			assert.Equal(t, tt.endpoint, client.endpoint)
			assert.Equal(t, http.DefaultClient, client.httpClient)
			assert.Equal(t, tt.headers, client.headers)
		})
	}
}

func TestClient_Get(t *testing.T) {
	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		path           string
		headers        map[string]string
		expectedData   json.RawMessage
		expectedError  string
		validateRequest func(*testing.T, *http.Request)
	}{
		{
			name: "successful_get_request_with_json_response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"result": "success", "data": [1, 2, 3]}`))
			},
			path:         "/api/v1/test",
			headers:      nil,
			expectedData: json.RawMessage(`{"result": "success", "data": [1, 2, 3]}`),
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/api/v1/test", r.URL.Path)
			},
		},
		{
			name: "successful_get_request_with_headers",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Verify headers were set
				assert.Equal(t, "xatu-cannon/1.0", r.Header.Get("User-Agent"))
				assert.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"authenticated": true}`))
			},
			path: "/api/v1/protected",
			headers: map[string]string{
				"User-Agent":    "xatu-cannon/1.0",
				"Authorization": "Bearer secret-token",
			},
			expectedData: json.RawMessage(`{"authenticated": true}`),
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
				assert.Equal(t, "/api/v1/protected", r.URL.Path)
				assert.Equal(t, "xatu-cannon/1.0", r.Header.Get("User-Agent"))
				assert.Equal(t, "Bearer secret-token", r.Header.Get("Authorization"))
			},
		},
		{
			name: "handles_utf8_bom_prefix",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				// Include UTF-8 BOM prefix
				_, _ = w.Write([]byte("\xef\xbb\xbf{\"message\": \"with BOM\"}"))
			},
			path:         "/api/v1/bom",
			headers:      nil,
			expectedData: json.RawMessage(`{"message": "with BOM"}`),
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
		{
			name: "handles_404_not_found",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Not Found"))
			},
			path:          "/api/v1/nonexistent",
			headers:       nil,
			expectedError: "status code: 404",
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
		{
			name: "handles_500_internal_server_error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte("Internal Server Error"))
			},
			path:          "/api/v1/error",
			headers:       nil,
			expectedError: "status code: 500",
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
		{
			name: "handles_invalid_json_response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`invalid json {`))
			},
			path:          "/api/v1/invalid",
			headers:       nil,
			expectedError: "invalid character",
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
		{
			name: "handles_empty_response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`null`))
			},
			path:         "/api/v1/empty",
			headers:      nil,
			expectedData: json.RawMessage(`null`),
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
		{
			name: "handles_array_response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[{"id": 1}, {"id": 2}]`))
			},
			path:         "/api/v1/array",
			headers:      nil,
			expectedData: json.RawMessage(`[{"id": 1}, {"id": 2}]`),
			validateRequest: func(t *testing.T, r *http.Request) {
				assert.Equal(t, "GET", r.Method)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.validateRequest != nil {
					tt.validateRequest(t, r)
				}
				tt.serverHandler(w, r)
			}))
			defer server.Close()

			// Create client
			client := NewClient(server.URL, tt.headers)

			// Make request
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			result, err := client.get(ctx, tt.path)

			if tt.expectedError != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedData, result)
			}
		})
	}
}

func TestClient_Get_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"result": "delayed"}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, nil)

	// Create context that times out quickly
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.get(ctx, "/api/v1/delayed")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestClient_Get_NetworkError(t *testing.T) {
	// Use an invalid endpoint that will cause a network error
	client := NewClient("http://nonexistent.invalid:9999", nil)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := client.get(ctx, "/api/v1/test")
	assert.Error(t, err)
	// Network errors might vary by system, so just check that an error occurred
}

func TestClient_Get_HeaderOverrides(t *testing.T) {
	// Test that client headers override any default headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify custom headers are set and override defaults
		assert.Equal(t, "custom-agent", r.Header.Get("User-Agent"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom"))
		
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	headers := map[string]string{
		"User-Agent": "custom-agent",
		"X-Custom":   "custom-value",
	}
	
	client := NewClient(server.URL, headers)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.get(ctx, "/test")
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{"success": true}`), result)
}

func TestClient_Get_MultipleHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify multiple headers are all set correctly
		assert.Equal(t, "xatu-cannon", r.Header.Get("User-Agent"))
		assert.Equal(t, "Bearer token123", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		assert.Equal(t, "custom-client", r.Header.Get("X-Client"))
		
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"headers": "verified"}`))
	}))
	defer server.Close()

	headers := map[string]string{
		"User-Agent":    "xatu-cannon",
		"Authorization": "Bearer token123",
		"Accept":        "application/json",
		"X-Client":      "custom-client",
	}
	
	client := NewClient(server.URL, headers)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := client.get(ctx, "/test")
	require.NoError(t, err)
	assert.Equal(t, json.RawMessage(`{"headers": "verified"}`), result)
}