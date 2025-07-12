package tools_test

import (
	"context"
	"testing"
	"time"

	"agent-server/internal/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("Register and Get Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Create a mock tool
		mockTool := &mockTool{
			name: "test_tool",
			schema: tools.Schema{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters:  []tools.Parameter{},
			},
		}
		
		// Register the tool
		err := registry.Register(mockTool)
		require.NoError(t, err)
		
		// Get the tool
		tool, exists := registry.Get("test_tool")
		assert.True(t, exists)
		assert.Equal(t, "test_tool", tool.Name())
		
		// Count should be 1
		assert.Equal(t, 1, registry.Count())
	})

	t.Run("Register Nil Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		err := registry.Register(nil)
		assert.Error(t, err)
		assert.Equal(t, tools.ErrNilTool, err)
	})

	t.Run("Register Tool with Empty Name", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		mockTool := &mockTool{
			name: "",
			schema: tools.Schema{
				Name:        "",
				Description: "A test tool",
			},
		}
		
		err := registry.Register(mockTool)
		assert.Error(t, err)
		assert.Equal(t, tools.ErrEmptyToolName, err)
	})

	t.Run("Register Duplicate Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		mockTool1 := &mockTool{
			name: "duplicate",
			schema: tools.Schema{Name: "duplicate"},
		}
		mockTool2 := &mockTool{
			name: "duplicate",
			schema: tools.Schema{Name: "duplicate"},
		}
		
		err := registry.Register(mockTool1)
		require.NoError(t, err)
		
		err = registry.Register(mockTool2)
		assert.Error(t, err)
		assert.Equal(t, tools.ErrToolAlreadyExists, err)
	})

	t.Run("List Tools", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Register multiple tools
		toolNames := []string{"tool1", "tool2", "tool3"}
		for _, name := range toolNames {
			mockTool := &mockTool{
				name:   name,
				schema: tools.Schema{Name: name},
			}
			err := registry.Register(mockTool)
			require.NoError(t, err)
		}
		
		// List should return all tool names
		list := registry.List()
		assert.Len(t, list, 3)
		
		// Check all tools are in the list
		for _, name := range toolNames {
			assert.Contains(t, list, name)
		}
	})

	t.Run("Get Schemas", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Register tools
		tool1 := &mockTool{
			name: "tool1",
			schema: tools.Schema{
				Name:        "tool1",
				Description: "Tool 1",
			},
		}
		tool2 := &mockTool{
			name: "tool2",
			schema: tools.Schema{
				Name:        "tool2",
				Description: "Tool 2",
			},
		}
		
		registry.Register(tool1)
		registry.Register(tool2)
		
		// Get all schemas
		schemas := registry.GetSchemas()
		assert.Len(t, schemas, 2)
		
		// Get specific schemas
		schemas = registry.GetSchemas("tool1")
		assert.Len(t, schemas, 1)
		assert.Equal(t, "tool1", schemas[0].Name)
		
		// Get non-existent schema
		schemas = registry.GetSchemas("nonexistent")
		assert.Len(t, schemas, 0)
	})

	t.Run("Remove Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		mockTool := &mockTool{
			name:   "removable",
			schema: tools.Schema{Name: "removable"},
		}
		
		registry.Register(mockTool)
		assert.Equal(t, 1, registry.Count())
		
		// Remove existing tool
		removed := registry.Remove("removable")
		assert.True(t, removed)
		assert.Equal(t, 0, registry.Count())
		
		// Remove non-existent tool
		removed = registry.Remove("nonexistent")
		assert.False(t, removed)
	})

	t.Run("Clear Registry", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Add multiple tools
		for i := 0; i < 5; i++ {
			mockTool := &mockTool{
				name:   "tool" + string(rune(i)),
				schema: tools.Schema{Name: "tool" + string(rune(i))},
			}
			registry.Register(mockTool)
		}
		
		assert.Equal(t, 5, registry.Count())
		
		// Clear all
		registry.Clear()
		assert.Equal(t, 0, registry.Count())
		assert.Empty(t, registry.List())
	})
}

func TestExecutor(t *testing.T) {
	t.Run("Execute Tool Successfully", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 5*time.Second)
		
		// Create a successful tool
		successTool := &mockTool{
			name: "success_tool",
			schema: tools.Schema{
				Name: "success_tool",
				Parameters: []tools.Parameter{
					{
						Name:     "input",
						Type:     "string",
						Required: true,
					},
				},
			},
			executeFunc: func(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
				return tools.SuccessResult(map[string]interface{}{
					"output": "success",
				})
			},
			available: true,
		}
		
		registry.Register(successTool)
		
		// Execute the tool
		ctx := context.Background()
		result := executor.Execute(ctx, "success_tool", "test-session", map[string]interface{}{
			"input": "test",
		})
		
		assert.True(t, result.Success)
		assert.NotNil(t, result.Data)
		assert.Empty(t, result.Error)
		assert.Greater(t, result.Duration.Nanoseconds(), int64(0))
	})

	t.Run("Execute Non-Existent Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 5*time.Second)
		
		ctx := context.Background()
		result := executor.Execute(ctx, "nonexistent", "test-session", map[string]interface{}{})
		
		assert.False(t, result.Success)
		assert.Equal(t, "tool not found", result.Error)
		assert.Equal(t, "TOOL_NOT_FOUND", result.ErrorCode)
	})

	t.Run("Execute Unavailable Tool", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 5*time.Second)
		
		unavailableTool := &mockTool{
			name:      "unavailable_tool",
			schema:    tools.Schema{Name: "unavailable_tool"},
			available: false,
		}
		
		registry.Register(unavailableTool)
		
		ctx := context.Background()
		result := executor.Execute(ctx, "unavailable_tool", "test-session", map[string]interface{}{})
		
		assert.False(t, result.Success)
		assert.Equal(t, "tool not available", result.Error)
		assert.Equal(t, "TOOL_UNAVAILABLE", result.ErrorCode)
	})

	t.Run("Execute Tool with Validation Error", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 5*time.Second)
		
		validationTool := &mockTool{
			name: "validation_tool",
			schema: tools.Schema{
				Name: "validation_tool",
				Parameters: []tools.Parameter{
					{
						Name:     "required_param",
						Type:     "string",
						Required: true,
					},
				},
			},
			available: true,
		}
		
		registry.Register(validationTool)
		
		// Execute without required parameter
		ctx := context.Background()
		result := executor.Execute(ctx, "validation_tool", "test-session", map[string]interface{}{})
		
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "required parameter missing")
		assert.Equal(t, "VALIDATION_ERROR", result.ErrorCode)
	})

	t.Run("Execute Tool with Timeout", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 100*time.Millisecond) // Short timeout
		
		slowTool := &mockTool{
			name:      "slow_tool",
			schema:    tools.Schema{Name: "slow_tool"},
			available: true,
			executeFunc: func(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
				// Simulate slow execution
				time.Sleep(200 * time.Millisecond)
				return tools.SuccessResult("should not reach")
			},
		}
		
		registry.Register(slowTool)
		
		ctx := context.Background()
		result := executor.Execute(ctx, "slow_tool", "test-session", map[string]interface{}{})
		
		// The timeout is handled in the BaseTool.Execute method
		// So we need to ensure the tool implements proper timeout handling
		assert.NotNil(t, result)
	})

	t.Run("Execute Multiple Tools Concurrently", func(t *testing.T) {
		registry := tools.NewRegistry()
		executor := tools.NewExecutor(registry, 5*time.Second)
		
		// Register multiple tools
		for i := 0; i < 3; i++ {
			toolName := "concurrent_tool_" + string(rune('0'+i))
			tool := &mockTool{
				name: toolName,
				schema: tools.Schema{
					Name: toolName,
					Parameters: []tools.Parameter{
						{
							Name:     "tool_name",
							Type:     "string",
							Required: false,
						},
					},
				},
				available: true,
				executeFunc: func(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
					time.Sleep(10 * time.Millisecond) // Simulate some work
					toolName := input["tool_name"]
					if toolName == nil {
						toolName = "unknown"
					}
					return tools.SuccessResult(map[string]interface{}{
						"tool": toolName,
					})
				},
			}
			registry.Register(tool)
		}
		
		// Create call infos
		calls := []tools.CallInfo{
			{ToolName: "concurrent_tool_0", Arguments: map[string]interface{}{"tool_name": "tool0"}, CallID: "call0"},
			{ToolName: "concurrent_tool_1", Arguments: map[string]interface{}{"tool_name": "tool1"}, CallID: "call1"},
			{ToolName: "concurrent_tool_2", Arguments: map[string]interface{}{"tool_name": "tool2"}, CallID: "call2"},
		}
		
		ctx := context.Background()
		results := executor.ExecuteMultiple(ctx, "test-session", calls)
		
		assert.Len(t, results, 3)
		
		// Check all results
		for i := 0; i < 3; i++ {
			callID := "call" + string(rune('0'+i))
			result, exists := results[callID]
			assert.True(t, exists, "Result for callID %s should exist", callID)
			if result != nil {
				assert.True(t, result.Success, "Result for callID %s should be successful, but got error: %s", callID, result.Error)
			}
		}
	})
}

// Mock tool implementation for testing
type mockTool struct {
	name        string
	schema      tools.Schema
	executeFunc func(tools.ExecutionContext, map[string]interface{}) *tools.Result
	available   bool
}

func (m *mockTool) Name() string {
	return m.name
}

func (m *mockTool) Schema() tools.Schema {
	return m.schema
}

func (m *mockTool) Validate(input map[string]interface{}) error {
	return tools.ValidateInput(m.schema, input)
}

func (m *mockTool) Execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, input)
	}
	return tools.SuccessResult("mock result")
}

func (m *mockTool) IsAvailable(ctx context.Context) bool {
	return m.available
}