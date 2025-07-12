package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"agent-server/internal/models"
	"agent-server/internal/storage"
	"agent-server/internal/tools"
	"agent-server/internal/tools/builtin"
)

// ToolService handles tool execution and management
type ToolService struct {
	registry   *tools.Registry
	executor   *tools.Executor
	repository storage.Repository
	logger     *slog.Logger
}

// NewToolService creates a new tool service
func NewToolService(repository storage.Repository, logger *slog.Logger) *ToolService {
	registry := tools.NewRegistry()
	executor := tools.NewExecutor(registry, 60*time.Second) // 60 second timeout

	// Register built-in tools
	if err := builtin.RegisterBuiltinTools(registry, repository.Memory()); err != nil {
		logger.Error("Failed to register built-in tools", "error", err)
	} else {
		logger.Info("Registered built-in tools", "count", registry.Count())
	}

	return &ToolService{
		registry:   registry,
		executor:   executor,
		repository: repository,
		logger:     logger,
	}
}

// GetRegistry returns the tool registry
func (ts *ToolService) GetRegistry() *tools.Registry {
	return ts.registry
}

// GetExecutor returns the tool executor
func (ts *ToolService) GetExecutor() *tools.Executor {
	return ts.executor
}

// ListTools returns information about available tools
func (ts *ToolService) ListTools(ctx context.Context) (*models.ToolsListResponse, error) {
	toolNames := ts.registry.List()
	toolInfos := make([]models.ToolInfo, 0, len(toolNames))

	for _, name := range toolNames {
		tool, exists := ts.registry.Get(name)
		if !exists {
			continue
		}

		schema := tool.Schema()
		parameters := make([]models.ToolParameterInfo, len(schema.Parameters))
		
		for i, param := range schema.Parameters {
			parameters[i] = models.ToolParameterInfo{
				Name:        param.Name,
				Type:        param.Type,
				Description: param.Description,
				Required:    param.Required,
				Default:     param.Default,
				Enum:        param.Enum,
			}
		}

		examples := make([]models.ToolExampleInfo, len(schema.Examples))
		for i, example := range schema.Examples {
			examples[i] = models.ToolExampleInfo{
				Description: example.Description,
				Input:       example.Input,
				Output:      example.Output,
			}
		}

		toolInfos = append(toolInfos, models.ToolInfo{
			Name:        schema.Name,
			Description: schema.Description,
			Parameters:  parameters,
			Available:   tool.IsAvailable(ctx),
			Category:    inferToolCategory(schema.Name),
			Examples:    examples,
		})
	}

	return &models.ToolsListResponse{
		Tools:      toolInfos,
		TotalCount: len(toolInfos),
	}, nil
}

// TestTool tests a tool with given parameters
func (ts *ToolService) TestTool(ctx context.Context, req *models.ToolTestRequest) (*models.ToolTestResponse, error) {
	timeout := 30 * time.Second
	if req.Timeout != nil {
		timeout = time.Duration(*req.Timeout) * time.Second
	}

	// Create a temporary executor with custom timeout
	testExecutor := tools.NewExecutor(ts.registry, timeout)

	// Execute the tool
	result := testExecutor.Execute(ctx, req.ToolName, "test-session", req.Arguments)

	response := &models.ToolTestResponse{
		Success:  result.Success,
		Result:   result.Data,
		Error:    result.Error,
		Duration: result.Duration.Milliseconds(),
	}

	// Add validation errors if any
	if !result.Success && result.ErrorCode == "VALIDATION_ERROR" {
		// Parse validation errors from the error message
		// This is a simplified approach - in production, you might want more structured validation
		response.Validation = []models.ValidationError{
			{
				Parameter: "unknown",
				Message:   result.Error,
			},
		}
	}

	return response, nil
}

// ExecuteToolCalls executes multiple tool calls and returns results
func (ts *ToolService) ExecuteToolCalls(ctx context.Context, sessionID string, toolCalls []models.LLMToolCall) ([]models.ToolCallResult, error) {
	results := make([]models.ToolCallResult, len(toolCalls))

	for i, toolCall := range toolCalls {
		result := ts.executeSingleToolCall(ctx, sessionID, toolCall)
		results[i] = result

		// Log the tool execution
		if err := ts.logToolExecution(ctx, sessionID, toolCall, result); err != nil {
			ts.logger.Error("Failed to log tool execution", 
				"tool_name", toolCall.Function.Name,
				"error", err)
		}
	}

	return results, nil
}

// executeSingleToolCall executes a single tool call
func (ts *ToolService) executeSingleToolCall(ctx context.Context, sessionID string, toolCall models.LLMToolCall) models.ToolCallResult {
	ts.logger.Info("Executing tool call", 
		"tool_name", toolCall.Function.Name,
		"session_id", sessionID,
		"call_id", toolCall.ID)

	// Parse arguments
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
		ts.logger.Error("Failed to parse tool arguments",
			"tool_name", toolCall.Function.Name,
			"arguments", toolCall.Function.Arguments,
			"error", err)
		
		return models.ToolCallResult{
			ID:       toolCall.ID,
			ToolName: toolCall.Function.Name,
			Success:  false,
			Error:    fmt.Sprintf("Invalid tool arguments: %v", err),
			Duration: 0,
		}
	}

	// Get agent ID from session
	agentID, err := ts.getAgentIDFromSession(ctx, sessionID)
	if err != nil {
		return models.ToolCallResult{
			ID:       toolCall.ID,
			ToolName: toolCall.Function.Name,
			Success:  false,
			Error:    fmt.Sprintf("Failed to get agent ID: %v", err),
			Duration: 0,
		}
	}

	// Execute the tool with proper context
	start := time.Now()
	result := ts.executeToolWithContext(ctx, toolCall.Function.Name, sessionID, agentID, arguments)
	duration := time.Since(start)

	return models.ToolCallResult{
		ID:       toolCall.ID,
		ToolName: toolCall.Function.Name,
		Success:  result.Success,
		Result:   result.Data,
		Error:    result.Error,
		Duration: duration.Milliseconds(),
	}
}

// logToolExecution logs tool execution to the database
func (ts *ToolService) logToolExecution(ctx context.Context, sessionID string, toolCall models.LLMToolCall, result models.ToolCallResult) error {
	// Parse arguments
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
		arguments = map[string]interface{}{"raw": toolCall.Function.Arguments}
	}

	// Create tool execution log
	log := &models.ToolExecutionLog{
		SessionID:  sessionID,
		ToolCallID: toolCall.ID,
		ToolName:   toolCall.Function.Name,
		Arguments:  models.JSON(arguments),
		Success:    result.Success,
		Error:      result.Error,
		Duration:   result.Duration,
		ExecutedAt: time.Now(),
	}

	if result.Result != nil {
		resultJSON := models.JSON(map[string]interface{}{"data": result.Result})
		log.Result = &resultJSON
	}

	// Save to database (this would require extending the repository interface)
	// For now, just log it
	ts.logger.Info("Tool execution logged",
		"tool_name", toolCall.Function.Name,
		"success", result.Success,
		"duration_ms", result.Duration)

	return nil
}

// GetToolDefinitions returns tool definitions for LLM providers
func (ts *ToolService) GetToolDefinitions(ctx context.Context, toolNames []string) ([]models.ToolDefinition, error) {
	var definitions []models.ToolDefinition

	// If no tool names specified, get all available tools
	if len(toolNames) == 0 {
		toolNames = ts.registry.List()
	}

	for _, name := range toolNames {
		tool, exists := ts.registry.Get(name)
		if !exists {
			continue
		}

		// Check if tool is available
		if !tool.IsAvailable(ctx) {
			continue
		}

		schema := tool.Schema()
		
		// Convert tool schema to LLM tool definition format
		definition := models.ToolDefinition{
			Type: "function",
			Function: models.ToolFunctionDefinition{
				Name:        schema.Name,
				Description: schema.Description,
				Parameters:  convertSchemaToJSONSchema(schema),
			},
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

// ParseToolCallsFromLLMResponse parses tool calls from LLM response
func (ts *ToolService) ParseToolCallsFromLLMResponse(response string) ([]models.LLMToolCall, error) {
	// This is a simplified implementation
	// In a real implementation, you would parse the actual LLM response format
	// which varies by provider (OpenAI, Anthropic, etc.)
	
	var toolCalls []models.LLMToolCall
	
	// For now, return empty slice
	// This would be implemented based on the specific LLM provider's response format
	
	return toolCalls, nil
}

// CreateToolResultMessages creates tool result messages for LLM context
func (ts *ToolService) CreateToolResultMessages(results []models.ToolCallResult) []models.ToolMessage {
	messages := make([]models.ToolMessage, len(results))

	for i, result := range results {
		content := map[string]interface{}{
			"success": result.Success,
		}

		if result.Success {
			content["result"] = result.Result
		} else {
			content["error"] = result.Error
		}

		contentJSON, _ := json.Marshal(content)

		messages[i] = models.ToolMessage{
			Role:       "tool",
			Content:    string(contentJSON),
			ToolCallID: result.ID,
		}
	}

	return messages
}

// inferToolCategory infers the category of a tool based on its name
func inferToolCategory(toolName string) string {
	name := strings.ToLower(toolName)
	
	if strings.Contains(name, "http") || strings.Contains(name, "web") || strings.Contains(name, "scraper") {
		return "web"
	}
	if strings.Contains(name, "mcp") || strings.Contains(name, "openmcp") {
		return "proxy"
	}
	if strings.Contains(name, "calculator") || strings.Contains(name, "math") {
		return "math"
	}
	if strings.Contains(name, "text") || strings.Contains(name, "json") {
		return "text"
	}
	
	return "utility"
}

// convertSchemaToJSONSchema converts internal tool schema to JSON Schema format
func convertSchemaToJSONSchema(schema tools.Schema) map[string]interface{} {
	jsonSchema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
		"required":   []string{},
	}

	properties := jsonSchema["properties"].(map[string]interface{})
	var required []string

	for _, param := range schema.Parameters {
		prop := map[string]interface{}{
			"type":        param.Type,
			"description": param.Description,
		}

		if param.Default != nil {
			prop["default"] = param.Default
		}

		if len(param.Enum) > 0 {
			prop["enum"] = param.Enum
		}

		if param.Minimum != nil {
			prop["minimum"] = *param.Minimum
		}

		if param.Maximum != nil {
			prop["maximum"] = *param.Maximum
		}

		if param.Pattern != "" {
			prop["pattern"] = param.Pattern
		}

		properties[param.Name] = prop

		if param.Required {
			required = append(required, param.Name)
		}
	}

	if len(required) > 0 {
		jsonSchema["required"] = required
	}

	return jsonSchema
}

// getAgentIDFromSession retrieves the agent ID associated with a session
func (ts *ToolService) getAgentIDFromSession(ctx context.Context, sessionID string) (string, error) {
	session, err := ts.repository.Session().GetByID(ctx, sessionID)
	if err != nil {
		return "", fmt.Errorf("failed to get session: %w", err)
	}
	if session == nil {
		return "", fmt.Errorf("session not found")
	}
	return session.AgentID, nil
}

// executeToolWithContext executes a tool with proper execution context
func (ts *ToolService) executeToolWithContext(ctx context.Context, toolName, sessionID, agentID string, arguments map[string]interface{}) *tools.Result {
	// Get the tool from registry
	tool, exists := ts.registry.Get(toolName)
	if !exists {
		return tools.ErrorResult("TOOL_NOT_FOUND", fmt.Sprintf("Tool '%s' not found", toolName))
	}

	// Check if tool is available
	if !tool.IsAvailable(ctx) {
		return tools.ErrorResult("TOOL_UNAVAILABLE", fmt.Sprintf("Tool '%s' is not available", toolName))
	}

	// Create execution context
	execCtx := tools.ExecutionContext{
		Context:   ctx,
		SessionID: sessionID,
		AgentID:   agentID,
		RequestID: "req-" + sessionID + "-" + toolName,
		Timeout:   60 * time.Second,
		Metadata:  make(map[string]interface{}),
	}

	// Execute the tool
	return tool.Execute(execCtx, arguments)
}