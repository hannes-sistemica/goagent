package builtin_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-server/internal/tools"
	"agent-server/internal/tools/builtin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPGetTool(t *testing.T) {
	// Create a test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "Hello, World!", "status": "success"}`))
		case "/text":
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello, World!"))
		case "/error":
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("Internal Server Error"))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not Found"))
		}
	}))
	defer testServer.Close()

	httpGet := builtin.NewHTTPGetTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "http_get", httpGet.Name())
		
		schema := httpGet.Schema()
		assert.Equal(t, "http_get", schema.Name)
		assert.Contains(t, schema.Description, "HTTP GET")
		assert.Len(t, schema.Parameters, 3)
		
		// Check URL parameter
		urlParam := schema.Parameters[0]
		assert.Equal(t, "url", urlParam.Name)
		assert.Equal(t, "string", urlParam.Type)
		assert.True(t, urlParam.Required)
		assert.NotEmpty(t, urlParam.Pattern)
	})

	t.Run("IsAvailable", func(t *testing.T) {
		ctx := context.Background()
		assert.True(t, httpGet.IsAvailable(ctx))
	})

	t.Run("Successful JSON Request", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/json",
			"headers": map[string]interface{}{
				"Accept": "application/json",
			},
			"timeout": 10.0,
		}

		result := httpGet.Execute(ctx, input)
		assert.True(t, result.Success, "Request should succeed")
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, 200, data["status_code"])
		assert.Equal(t, "application/json", data["content_type"])
		
		// Check parsed JSON data
		responseData, ok := data["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "Hello, World!", responseData["message"])
		assert.Equal(t, "success", responseData["status"])
		
		// Check metadata
		assert.Contains(t, result.Metadata, "url")
		assert.Contains(t, result.Metadata, "response_size")
	})

	t.Run("Successful Text Request", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/text",
		}

		result := httpGet.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, 200, data["status_code"])
		assert.Equal(t, "text/plain", data["content_type"])
		assert.Equal(t, "Hello, World!", data["data"])
	})

	t.Run("HTTP Error Response", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/error",
		}

		result := httpGet.Execute(ctx, input)
		assert.True(t, result.Success) // HTTP errors are still "successful" executions
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, 500, data["status_code"])
		assert.Equal(t, "Internal Server Error", data["data"])
	})

	t.Run("Invalid URL", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": "not-a-valid-url",
		}

		result := httpGet.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "HTTP request failed")
	})

	t.Run("Network Error", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": "http://localhost:99999/nonexistent", // Port that should be closed
		}

		result := httpGet.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "HTTP request failed")
	})

	t.Run("Custom Headers", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/json",
			"headers": map[string]interface{}{
				"User-Agent":    "test-agent",
				"Authorization": "Bearer token123",
			},
		}

		result := httpGet.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 200, data["status_code"])
	})
}

func TestHTTPPostTool(t *testing.T) {
	// Create a test HTTP server
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		switch r.URL.Path {
		case "/echo":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"received": "data", "method": "POST"}`))
		case "/created":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 123, "status": "created"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	httpPost := builtin.NewHTTPPostTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "http_post", httpPost.Name())
		
		schema := httpPost.Schema()
		assert.Equal(t, "http_post", schema.Name)
		assert.Contains(t, schema.Description, "HTTP POST")
		assert.Len(t, schema.Parameters, 5)
	})

	t.Run("Successful POST with JSON Data", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/echo",
			"data": map[string]interface{}{
				"name":  "John Doe",
				"email": "john@example.com",
			},
			"headers": map[string]interface{}{
				"Authorization": "Bearer token123",
			},
		}

		result := httpPost.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, 200, data["status_code"])
		
		// Check response data
		responseData, ok := data["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "data", responseData["received"])
		assert.Equal(t, "POST", responseData["method"])
	})

	t.Run("POST with Created Response", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/created",
			"data": map[string]interface{}{
				"name": "New Item",
			},
		}

		result := httpPost.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, 201, data["status_code"])
		
		responseData, ok := data["data"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, float64(123), responseData["id"]) // JSON numbers are float64
		assert.Equal(t, "created", responseData["status"])
	})

	t.Run("POST without Data", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/echo",
		}

		result := httpPost.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 200, data["status_code"])
	})

	t.Run("Custom Content Type", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url":          testServer.URL + "/echo",
			"data":         "plain text data",
			"content_type": "text/plain",
		}

		result := httpPost.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, 200, data["status_code"])
	})

	t.Run("Invalid URL", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": "invalid-url",
		}

		result := httpPost.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "HTTP request failed")
	})
}

func TestWebScraperTool(t *testing.T) {
	// Create a test HTTP server with HTML content
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/html":
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`
				<!DOCTYPE html>
				<html>
				<head>
					<title>Test Page</title>
				</head>
				<body>
					<h1>Welcome to Test Page</h1>
					<p>This is a test paragraph with some content.</p>
					<div>Another section with text.</div>
				</body>
				</html>
			`))
		case "/nothtml":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"message": "This is not HTML"}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer testServer.Close()

	scraper := builtin.NewWebScraperTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "web_scraper", scraper.Name())
		
		schema := scraper.Schema()
		assert.Equal(t, "web_scraper", schema.Name)
		assert.Contains(t, schema.Description, "Scrapes")
		assert.Len(t, schema.Parameters, 3)
	})

	t.Run("Scrape HTML Content", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url":        testServer.URL + "/html",
			"max_length": 1000.0,
		}

		result := scraper.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		assert.Equal(t, "Test Page", data["title"])
		assert.Contains(t, data["content"], "Welcome to Test Page")
		assert.Contains(t, data["content"], "test paragraph")
		assert.Equal(t, testServer.URL+"/html", data["url"])
		
		// Check that HTML tags are stripped
		content := data["content"].(string)
		assert.NotContains(t, content, "<h1>")
		assert.NotContains(t, content, "</p>")
		
		// Check metadata
		assert.Contains(t, result.Metadata, "status_code")
		assert.Contains(t, result.Metadata, "content_type")
		assert.Contains(t, result.Metadata, "original_size")
	})

	t.Run("Content Length Limit", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url":        testServer.URL + "/html",
			"max_length": 50.0, // Very short limit
		}

		result := scraper.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		require.True(t, ok)
		
		content := data["content"].(string)
		assert.LessOrEqual(t, len(content), 53) // 50 + "..." = 53
		assert.Contains(t, content, "...")
	})

	t.Run("Non-HTML Content", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": testServer.URL + "/nothtml",
		}

		result := scraper.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "does not return HTML content")
	})

	t.Run("Invalid URL", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": "not-a-url",
		}

		result := scraper.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "HTTP request failed")
	})

	t.Run("Network Error", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"url": "http://localhost:99999/nonexistent",
		}

		result := scraper.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "HTTP request failed")
	})
}