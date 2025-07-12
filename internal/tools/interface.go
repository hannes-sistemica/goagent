package tools

import (
	"context"
	"encoding/json"
	"time"
)

// Parameter represents a tool parameter definition
type Parameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`        // string, number, boolean, object, array
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
	Enum        []string    `json:"enum,omitempty"`        // for string parameters
	Minimum     *float64    `json:"minimum,omitempty"`     // for number parameters
	Maximum     *float64    `json:"maximum,omitempty"`     // for number parameters
	Pattern     string      `json:"pattern,omitempty"`     // regex for string parameters
}

// Schema defines the tool's input/output schema
type Schema struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []Parameter `json:"parameters"`
	Examples    []Example   `json:"examples,omitempty"`
}

// Example represents a tool usage example
type Example struct {
	Description string                 `json:"description"`
	Input       map[string]interface{} `json:"input"`
	Output      interface{}            `json:"output"`
}

// ExecutionContext provides context for tool execution
type ExecutionContext struct {
	Context     context.Context
	SessionID   string
	AgentID     string                 // Agent executing the tool
	UserID      string                 // future use
	RequestID   string
	Timeout     time.Duration
	Metadata    map[string]interface{}
}

// Result represents the result of tool execution
type Result struct {
	Success   bool                   `json:"success"`
	Data      interface{}            `json:"data,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ErrorCode string                 `json:"error_code,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Duration  time.Duration          `json:"duration"`
}

// CallInfo represents information about a tool call
type CallInfo struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	CallID    string                 `json:"call_id,omitempty"`
}

// Tool defines the interface for all tools
type Tool interface {
	// Name returns the tool's unique name
	Name() string
	
	// Schema returns the tool's schema definition
	Schema() Schema
	
	// Validate validates the input parameters
	Validate(input map[string]interface{}) error
	
	// Execute runs the tool with the given input
	Execute(ctx ExecutionContext, input map[string]interface{}) *Result
	
	// IsAvailable checks if the tool is available for execution
	IsAvailable(ctx context.Context) bool
}

// Registry manages tool registration and discovery
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	if tool == nil {
		return ErrNilTool
	}
	
	name := tool.Name()
	if name == "" {
		return ErrEmptyToolName
	}
	
	if _, exists := r.tools[name]; exists {
		return ErrToolAlreadyExists
	}
	
	r.tools[name] = tool
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns all registered tool names
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// GetSchemas returns schemas for all tools or specific tools
func (r *Registry) GetSchemas(toolNames ...string) []Schema {
	var schemas []Schema
	
	if len(toolNames) == 0 {
		// Return all schemas
		for _, tool := range r.tools {
			schemas = append(schemas, tool.Schema())
		}
	} else {
		// Return specific schemas
		for _, name := range toolNames {
			if tool, exists := r.tools[name]; exists {
				schemas = append(schemas, tool.Schema())
			}
		}
	}
	
	return schemas
}

// Remove removes a tool from the registry
func (r *Registry) Remove(name string) bool {
	if _, exists := r.tools[name]; exists {
		delete(r.tools, name)
		return true
	}
	return false
}

// Clear removes all tools from the registry
func (r *Registry) Clear() {
	r.tools = make(map[string]Tool)
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	return len(r.tools)
}

// Executor provides high-level tool execution capabilities
type Executor struct {
	registry *Registry
	timeout  time.Duration
}

// NewExecutor creates a new tool executor
func NewExecutor(registry *Registry, timeout time.Duration) *Executor {
	return &Executor{
		registry: registry,
		timeout:  timeout,
	}
}

// Execute executes a tool with the given parameters
func (e *Executor) Execute(ctx context.Context, toolName string, sessionID string, input map[string]interface{}) *Result {
	// Get the tool
	tool, exists := e.registry.Get(toolName)
	if !exists {
		return &Result{
			Success:   false,
			Error:     "tool not found",
			ErrorCode: "TOOL_NOT_FOUND",
			Duration:  0,
		}
	}
	
	// Check availability
	if !tool.IsAvailable(ctx) {
		return &Result{
			Success:   false,
			Error:     "tool not available",
			ErrorCode: "TOOL_UNAVAILABLE",
			Duration:  0,
		}
	}
	
	// Validate input
	if err := tool.Validate(input); err != nil {
		return &Result{
			Success:   false,
			Error:     err.Error(),
			ErrorCode: "VALIDATION_ERROR",
			Duration:  0,
		}
	}
	
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, e.timeout)
	defer cancel()
	
	executionContext := ExecutionContext{
		Context:   execCtx,
		SessionID: sessionID,
		AgentID:   "default-agent", // TODO: Get from session lookup
		RequestID: generateRequestID(),
		Timeout:   e.timeout,
		Metadata:  make(map[string]interface{}),
	}
	
	// Execute the tool
	start := time.Now()
	result := tool.Execute(executionContext, input)
	result.Duration = time.Since(start)
	
	return result
}

// ExecuteMultiple executes multiple tools concurrently
func (e *Executor) ExecuteMultiple(ctx context.Context, sessionID string, calls []CallInfo) map[string]*Result {
	results := make(map[string]*Result)
	resultChan := make(chan struct {
		callID string
		result *Result
	}, len(calls))
	
	// Execute tools concurrently
	for _, call := range calls {
		go func(call CallInfo) {
			result := e.Execute(ctx, call.ToolName, sessionID, call.Arguments)
			resultChan <- struct {
				callID string
				result *Result
			}{call.CallID, result}
		}(call)
	}
	
	// Collect results
	for i := 0; i < len(calls); i++ {
		res := <-resultChan
		results[res.callID] = res.result
	}
	
	return results
}

// Helper function to generate unique request IDs
func generateRequestID() string {
	// Simple timestamp-based ID for now
	return time.Now().Format("20060102150405") + "-" + time.Now().Format("000")
}

// MarshalJSON provides custom JSON marshaling for Result
func (r *Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	return json.Marshal(&struct {
		DurationMs int64 `json:"duration_ms"`
		*Alias
	}{
		DurationMs: r.Duration.Milliseconds(),
		Alias:      (*Alias)(r),
	})
}