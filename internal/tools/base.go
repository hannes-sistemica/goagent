package tools

import (
	"context"
	"time"
)

// BaseTool provides a base implementation for tools
type BaseTool struct {
	name        string
	schema      Schema
	executeFunc func(ExecutionContext, map[string]interface{}) *Result
	available   func(context.Context) bool
}

// NewBaseTool creates a new base tool
func NewBaseTool(name string, schema Schema, executeFunc func(ExecutionContext, map[string]interface{}) *Result) *BaseTool {
	return &BaseTool{
		name:        name,
		schema:      schema,
		executeFunc: executeFunc,
		available: func(ctx context.Context) bool {
			return true // Default to always available
		},
	}
}

// Name returns the tool's name
func (bt *BaseTool) Name() string {
	return bt.name
}

// Schema returns the tool's schema
func (bt *BaseTool) Schema() Schema {
	return bt.schema
}

// Validate validates the input using the schema
func (bt *BaseTool) Validate(input map[string]interface{}) error {
	return ValidateInput(bt.schema, input)
}

// Execute executes the tool with error handling
func (bt *BaseTool) Execute(ctx ExecutionContext, input map[string]interface{}) *Result {
	// Sanitize input
	sanitized, err := SanitizeInput(bt.schema, input)
	if err != nil {
		return &Result{
			Success:   false,
			Error:     err.Error(),
			ErrorCode: "VALIDATION_ERROR",
			Duration:  0,
		}
	}
	
	// Check for context cancellation
	select {
	case <-ctx.Context.Done():
		return &Result{
			Success:   false,
			Error:     "execution cancelled",
			ErrorCode: "CANCELLED",
			Duration:  0,
		}
	default:
	}
	
	// Execute with timeout and error recovery
	start := time.Now()
	
	// Create a done channel for the execution
	done := make(chan *Result, 1)
	
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- &Result{
					Success:   false,
					Error:     "tool execution panicked",
					ErrorCode: "PANIC",
					Duration:  time.Since(start),
					Metadata: map[string]interface{}{
						"panic": r,
					},
				}
			}
		}()
		
		result := bt.executeFunc(ctx, sanitized)
		if result == nil {
			result = &Result{
				Success:   false,
				Error:     "tool returned nil result",
				ErrorCode: "NIL_RESULT",
				Duration:  time.Since(start),
			}
		}
		done <- result
	}()
	
	// Wait for completion or timeout
	select {
	case result := <-done:
		return result
	case <-ctx.Context.Done():
		return &Result{
			Success:   false,
			Error:     "execution timeout",
			ErrorCode: "TIMEOUT",
			Duration:  time.Since(start),
		}
	}
}

// IsAvailable checks if the tool is available
func (bt *BaseTool) IsAvailable(ctx context.Context) bool {
	return bt.available(ctx)
}

// SetAvailabilityCheck sets a custom availability check function
func (bt *BaseTool) SetAvailabilityCheck(available func(context.Context) bool) {
	bt.available = available
}

// HTTPBaseTool provides a base for HTTP-based tools
type HTTPBaseTool struct {
	*BaseTool
	baseURL    string
	timeout    time.Duration
	headers    map[string]string
	retryCount int
}

// NewHTTPBaseTool creates a new HTTP-based tool
func NewHTTPBaseTool(name string, schema Schema, baseURL string, timeout time.Duration) *HTTPBaseTool {
	return &HTTPBaseTool{
		BaseTool:   NewBaseTool(name, schema, nil),
		baseURL:    baseURL,
		timeout:    timeout,
		headers:    make(map[string]string),
		retryCount: 3,
	}
}

// SetHeaders sets default headers for HTTP requests
func (ht *HTTPBaseTool) SetHeaders(headers map[string]string) {
	ht.headers = headers
}

// SetRetryCount sets the number of retry attempts
func (ht *HTTPBaseTool) SetRetryCount(count int) {
	ht.retryCount = count
}

// GetBaseURL returns the base URL
func (ht *HTTPBaseTool) GetBaseURL() string {
	return ht.baseURL
}

// GetTimeout returns the timeout duration
func (ht *HTTPBaseTool) GetTimeout() time.Duration {
	return ht.timeout
}

// GetHeaders returns the default headers
func (ht *HTTPBaseTool) GetHeaders() map[string]string {
	return ht.headers
}

// GetRetryCount returns the retry count
func (ht *HTTPBaseTool) GetRetryCount() int {
	return ht.retryCount
}

// SuccessResult creates a successful result
func SuccessResult(data interface{}, metadata ...map[string]interface{}) *Result {
	result := &Result{
		Success: true,
		Data:    data,
	}
	
	if len(metadata) > 0 {
		result.Metadata = metadata[0]
	}
	
	return result
}

// ErrorResult creates an error result
func ErrorResult(errorCode, message string, metadata ...map[string]interface{}) *Result {
	result := &Result{
		Success:   false,
		Error:     message,
		ErrorCode: errorCode,
	}
	
	if len(metadata) > 0 {
		result.Metadata = metadata[0]
	}
	
	return result
}

// ValidationErrorResult creates a validation error result
func ValidationErrorResult(err error) *Result {
	return &Result{
		Success:   false,
		Error:     err.Error(),
		ErrorCode: "VALIDATION_ERROR",
	}
}