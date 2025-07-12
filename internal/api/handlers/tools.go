package handlers

import (
	"net/http"
	"strconv"

	"agent-server/internal/models"
	"agent-server/internal/services"

	"github.com/gin-gonic/gin"
)

// ToolsHandler handles tool-related HTTP requests
type ToolsHandler struct {
	toolService *services.ToolService
}

// NewToolsHandler creates a new tools handler
func NewToolsHandler(toolService *services.ToolService) *ToolsHandler {
	return &ToolsHandler{
		toolService: toolService,
	}
}

// ListTools returns a list of available tools
// @Summary List available tools
// @Description Get a list of all available tools with their schemas and availability status
// @Tags tools
// @Accept json
// @Produce json
// @Success 200 {object} models.ToolsListResponse
// @Failure 500 {object} map[string]string
// @Router /tools [get]
func (h *ToolsHandler) ListTools(c *gin.Context) {
	ctx := c.Request.Context()

	tools, err := h.toolService.ListTools(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to list tools",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, tools)
}

// GetTool returns information about a specific tool
// @Summary Get tool information
// @Description Get detailed information about a specific tool including its schema
// @Tags tools
// @Accept json
// @Produce json
// @Param tool_name path string true "Tool name"
// @Success 200 {object} models.ToolInfo
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tools/{tool_name} [get]
func (h *ToolsHandler) GetTool(c *gin.Context) {
	ctx := c.Request.Context()
	toolName := c.Param("tool_name")

	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool name is required",
		})
		return
	}

	// Get tool from registry
	tool, exists := h.toolService.GetRegistry().Get(toolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tool not found",
			"tool_name": toolName,
		})
		return
	}

	// Convert to ToolInfo
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

	toolInfo := models.ToolInfo{
		Name:        schema.Name,
		Description: schema.Description,
		Parameters:  parameters,
		Available:   tool.IsAvailable(ctx),
		Examples:    examples,
	}

	c.JSON(http.StatusOK, toolInfo)
}

// TestTool tests a tool with provided parameters
// @Summary Test a tool
// @Description Test a tool execution with provided parameters
// @Tags tools
// @Accept json
// @Produce json
// @Param tool_name path string true "Tool name"
// @Param request body models.ToolTestRequest true "Tool test request"
// @Success 200 {object} models.ToolTestResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tools/{tool_name}/test [post]
func (h *ToolsHandler) TestTool(c *gin.Context) {
	ctx := c.Request.Context()
	toolName := c.Param("tool_name")

	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool name is required",
		})
		return
	}

	// Check if tool exists
	_, exists := h.toolService.GetRegistry().Get(toolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tool not found",
			"tool_name": toolName,
		})
		return
	}

	var req models.ToolTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Override tool name from URL
	req.ToolName = toolName

	// Test the tool
	response, err := h.toolService.TestTool(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to test tool",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetToolSchemas returns JSON schemas for tools (for LLM integration)
// @Summary Get tool schemas
// @Description Get JSON schemas for tools in LLM-compatible format
// @Tags tools
// @Accept json
// @Produce json
// @Param tools query string false "Comma-separated list of tool names (empty for all)"
// @Success 200 {array} models.ToolDefinition
// @Failure 500 {object} map[string]string
// @Router /tools/schemas [get]
func (h *ToolsHandler) GetToolSchemas(c *gin.Context) {
	ctx := c.Request.Context()
	
	// Parse tool names from query parameter
	var toolNames []string
	if toolsParam := c.Query("tools"); toolsParam != "" {
		toolNames = parseCommaSeparatedString(toolsParam)
	}

	definitions, err := h.toolService.GetToolDefinitions(ctx, toolNames)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get tool schemas",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, definitions)
}

// ExecuteTool executes a tool directly (for testing/debugging)
// @Summary Execute a tool
// @Description Execute a tool with provided parameters (for testing/debugging)
// @Tags tools
// @Accept json
// @Produce json
// @Param tool_name path string true "Tool name"
// @Param request body map[string]interface{} true "Tool execution parameters"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /tools/{tool_name}/execute [post]
func (h *ToolsHandler) ExecuteTool(c *gin.Context) {
	ctx := c.Request.Context()
	toolName := c.Param("tool_name")

	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Tool name is required",
		})
		return
	}

	// Check if tool exists
	_, exists := h.toolService.GetRegistry().Get(toolName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Tool not found",
			"tool_name": toolName,
		})
		return
	}

	var parameters map[string]interface{}
	if err := c.ShouldBindJSON(&parameters); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Execute the tool
	result := h.toolService.GetExecutor().Execute(ctx, toolName, "direct-execution", parameters)

	// Return the raw result
	response := map[string]interface{}{
		"success":     result.Success,
		"tool_name":   toolName,
		"duration_ms": result.Duration.Milliseconds(),
	}

	if result.Success {
		response["result"] = result.Data
	} else {
		response["error"] = result.Error
		response["error_code"] = result.ErrorCode
	}

	if result.Metadata != nil {
		response["metadata"] = result.Metadata
	}

	status := http.StatusOK
	if !result.Success {
		status = http.StatusBadRequest
	}

	c.JSON(status, response)
}

// GetToolUsageStats returns usage statistics for tools
// @Summary Get tool usage statistics
// @Description Get usage statistics for all tools or a specific tool
// @Tags tools
// @Accept json
// @Produce json
// @Param tool_name query string false "Specific tool name (empty for all tools)"
// @Param days query int false "Number of days to look back (default: 7)"
// @Success 200 {array} models.ToolUsageStats
// @Failure 500 {object} map[string]string
// @Router /tools/stats [get]
func (h *ToolsHandler) GetToolUsageStats(c *gin.Context) {
	// This would require implementing usage tracking in the database
	// For now, return empty stats
	
	toolName := c.Query("tool_name")
	daysParam := c.Query("days")
	
	days := 7
	if daysParam != "" {
		if d, err := strconv.Atoi(daysParam); err == nil && d > 0 {
			days = d
		}
	}
	
	// Use the days parameter for future implementation
	_ = days

	// Placeholder implementation
	stats := []models.ToolUsageStats{}
	
	if toolName != "" {
		// Return stats for specific tool
		if _, exists := h.toolService.GetRegistry().Get(toolName); exists {
			stats = append(stats, models.ToolUsageStats{
				ToolName:        toolName,
				TotalCalls:      0,
				SuccessfulCalls: 0,
				FailedCalls:     0,
				AvgDuration:     0,
			})
		}
	} else {
		// Return stats for all tools
		toolNames := h.toolService.GetRegistry().List()
		for _, name := range toolNames {
			stats = append(stats, models.ToolUsageStats{
				ToolName:        name,
				TotalCalls:      0,
				SuccessfulCalls: 0,
				FailedCalls:     0,
				AvgDuration:     0,
			})
		}
	}

	c.JSON(http.StatusOK, stats)
}

// Helper function to parse comma-separated strings
func parseCommaSeparatedString(s string) []string {
	if s == "" {
		return nil
	}
	
	var result []string
	for _, item := range splitAndTrim(s, ",") {
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// Helper function to split and trim strings
func splitAndTrim(s, sep string) []string {
	var result []string
	for _, item := range splitString(s, sep) {
		trimmed := trimWhitespace(item)
		result = append(result, trimmed)
	}
	return result
}

// Simple string split function
func splitString(s, sep string) []string {
	if s == "" {
		return []string{}
	}
	
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

// Simple whitespace trim function
func trimWhitespace(s string) string {
	start := 0
	end := len(s)
	
	// Trim leading whitespace
	for start < len(s) && isWhitespace(s[start]) {
		start++
	}
	
	// Trim trailing whitespace
	for end > start && isWhitespace(s[end-1]) {
		end--
	}
	
	return s[start:end]
}

// Check if character is whitespace
func isWhitespace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}