package tools

import (
	"errors"
	"fmt"
)

// Common tool errors
var (
	ErrNilTool            = errors.New("tool cannot be nil")
	ErrEmptyToolName      = errors.New("tool name cannot be empty")
	ErrToolAlreadyExists  = errors.New("tool already exists")
	ErrToolNotFound       = errors.New("tool not found")
	ErrInvalidParameter   = errors.New("invalid parameter")
	ErrMissingParameter   = errors.New("missing required parameter")
	ErrParameterType      = errors.New("parameter type mismatch")
	ErrExecutionTimeout   = errors.New("tool execution timeout")
	ErrExecutionFailed    = errors.New("tool execution failed")
	ErrNotAvailable       = errors.New("tool not available")
	ErrInvalidInput       = errors.New("invalid input")
)

// ValidationError represents a parameter validation error
type ValidationError struct {
	Parameter string
	Message   string
	Value     interface{}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error for parameter '%s': %s", e.Parameter, e.Message)
}

// NewValidationError creates a new validation error
func NewValidationError(parameter, message string, value interface{}) *ValidationError {
	return &ValidationError{
		Parameter: parameter,
		Message:   message,
		Value:     value,
	}
}

// ExecutionError represents a tool execution error
type ExecutionError struct {
	ToolName  string
	Code      string
	Message   string
	Cause     error
}

func (e *ExecutionError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s' execution failed [%s]: %s (caused by: %v)", 
			e.ToolName, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s' execution failed [%s]: %s", 
		e.ToolName, e.Code, e.Message)
}

func (e *ExecutionError) Unwrap() error {
	return e.Cause
}

// NewExecutionError creates a new execution error
func NewExecutionError(toolName, code, message string, cause error) *ExecutionError {
	return &ExecutionError{
		ToolName: toolName,
		Code:     code,
		Message:  message,
		Cause:    cause,
	}
}

// IsValidationError checks if an error is a validation error
func IsValidationError(err error) bool {
	_, ok := err.(*ValidationError)
	return ok
}

// IsExecutionError checks if an error is an execution error
func IsExecutionError(err error) bool {
	_, ok := err.(*ExecutionError)
	return ok
}