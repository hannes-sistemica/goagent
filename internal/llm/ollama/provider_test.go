package ollama

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"agent-server/internal/llm"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProvider_Name(t *testing.T) {
	provider := NewProvider("")
	assert.Equal(t, "ollama", provider.Name())
}

func TestNewProvider(t *testing.T) {
	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "default URL",
			baseURL:  "",
			expected: "http://localhost:11434",
		},
		{
			name:     "custom URL",
			baseURL:  "http://custom:8080",
			expected: "http://custom:8080",
		},
		{
			name:     "URL with trailing slash",
			baseURL:  "http://custom:8080/",
			expected: "http://custom:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewProvider(tt.baseURL)
			assert.Equal(t, tt.expected, provider.baseURL)
		})
	}
}

func TestProvider_ValidateConfig(t *testing.T) {
	provider := NewProvider("")

	tests := []struct {
		name        string
		config      map[string]interface{}
		expectError bool
	}{
		{
			name:        "nil config",
			config:      nil,
			expectError: false,
		},
		{
			name:        "empty config",
			config:      map[string]interface{}{},
			expectError: false,
		},
		{
			name: "valid config",
			config: map[string]interface{}{
				"temperature":  0.7,
				"num_predict": 100,
			},
			expectError: false,
		},
		{
			name: "temperature as float32",
			config: map[string]interface{}{
				"temperature": float32(0.5),
			},
			expectError: false,
		},
		{
			name: "num_predict as float64",
			config: map[string]interface{}{
				"num_predict": float64(150),
			},
			expectError: false,
		},
		{
			name: "invalid temperature type",
			config: map[string]interface{}{
				"temperature": "invalid",
			},
			expectError: true,
		},
		{
			name: "invalid num_predict type",
			config: map[string]interface{}{
				"num_predict": "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := provider.ValidateConfig(tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProvider_BuildOptions(t *testing.T) {
	provider := NewProvider("")

	tests := []struct {
		name     string
		request  *llm.ChatRequest
		expected map[string]interface{}
	}{
		{
			name: "basic request",
			request: &llm.ChatRequest{
				Temperature: 0.7,
				MaxTokens:   100,
			},
			expected: map[string]interface{}{
				"temperature":  float32(0.7),
				"num_predict": 100,
			},
		},
		{
			name: "request with additional options",
			request: &llm.ChatRequest{
				Temperature: 0.5,
				Options: map[string]interface{}{
					"top_k": 40,
					"top_p": 0.9,
				},
			},
			expected: map[string]interface{}{
				"temperature": float32(0.5),
				"top_k":       40,
				"top_p":       0.9,
			},
		},
		{
			name: "empty request",
			request: &llm.ChatRequest{},
			expected: map[string]interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := provider.buildOptions(tt.request)
			assert.Equal(t, tt.expected, options)
		})
	}
}

func TestProvider_Chat(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Mock response
		response := `{
			"model": "llama2",
			"message": {
				"role": "assistant",
				"content": "Hello! How can I help you today?"
			},
			"done": true,
			"created_at": "2023-01-01T00:00:00Z",
			"usage": {
				"prompt_tokens": 10,
				"completion_tokens": 8,
				"total_tokens": 18
			}
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	provider := NewProvider(server.URL)

	request := &llm.ChatRequest{
		Model: "llama2",
		Messages: []llm.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Temperature: 0.7,
		MaxTokens:   100,
	}

	ctx := context.Background()
	response, err := provider.Chat(ctx, request)

	require.NoError(t, err)
	assert.Equal(t, "Hello! How can I help you today?", response.Content)
	assert.Equal(t, "llama2", response.Model)
	assert.NotNil(t, response.Usage)
	assert.Equal(t, 10, response.Usage.PromptTokens)
	assert.Equal(t, 8, response.Usage.CompletionTokens)
	assert.Equal(t, 18, response.Usage.TotalTokens)
}

func TestProvider_Chat_Error(t *testing.T) {
	// Create a mock server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	provider := NewProvider(server.URL)

	request := &llm.ChatRequest{
		Model: "llama2",
		Messages: []llm.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	ctx := context.Background()
	_, err := provider.Chat(ctx, request)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ollama API error 500")
}

func TestProvider_Models(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, "/api/tags", r.URL.Path)

		// Mock response
		response := `{
			"models": [
				{
					"name": "llama2",
					"modified_at": "2023-01-01T00:00:00Z",
					"size": 1000000
				},
				{
					"name": "codellama",
					"modified_at": "2023-01-02T00:00:00Z",
					"size": 2000000
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
	}))
	defer server.Close()

	provider := NewProvider(server.URL)

	ctx := context.Background()
	models, err := provider.Models(ctx)

	require.NoError(t, err)
	assert.Len(t, models, 2)
	assert.Contains(t, models, "llama2")
	assert.Contains(t, models, "codellama")
}

func TestProvider_IsAvailable(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *httptest.Server
		expected bool
	}{
		{
			name: "available",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(`{"models": []}`))
				}))
			},
			expected: true,
		},
		{
			name: "unavailable",
			setup: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setup()
			defer server.Close()

			provider := NewProvider(server.URL)
			ctx := context.Background()

			available := provider.IsAvailable(ctx)
			assert.Equal(t, tt.expected, available)
		})
	}
}

func TestProvider_Stream(t *testing.T) {
	// Create a mock server that returns streaming response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/chat", r.URL.Path)

		// Mock streaming response
		responses := []string{
			`{"model":"llama2","message":{"role":"assistant","content":"Hello"},"done":false,"created_at":"2023-01-01T00:00:00Z"}`,
			`{"model":"llama2","message":{"role":"assistant","content":" there"},"done":false,"created_at":"2023-01-01T00:00:00Z"}`,
			`{"model":"llama2","message":{"role":"assistant","content":"!"},"done":true,"created_at":"2023-01-01T00:00:00Z"}`,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		for _, response := range responses {
			w.Write([]byte(response + "\n"))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}
	}))
	defer server.Close()

	provider := NewProvider(server.URL)

	request := &llm.ChatRequest{
		Model: "llama2",
		Messages: []llm.ChatMessage{
			{Role: "user", Content: "Hello"},
		},
		Stream: true,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	chunks, err := provider.Stream(ctx, request)
	require.NoError(t, err)

	var receivedChunks []llm.StreamChunk
	for chunk := range chunks {
		receivedChunks = append(receivedChunks, chunk)
		if chunk.Done {
			break
		}
	}

	assert.Len(t, receivedChunks, 3)
	assert.Equal(t, "Hello", receivedChunks[0].Content)
	assert.Equal(t, " there", receivedChunks[1].Content)
	assert.Equal(t, "!", receivedChunks[2].Content)
	assert.True(t, receivedChunks[2].Done)
}