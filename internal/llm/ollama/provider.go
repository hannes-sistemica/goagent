package ollama

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"agent-server/internal/llm"
	"agent-server/internal/models"

	"github.com/sirupsen/logrus"
)

// Provider implements the LLM provider interface for Ollama
type Provider struct {
	baseURL    string
	httpClient *http.Client
}

// NewProvider creates a new Ollama provider
func NewProvider(baseURL string) *Provider {
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	return &Provider{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// Name returns the provider name
func (p *Provider) Name() string {
	return "ollama"
}

// ollamaChatRequest represents the request format for Ollama chat API
type ollamaChatRequest struct {
	Model     string                `json:"model"`
	Messages  []llm.ChatMessage     `json:"messages"`
	Stream    bool                  `json:"stream"`
	Options   map[string]interface{} `json:"options,omitempty"`
	Tools     []interface{}         `json:"tools,omitempty"`
	ToolChoice interface{}          `json:"tool_choice,omitempty"`
}

// ollamaChatResponse represents the response format from Ollama chat API
type ollamaChatResponse struct {
	Model     string            `json:"model"`
	Message   ollamaMessage     `json:"message"`
	Done      bool              `json:"done"`
	CreatedAt time.Time         `json:"created_at"`
	Usage     *ollamaUsage      `json:"usage,omitempty"`
}

// ollamaMessage represents a message in Ollama format
type ollamaMessage struct {
	Role      string          `json:"role"`
	Content   string          `json:"content"`
	ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
}

// ollamaToolCall represents a tool call in Ollama format
type ollamaToolCall struct {
	Function ollamaToolFunction `json:"function"`
}

// ollamaToolFunction represents a tool function call
type ollamaToolFunction struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ollamaUsage represents usage statistics from Ollama
type ollamaUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ollamaModel represents a model in Ollama's model list
type ollamaModel struct {
	Name       string    `json:"name"`
	ModifiedAt time.Time `json:"modified_at"`
	Size       int64     `json:"size"`
}

// ollamaModelsResponse represents the response from Ollama's models API
type ollamaModelsResponse struct {
	Models []ollamaModel `json:"models"`
}

// Chat sends a chat request to Ollama
func (p *Provider) Chat(ctx context.Context, req *llm.ChatRequest) (*llm.ChatResponse, error) {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   false,
		Options:  p.buildOptions(req),
	}

	// Add tools if present
	if tools, ok := req.Options["tools"]; ok {
		// Convert []models.ToolDefinition to []interface{}
		if toolDefs, ok := tools.([]models.ToolDefinition); ok {
			toolsInterface := make([]interface{}, len(toolDefs))
			for i, td := range toolDefs {
				toolsInterface[i] = td
			}
			ollamaReq.Tools = toolsInterface
		}
	}
	if toolChoice, ok := req.Options["tool_choice"]; ok {
		ollamaReq.ToolChoice = toolChoice
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	response := &llm.ChatResponse{
		Content: ollamaResp.Message.Content,
		Model:   ollamaResp.Model,
		Metadata: map[string]interface{}{
			"created_at": ollamaResp.CreatedAt,
		},
	}

	// Add tool calls to metadata if present
	if len(ollamaResp.Message.ToolCalls) > 0 {
		toolCalls := make([]map[string]interface{}, len(ollamaResp.Message.ToolCalls))
		for i, tc := range ollamaResp.Message.ToolCalls {
			toolCalls[i] = map[string]interface{}{
				"function": map[string]interface{}{
					"name":      tc.Function.Name,
					"arguments": tc.Function.Arguments,
				},
			}
		}
		response.Metadata["tool_calls"] = toolCalls
	}

	if ollamaResp.Usage != nil {
		response.Usage = &llm.Usage{
			PromptTokens:     ollamaResp.Usage.PromptTokens,
			CompletionTokens: ollamaResp.Usage.CompletionTokens,
			TotalTokens:      ollamaResp.Usage.TotalTokens,
		}
	}

	return response, nil
}

// Stream sends a streaming chat request to Ollama
func (p *Provider) Stream(ctx context.Context, req *llm.ChatRequest) (<-chan llm.StreamChunk, error) {
	ollamaReq := ollamaChatRequest{
		Model:    req.Model,
		Messages: req.Messages,
		Stream:   true,
		Options:  p.buildOptions(req),
	}

	// Add tools if present
	if tools, ok := req.Options["tools"]; ok {
		// Convert []models.ToolDefinition to []interface{}
		if toolDefs, ok := tools.([]models.ToolDefinition); ok {
			toolsInterface := make([]interface{}, len(toolDefs))
			for i, td := range toolDefs {
				toolsInterface[i] = td
			}
			ollamaReq.Tools = toolsInterface
		}
	}
	if toolChoice, ok := req.Options["tool_choice"]; ok {
		ollamaReq.ToolChoice = toolChoice
	}

	reqBody, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(body))
	}

	chunks := make(chan llm.StreamChunk, 10)

	go func() {
		defer resp.Body.Close()
		defer close(chunks)

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var ollamaResp ollamaChatResponse
			if err := json.Unmarshal([]byte(line), &ollamaResp); err != nil {
				logrus.WithError(err).Error("Failed to parse streaming response")
				continue
			}

			chunk := llm.StreamChunk{
				Content: ollamaResp.Message.Content,
				Done:    ollamaResp.Done,
				Model:   ollamaResp.Model,
				Metadata: map[string]interface{}{
					"created_at": ollamaResp.CreatedAt,
				},
			}

			select {
			case chunks <- chunk:
			case <-ctx.Done():
				return
			}

			if ollamaResp.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			logrus.WithError(err).Error("Error reading streaming response")
		}
	}()

	return chunks, nil
}

// Models returns the list of available models from Ollama
func (p *Provider) Models(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API error %d: %s", resp.StatusCode, string(body))
	}

	var modelsResp ollamaModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	models := make([]string, len(modelsResp.Models))
	for i, model := range modelsResp.Models {
		models[i] = model.Name
	}

	return models, nil
}

// ValidateConfig validates Ollama-specific configuration
func (p *Provider) ValidateConfig(config map[string]interface{}) error {
	// Ollama doesn't require API keys, so just validate structure
	if config == nil {
		return nil
	}

	// Validate known options
	if temp, ok := config["temperature"]; ok {
		if _, ok := temp.(float64); !ok {
			if _, ok := temp.(float32); !ok {
				return fmt.Errorf("temperature must be a number")
			}
		}
	}

	if tokens, ok := config["num_predict"]; ok {
		if _, ok := tokens.(int); !ok {
			if _, ok := tokens.(float64); !ok {
				return fmt.Errorf("num_predict must be an integer")
			}
		}
	}

	return nil
}

// IsAvailable checks if Ollama is available
func (p *Provider) IsAvailable(ctx context.Context) bool {
	httpReq, err := http.NewRequestWithContext(ctx, "GET", p.baseURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// buildOptions builds Ollama-specific options from the request
func (p *Provider) buildOptions(req *llm.ChatRequest) map[string]interface{} {
	options := make(map[string]interface{})

	if req.Temperature > 0 {
		options["temperature"] = req.Temperature
	}

	if req.MaxTokens > 0 {
		options["num_predict"] = req.MaxTokens
	}

	// Add any additional options from the request
	for k, v := range req.Options {
		options[k] = v
	}

	return options
}