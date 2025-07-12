package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ToolCall represents a tool call made by an AI agent
type ToolCall struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	MessageID string    `json:"message_id" gorm:"not null"`
	ToolName  string    `json:"tool_name" gorm:"not null"`
	Arguments JSON      `json:"arguments" gorm:"type:json"`
	Result    *JSON     `json:"result,omitempty" gorm:"type:json"`
	Success   bool      `json:"success" gorm:"default:false"`
	Error     string    `json:"error,omitempty"`
	Duration  int64     `json:"duration_ms" gorm:"default:0"` // Duration in milliseconds
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relationships
	Message Message `json:"message,omitempty" gorm:"foreignKey:MessageID"`
}

// BeforeCreate hook to generate UUID
func (tc *ToolCall) BeforeCreate(tx *gorm.DB) error {
	if tc.ID == "" {
		tc.ID = uuid.New().String()
	}
	return nil
}

// ToolCallRequest represents a request to call a tool
type ToolCallRequest struct {
	ToolName  string                 `json:"tool_name" validate:"required"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ToolCallResult represents the result of a tool call
type ToolCallResult struct {
	ID       string                 `json:"id"`
	ToolName string                 `json:"tool_name"`
	Success  bool                   `json:"success"`
	Result   interface{}            `json:"result,omitempty"`
	Error    string                 `json:"error,omitempty"`
	Duration int64                  `json:"duration_ms"`
}

// EnhancedChatRequest extends ChatRequest with tool calling capabilities
type EnhancedChatRequest struct {
	Message     string                 `json:"message" validate:"required"`
	Tools       []string               `json:"tools,omitempty"`       // Available tool names
	ToolChoice  string                 `json:"tool_choice,omitempty"` // "auto", "none", or specific tool name
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Stream      bool                   `json:"stream,omitempty"`
	MaxTokens   *int                   `json:"max_tokens,omitempty"`
	Temperature *float32               `json:"temperature,omitempty"`
}

// EnhancedChatResponse extends ChatResponse with tool calling information
type EnhancedChatResponse struct {
	UserMessageID      string           `json:"user_message_id"`
	AssistantMessageID string           `json:"assistant_message_id"`
	Response           string           `json:"response"`
	ToolCalls          []ToolCallResult `json:"tool_calls,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
	FinishReason       string           `json:"finish_reason,omitempty"` // "stop", "length", "tool_calls"
}

// ToolDefinition represents a tool schema for LLM providers
type ToolDefinition struct {
	Type     string                 `json:"type"` // Always "function" for now
	Function ToolFunctionDefinition `json:"function"`
}

// ToolFunctionDefinition represents the function definition for a tool
type ToolFunctionDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"` // JSON Schema
}

// LLMToolCall represents a tool call from an LLM response
type LLMToolCall struct {
	ID       string                 `json:"id,omitempty"`
	Type     string                 `json:"type"` // "function"
	Function LLMToolCallFunction    `json:"function"`
}

// LLMToolCallFunction represents the function call details
type LLMToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolMessage represents a tool result message for LLM context
type ToolMessage struct {
	Role       string `json:"role"`       // "tool"
	Content    string `json:"content"`    // JSON string of result
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// Agent configuration extension for tools
type AgentToolConfig struct {
	EnabledTools []string               `json:"enabled_tools,omitempty"`
	ToolChoice   string                 `json:"tool_choice,omitempty"` // "auto", "none", "required"
	ToolConfig   map[string]interface{} `json:"tool_config,omitempty"`
}

// Session configuration extension for tools
type SessionToolConfig struct {
	EnabledTools     []string               `json:"enabled_tools,omitempty"`
	ToolChoice       string                 `json:"tool_choice,omitempty"`
	MaxToolCalls     *int                   `json:"max_tool_calls,omitempty"`
	ToolTimeout      *int                   `json:"tool_timeout_seconds,omitempty"`
	ParallelToolCalls bool                  `json:"parallel_tool_calls,omitempty"`
}

// ToolExecutionLog represents a log entry for tool execution
type ToolExecutionLog struct {
	ID          string    `json:"id" gorm:"primaryKey"`
	SessionID   string    `json:"session_id" gorm:"not null"`
	ToolCallID  string    `json:"tool_call_id" gorm:"not null"`
	ToolName    string    `json:"tool_name" gorm:"not null"`
	Arguments   JSON      `json:"arguments" gorm:"type:json"`
	Result      *JSON     `json:"result,omitempty" gorm:"type:json"`
	Success     bool      `json:"success"`
	Error       string    `json:"error,omitempty"`
	Duration    int64     `json:"duration_ms"`
	ExecutedAt  time.Time `json:"executed_at"`
	
	// Relationships
	Session  ChatSession `json:"session,omitempty" gorm:"foreignKey:SessionID"`
	ToolCall ToolCall    `json:"tool_call,omitempty" gorm:"foreignKey:ToolCallID"`
}

// BeforeCreate hook to generate UUID
func (tel *ToolExecutionLog) BeforeCreate(tx *gorm.DB) error {
	if tel.ID == "" {
		tel.ID = uuid.New().String()
	}
	return nil
}

// ToolUsageStats represents usage statistics for tools
type ToolUsageStats struct {
	ToolName        string    `json:"tool_name"`
	TotalCalls      int64     `json:"total_calls"`
	SuccessfulCalls int64     `json:"successful_calls"`
	FailedCalls     int64     `json:"failed_calls"`
	AvgDuration     float64   `json:"avg_duration_ms"`
	LastUsed        time.Time `json:"last_used"`
}

// ToolsListResponse represents the response for listing available tools
type ToolsListResponse struct {
	Tools      []ToolInfo `json:"tools"`
	TotalCount int        `json:"total_count"`
}

// ToolInfo represents basic information about a tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []ToolParameterInfo    `json:"parameters"`
	Available   bool                   `json:"available"`
	Category    string                 `json:"category,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Examples    []ToolExampleInfo      `json:"examples,omitempty"`
}

// ToolParameterInfo represents information about a tool parameter
type ToolParameterInfo struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
}

// ToolExampleInfo represents an example of tool usage
type ToolExampleInfo struct {
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
}

// ToolTestRequest represents a request to test a tool
type ToolTestRequest struct {
	ToolName  string                 `json:"tool_name" validate:"required"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
	Timeout   *int                   `json:"timeout_seconds,omitempty"`
}

// ToolTestResponse represents the response from testing a tool
type ToolTestResponse struct {
	Success    bool                   `json:"success"`
	Result     interface{}            `json:"result,omitempty"`
	Error      string                 `json:"error,omitempty"`
	Duration   int64                  `json:"duration_ms"`
	Validation []ValidationError      `json:"validation_errors,omitempty"`
}

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string `json:"parameter"`
	Message   string `json:"message"`
	Value     string `json:"value,omitempty"`
}