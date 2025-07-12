package llm

import (
	"context"

	"agent-server/internal/models"
)

// ChatMessage represents a message in the LLM chat format
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a request to an LLM provider
type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	Temperature float32       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
	Options     map[string]interface{} `json:"options,omitempty"`
}

// ChatResponse represents a response from an LLM provider
type ChatResponse struct {
	Content   string                 `json:"content"`
	Model     string                 `json:"model"`
	Usage     *Usage                 `json:"usage,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	FinishReason string              `json:"finish_reason,omitempty"`
}

// Usage represents token usage information
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Content      string                 `json:"content"`
	Done         bool                   `json:"done"`
	Model        string                 `json:"model,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	FinishReason string                 `json:"finish_reason,omitempty"`
}

// Provider defines the interface for LLM providers
type Provider interface {
	// Name returns the provider name
	Name() string
	
	// Chat sends a chat request and returns a response
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	
	// Stream sends a chat request and returns a stream of chunks
	Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error)
	
	// Models returns the list of available models
	Models(ctx context.Context) ([]string, error)
	
	// ValidateConfig validates provider-specific configuration
	ValidateConfig(config map[string]interface{}) error
	
	// IsAvailable checks if the provider is available
	IsAvailable(ctx context.Context) bool
}

// Registry manages LLM providers
type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(provider Provider) {
	r.providers[provider.Name()] = provider
}

// Get retrieves a provider by name
func (r *Registry) Get(name string) (Provider, bool) {
	provider, exists := r.providers[name]
	return provider, exists
}

// List returns all available provider names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// ConvertMessages converts internal messages to LLM format
func ConvertMessages(messages []*models.Message) []ChatMessage {
	result := make([]ChatMessage, len(messages))
	for i, msg := range messages {
		result[i] = ChatMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return result
}