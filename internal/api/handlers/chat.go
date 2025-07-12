package handlers

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"agent-server/internal/models"
	"agent-server/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ChatHandler handles chat-related requests with tool calling support
type ChatHandler struct {
	chatService *services.ChatService
	toolService *services.ToolService
	validator   *validator.Validate
	logger      *slog.Logger
}

// NewChatHandler creates a new chat handler
func NewChatHandler(
	chatService *services.ChatService,
	toolService *services.ToolService,
	logger *slog.Logger,
) *ChatHandler {
	return &ChatHandler{
		chatService: chatService,
		toolService: toolService,
		validator:   validator.New(),
		logger:      logger,
	}
}

// Chat handles synchronous chat requests
func (h *ChatHandler) Chat(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req services.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set session ID
	req.SessionID = sessionID

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Process chat request
	response, err := h.chatService.Chat(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Chat request failed", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Chat request failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Stream handles streaming chat requests
func (h *ChatHandler) Stream(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req services.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Set session ID and streaming
	req.SessionID = sessionID
	req.Stream = true

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Set headers for Server-Sent Events
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Access-Control-Allow-Origin", "*")

	// Start streaming
	chunks, err := h.chatService.Stream(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Streaming chat failed", "session_id", sessionID, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming failed", "details": err.Error()})
		return
	}

	// Stream chunks to client
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Streaming not supported"})
		return
	}

	for chunk := range chunks {
		// Write chunk as SSE
		if chunk.Content != "" || chunk.Done {
			fmt.Fprintf(c.Writer, "data: %s\n\n", h.formatSSEData(chunk))
			flusher.Flush()
		}

		if chunk.Done {
			break
		}

		// Check if client disconnected
		select {
		case <-c.Request.Context().Done():
			return
		default:
		}
	}
}

// formatSSEData formats a chunk as JSON for Server-Sent Events
func (h *ChatHandler) formatSSEData(chunk services.StreamChunk) string {
	data := map[string]interface{}{
		"content": chunk.Content,
		"done":    chunk.Done,
	}

	if chunk.MessageID != "" {
		data["message_id"] = chunk.MessageID
	}

	if chunk.Metadata != nil {
		data["metadata"] = chunk.Metadata
	}

	// Convert to JSON (ignoring errors for simplicity)
	jsonData, _ := json.Marshal(data)
	return string(jsonData)
}

// ChatWithTools handles chat requests with explicit tool calling
func (h *ChatHandler) ChatWithTools(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req models.EnhancedChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Processing chat request with tools",
		"session_id", sessionID,
		"message_length", len(req.Message),
		"tools_count", len(req.Tools),
		"tool_choice", req.ToolChoice)

	// Process chat request with tools
	response, err := h.chatService.ChatWithTools(c.Request.Context(), &req, sessionID)
	if err != nil {
		h.logger.Error("Chat with tools request failed",
			"session_id", sessionID,
			"error", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Chat request failed",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Chat with tools completed",
		"session_id", sessionID,
		"user_message_id", response.UserMessageID,
		"assistant_message_id", response.AssistantMessageID,
		"tool_calls_count", len(response.ToolCalls),
		"finish_reason", response.FinishReason)

	c.JSON(http.StatusOK, response)
}

// ChatWithAutoTools handles chat requests with automatic tool selection
func (h *ChatHandler) ChatWithAutoTools(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var basicReq services.ChatRequest
	if err := c.ShouldBindJSON(&basicReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Convert to enhanced request with auto tool selection
	req := models.EnhancedChatRequest{
		Message:    basicReq.Message,
		Tools:      []string{}, // Empty means all available tools
		ToolChoice: "auto",     // Let the LLM decide
		Metadata:   basicReq.Metadata,
		Stream:     basicReq.Stream,
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Validation failed",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Processing auto-tools chat request",
		"session_id", sessionID,
		"message_length", len(req.Message))

	// Process chat request with automatic tool selection
	response, err := h.chatService.ChatWithTools(c.Request.Context(), &req, sessionID)
	if err != nil {
		h.logger.Error("Auto-tools chat request failed",
			"session_id", sessionID,
			"error", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Chat request failed",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Auto-tools chat completed",
		"session_id", sessionID,
		"tool_calls_count", len(response.ToolCalls),
		"finish_reason", response.FinishReason)

	c.JSON(http.StatusOK, response)
}

// ListAvailableTools returns tools available for a session
func (h *ChatHandler) ListAvailableTools(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Get available tools
	tools, err := h.toolService.ListTools(ctx)
	if err != nil {
		h.logger.Error("Failed to list tools for session",
			"session_id", sessionID,
			"error", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to list tools",
			"details": err.Error(),
		})
		return
	}

	// Filter to only available tools
	availableTools := make([]models.ToolInfo, 0, len(tools.Tools))
	for _, tool := range tools.Tools {
		if tool.Available {
			availableTools = append(availableTools, tool)
		}
	}

	response := &models.ToolsListResponse{
		Tools:      availableTools,
		TotalCount: len(availableTools),
	}

	c.JSON(http.StatusOK, response)
}

// GetToolSchema returns the schema for a specific tool
func (h *ChatHandler) GetToolSchema(c *gin.Context) {
	sessionID := c.Param("id")
	toolName := c.Param("tool_name")
	
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}
	
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tool name is required"})
		return
	}

	ctx := c.Request.Context()

	// Get tool definitions
	definitions, err := h.toolService.GetToolDefinitions(ctx, []string{toolName})
	if err != nil {
		h.logger.Error("Failed to get tool definition",
			"session_id", sessionID,
			"tool_name", toolName,
			"error", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get tool schema",
			"details": err.Error(),
		})
		return
	}

	if len(definitions) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"error":     "Tool not found or not available",
			"tool_name": toolName,
		})
		return
	}

	c.JSON(http.StatusOK, definitions[0])
}

// TestToolForSession tests a tool within a session context
func (h *ChatHandler) TestToolForSession(c *gin.Context) {
	sessionID := c.Param("id")
	toolName := c.Param("tool_name")
	
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}
	
	if toolName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tool name is required"})
		return
	}

	var req models.ToolTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Override tool name from URL
	req.ToolName = toolName

	h.logger.Info("Testing tool for session",
		"session_id", sessionID,
		"tool_name", toolName)

	// Test the tool
	response, err := h.toolService.TestTool(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Tool test failed",
			"session_id", sessionID,
			"tool_name", toolName,
			"error", err)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to test tool",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Tool test completed",
		"session_id", sessionID,
		"tool_name", toolName,
		"success", response.Success,
		"duration_ms", response.Duration)

	c.JSON(http.StatusOK, response)
}

// GetToolCallHistory returns the history of tool calls for a session
func (h *ChatHandler) GetToolCallHistory(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 20
	
	if pageParam := c.Query("page"); pageParam != "" {
		if p := parseInt(pageParam, 1); p > 0 {
			page = p
		}
	}
	
	if sizeParam := c.Query("page_size"); sizeParam != "" {
		if s := parseInt(sizeParam, 20); s > 0 && s <= 100 {
			pageSize = s
		}
	}

	// This would require implementing tool call history retrieval
	// For now, return placeholder data
	h.logger.Info("Retrieving tool call history",
		"session_id", sessionID,
		"page", page,
		"page_size", pageSize)

	// Placeholder response
	response := map[string]interface{}{
		"tool_calls":  []interface{}{},
		"total_count": 0,
		"page":        page,
		"page_size":   pageSize,
		"has_more":    false,
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to parse integer with default
func parseInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	
	// Simple integer parsing
	result := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return defaultValue
		}
	}
	return result
}