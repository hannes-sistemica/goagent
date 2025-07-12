package services_test

import (
	"context"
	"log/slog"
	"testing"

	"agent-server/internal/models"
	"agent-server/internal/services"
	"agent-server/internal/storage/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolService_Integration(t *testing.T) {
	// Use in-memory SQLite for testing
	repo, err := sqlite.NewRepository(":memory:")
	require.NoError(t, err)
	defer repo.Close()

	// Create logger
	logger := slog.Default()
	
	t.Run("NewToolService", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		assert.NotNil(t, service)
	})

	t.Run("GetRegistry", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		registry := service.GetRegistry()
		
		assert.NotNil(t, registry)
		assert.Greater(t, registry.Count(), 0) // Should have built-in tools
	})

	t.Run("ListTools", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		ctx := context.Background()
		
		response, err := service.ListTools(ctx)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.NotEmpty(t, response.Tools)
		
		// Check for expected built-in tools
		toolNames := make([]string, len(response.Tools))
		for i, tool := range response.Tools {
			toolNames[i] = tool.Name
		}
		
		expectedTools := []string{
			"calculator", "text_processor", "json_processor",
			"http_get", "http_post", "web_scraper",
			"mcp_proxy", "openmcp_proxy",
		}
		
		for _, expected := range expectedTools {
			assert.Contains(t, toolNames, expected)
		}
	})

	t.Run("TestTool", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		ctx := context.Background()
		
		// Test valid tool with valid arguments
		request := &models.ToolTestRequest{
			ToolName: "calculator",
			Arguments: map[string]interface{}{
				"expression": "sqrt(16)",
			},
		}
		
		response, err := service.TestTool(ctx, request)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.True(t, response.Success)
		assert.NotNil(t, response.Result)
		
		// Test with invalid tool
		invalidRequest := &models.ToolTestRequest{
			ToolName:  "nonexistent",
			Arguments: map[string]interface{}{},
		}
		
		response, err = service.TestTool(ctx, invalidRequest)
		require.NoError(t, err)
		assert.NotNil(t, response)
		assert.False(t, response.Success)
		assert.Contains(t, response.Error, "tool not found")
	})

	t.Run("GetToolDefinitions", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		ctx := context.Background()
		
		// Get specific tool definitions
		toolNames := []string{"calculator", "text_processor"}
		definitions, err := service.GetToolDefinitions(ctx, toolNames)
		require.NoError(t, err)
		assert.Len(t, definitions, 2)
		
		// Check definition structure
		for _, def := range definitions {
			assert.Equal(t, "function", def.Type)
			assert.NotEmpty(t, def.Function.Name)
			assert.NotEmpty(t, def.Function.Description)
			assert.NotNil(t, def.Function.Parameters)
		}
		
		// Get all tool definitions
		allDefinitions, err := service.GetToolDefinitions(ctx, nil)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allDefinitions), 8) // At least 8 built-in tools
	})

	t.Run("ExecuteToolCalls", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		ctx := context.Background()
		
		// Create a test agent and session
		agent := &models.Agent{
			Name:     "Test Agent",
			Provider: "test",
			Model:    "test-model",
			SystemPrompt: "You are a helpful assistant",
		}
		err := repo.Agent().Create(ctx, agent)
		require.NoError(t, err)
		
		session := &models.ChatSession{
			AgentID: agent.ID,
			ContextStrategy: "last_n",
		}
		err = repo.Session().Create(ctx, session)
		require.NoError(t, err)
		
		// Create tool calls
		toolCalls := []models.LLMToolCall{
			{
				ID:   "call-1",
				Type: "function",
				Function: models.LLMToolCallFunction{
					Name:      "calculator",
					Arguments: `{"expression": "2 + 3"}`,
				},
			},
		}
		
		results, err := service.ExecuteToolCalls(ctx, session.ID, toolCalls)
		require.NoError(t, err)
		assert.Len(t, results, 1)
		
		result := results[0]
		assert.Equal(t, "call-1", result.ID)
		assert.Equal(t, "calculator", result.ToolName)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Result)
	})

	t.Run("GetExecutor", func(t *testing.T) {
		service := services.NewToolService(repo, logger)
		executor := service.GetExecutor()
		
		assert.NotNil(t, executor)
	})
}