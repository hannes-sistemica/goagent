package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	contextpkg "agent-server/internal/context"
	"agent-server/internal/llm"
	"agent-server/internal/models"
	"agent-server/internal/storage"

	"github.com/google/uuid"
)

// ChatService handles chat operations with LLM integration and tool calling support
type ChatService struct {
	repo          storage.Repository
	llmRegistry   *llm.Registry
	ctxRegistry   *contextpkg.StrategyRegistry
	toolService   *ToolService
	promptService *PromptService
	logger        *slog.Logger
}

// NewChatService creates a new chat service with tool support
func NewChatService(
	repo storage.Repository,
	llmRegistry *llm.Registry,
	ctxRegistry *contextpkg.StrategyRegistry,
	toolService *ToolService,
	promptService *PromptService,
	logger *slog.Logger,
) *ChatService {
	return &ChatService{
		repo:          repo,
		llmRegistry:   llmRegistry,
		ctxRegistry:   ctxRegistry,
		toolService:   toolService,
		promptService: promptService,
		logger:        logger,
	}
}

// ChatRequest represents a chat request
type ChatRequest struct {
	SessionID string                 `json:"session_id"`
	Message   string                 `json:"message"`
	Metadata  map[string]interface{} `json:"metadata"`
	Stream    bool                   `json:"stream"`
}

// ChatResponse represents a chat response
type ChatResponse struct {
	UserMessageID      string                 `json:"user_message_id"`
	AssistantMessageID string                 `json:"assistant_message_id"`
	Response           string                 `json:"response"`
	Metadata           map[string]interface{} `json:"metadata"`
}

// Chat processes a chat request and returns a response
func (s *ChatService) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// Get session with agent info
	session, err := s.repo.Session().GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Get LLM provider
	provider, exists := s.llmRegistry.Get(session.Agent.Provider)
	if !exists {
		return nil, fmt.Errorf("unsupported LLM provider: %s", session.Agent.Provider)
	}

	// Check if provider is available
	if !provider.IsAvailable(ctx) {
		return nil, fmt.Errorf("LLM provider %s is not available", session.Agent.Provider)
	}

	// Save user message
	userMessage := &models.Message{
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Message,
		Metadata:  models.JSON(req.Metadata),
	}

	if err := s.repo.Message().Create(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get message history for context
	messages, _, err := s.repo.Message().ListBySessionID(ctx, req.SessionID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	// Build context using strategy
	strategy, exists := s.ctxRegistry.Get(session.ContextStrategy)
	if !exists {
		return nil, fmt.Errorf("unknown context strategy: %s", session.ContextStrategy)
	}

	contextMessages, err := strategy.BuildContext(
		ctx,
		session.Agent.SystemPrompt,
		"", // No additional agent prompt for now
		messages,
		session.ContextConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	// Prepare LLM request
	llmRequest := &llm.ChatRequest{
		Model:       session.Agent.Model,
		Messages:    llm.ConvertMessages(contextMessages),
		Temperature: session.Agent.Temperature,
		MaxTokens:   session.Agent.MaxTokens,
		Stream:      req.Stream,
		Options:     session.Agent.Config,
	}

	// Call LLM provider
	llmResponse, err := provider.Chat(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM request failed: %w", err)
	}

	// Prepare metadata
	metadata := map[string]interface{}{
		"provider":       session.Agent.Provider,
		"model":          session.Agent.Model,
		"context_length": len(contextMessages),
		"strategy":       session.ContextStrategy,
	}

	// Add usage info if available
	if llmResponse.Usage != nil {
		metadata["usage"] = map[string]interface{}{
			"prompt_tokens":     llmResponse.Usage.PromptTokens,
			"completion_tokens": llmResponse.Usage.CompletionTokens,
			"total_tokens":      llmResponse.Usage.TotalTokens,
		}
	}

	// Add LLM metadata
	for k, v := range llmResponse.Metadata {
		metadata[k] = v
	}

	// Save assistant message
	assistantMessage := &models.Message{
		SessionID: req.SessionID,
		Role:      "assistant",
		Content:   llmResponse.Content,
		Metadata:  models.JSON(metadata),
	}

	if err := s.repo.Message().Create(ctx, assistantMessage); err != nil {
		return nil, fmt.Errorf("failed to save assistant message: %w", err)
	}

	s.logger.Info("Chat completed successfully",
		"session_id", req.SessionID,
		"user_message_id", userMessage.ID,
		"assistant_message_id", assistantMessage.ID,
		"provider", session.Agent.Provider,
		"model", session.Agent.Model)

	return &ChatResponse{
		UserMessageID:      userMessage.ID,
		AssistantMessageID: assistantMessage.ID,
		Response:           llmResponse.Content,
		Metadata:           metadata,
	}, nil
}

// StreamChunk represents a streaming response chunk
type StreamChunk struct {
	Content      string                 `json:"content"`
	Done         bool                   `json:"done"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	MessageID    string                 `json:"message_id,omitempty"`
}

// Stream processes a streaming chat request
func (s *ChatService) Stream(ctx context.Context, req *ChatRequest) (<-chan StreamChunk, error) {
	// Get session with agent info
	session, err := s.repo.Session().GetByID(ctx, req.SessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Get LLM provider
	provider, exists := s.llmRegistry.Get(session.Agent.Provider)
	if !exists {
		return nil, fmt.Errorf("unsupported LLM provider: %s", session.Agent.Provider)
	}

	// Check if provider is available
	if !provider.IsAvailable(ctx) {
		return nil, fmt.Errorf("LLM provider %s is not available", session.Agent.Provider)
	}

	// Save user message
	userMessage := &models.Message{
		SessionID: req.SessionID,
		Role:      "user",
		Content:   req.Message,
		Metadata:  models.JSON(req.Metadata),
	}

	if err := s.repo.Message().Create(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	// Get message history for context
	messages, _, err := s.repo.Message().ListBySessionID(ctx, req.SessionID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	// Build context using strategy
	strategy, exists := s.ctxRegistry.Get(session.ContextStrategy)
	if !exists {
		return nil, fmt.Errorf("unknown context strategy: %s", session.ContextStrategy)
	}

	contextMessages, err := strategy.BuildContext(
		ctx,
		session.Agent.SystemPrompt,
		"",
		messages,
		session.ContextConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to build context: %w", err)
	}

	// Prepare LLM request
	llmRequest := &llm.ChatRequest{
		Model:       session.Agent.Model,
		Messages:    llm.ConvertMessages(contextMessages),
		Temperature: session.Agent.Temperature,
		MaxTokens:   session.Agent.MaxTokens,
		Stream:      true,
		Options:     session.Agent.Config,
	}

	// Start streaming from LLM provider
	llmChunks, err := provider.Stream(ctx, llmRequest)
	if err != nil {
		return nil, fmt.Errorf("LLM streaming failed: %w", err)
	}

	// Create output channel
	outputChunks := make(chan StreamChunk, 10)

	// Process streaming response
	go func() {
		defer close(outputChunks)

		var fullResponse strings.Builder
		var assistantMessage *models.Message

		for chunk := range llmChunks {
			// Forward chunk to client
			outputChunk := StreamChunk{
				Content:  chunk.Content,
				Done:     chunk.Done,
				Metadata: chunk.Metadata,
			}

			select {
			case outputChunks <- outputChunk:
			case <-ctx.Done():
				return
			}

			// Accumulate response
			fullResponse.WriteString(chunk.Content)

			// Save final message when done
			if chunk.Done {
				metadata := map[string]interface{}{
					"provider":       session.Agent.Provider,
					"model":          session.Agent.Model,
					"context_length": len(contextMessages),
					"strategy":       session.ContextStrategy,
					"streamed":       true,
				}

				// Add chunk metadata
				for k, v := range chunk.Metadata {
					metadata[k] = v
				}

				assistantMessage = &models.Message{
					SessionID: req.SessionID,
					Role:      "assistant",
					Content:   fullResponse.String(),
					Metadata:  models.JSON(metadata),
				}

				if err := s.repo.Message().Create(ctx, assistantMessage); err != nil {
					s.logger.Error("Failed to save streamed assistant message", "error", err)
				} else {
					// Send final chunk with message ID
					finalChunk := StreamChunk{
						Content:   "",
						Done:      true,
						MessageID: assistantMessage.ID,
						Metadata: map[string]interface{}{
							"user_message_id": userMessage.ID,
						},
					}

					select {
					case outputChunks <- finalChunk:
					case <-ctx.Done():
						return
					}
				}

				s.logger.Info("Streaming chat completed successfully",
					"session_id", req.SessionID,
					"user_message_id", userMessage.ID,
					"assistant_message_id", assistantMessage.ID,
					"provider", session.Agent.Provider,
					"model", session.Agent.Model,
					"response_length", fullResponse.Len())

				break
			}
		}
	}()

	return outputChunks, nil
}

// ChatWithTools processes a chat request with tool calling support
func (s *ChatService) ChatWithTools(ctx context.Context, req *models.EnhancedChatRequest, sessionID string) (*models.EnhancedChatResponse, error) {
	// Get session with agent info
	session, err := s.repo.Session().GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	if session == nil {
		return nil, fmt.Errorf("session not found")
	}

	// Get LLM provider
	provider, exists := s.llmRegistry.Get(session.Agent.Provider)
	if !exists {
		return nil, fmt.Errorf("unsupported LLM provider: %s", session.Agent.Provider)
	}

	// Check if provider is available
	if !provider.IsAvailable(ctx) {
		return nil, fmt.Errorf("LLM provider %s is not available", session.Agent.Provider)
	}

	// Save user message
	userMessage := &models.Message{
		SessionID: sessionID,
		Role:      "user",
		Content:   req.Message,
		Metadata:  models.JSON(req.Metadata),
	}

	if err := s.repo.Message().Create(ctx, userMessage); err != nil {
		return nil, fmt.Errorf("failed to save user message: %w", err)
	}

	s.logger.Info("Processing chat request with tools",
		"session_id", sessionID,
		"user_message_id", userMessage.ID,
		"tools_requested", len(req.Tools),
		"tool_choice", req.ToolChoice)

	// Get available tools
	availableTools := req.Tools
	if len(availableTools) == 0 {
		// If no tools specified, use all available tools
		toolsList, err := s.toolService.ListTools(ctx)
		if err == nil {
			for _, tool := range toolsList.Tools {
				if tool.Available {
					availableTools = append(availableTools, tool.Name)
				}
			}
		}
	}

	// Process the conversation with potential tool calls
	response, err := s.processWithToolCalls(ctx, session, userMessage, availableTools, req)
	if err != nil {
		return nil, fmt.Errorf("failed to process chat with tools: %w", err)
	}

	return response, nil
}

// processWithToolCalls handles the main conversation loop with tool calling
func (s *ChatService) processWithToolCalls(
	ctx context.Context,
	session *models.ChatSession,
	userMessage *models.Message,
	availableTools []string,
	req *models.EnhancedChatRequest,
) (*models.EnhancedChatResponse, error) {
	maxIterations := 5 // Prevent infinite loops
	var allToolCalls []models.ToolCallResult
	var conversationMessages []*models.Message

	// Get initial message history
	messages, _, err := s.repo.Message().ListBySessionID(ctx, session.ID, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get message history: %w", err)
	}

	conversationMessages = messages

	for iteration := 0; iteration < maxIterations; iteration++ {
		s.logger.Debug("Tool conversation iteration",
			"iteration", iteration,
			"session_id", session.ID)

		// Build context using strategy with dynamic prompt
		strategy, exists := s.ctxRegistry.Get(session.ContextStrategy)
		if !exists {
			return nil, fmt.Errorf("unknown context strategy: %s", session.ContextStrategy)
		}

		// Generate dynamic system prompt with tool descriptions
		enhancedSystemPrompt := s.promptService.BuildSystemPrompt(ctx, session.Agent.SystemPrompt, availableTools)

		contextMessages, err := strategy.BuildContext(
			ctx,
			enhancedSystemPrompt,
			"",
			conversationMessages,
			session.ContextConfig,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to build context: %w", err)
		}

		// Get tool definitions if tools are available
		var toolDefinitions []models.ToolDefinition
		if len(availableTools) > 0 && req.ToolChoice != "none" {
			toolDefinitions, err = s.toolService.GetToolDefinitions(ctx, availableTools)
			if err != nil {
				s.logger.Error("Failed to get tool definitions", "error", err)
				// Continue without tools
			}
		}

		// Prepare LLM request
		llmRequest := &llm.ChatRequest{
			Model:       session.Agent.Model,
			Messages:    llm.ConvertMessages(contextMessages),
			Temperature: session.Agent.Temperature,
			MaxTokens:   session.Agent.MaxTokens,
			Stream:      req.Stream,
			Options:     session.Agent.Config,
		}

		// Override with request-specific parameters
		if req.Temperature != nil {
			llmRequest.Temperature = *req.Temperature
		}
		if req.MaxTokens != nil {
			llmRequest.MaxTokens = *req.MaxTokens
		}

		// Get LLM provider
		provider, exists := s.llmRegistry.Get(session.Agent.Provider)
		if !exists {
			return nil, fmt.Errorf("unsupported LLM provider: %s", session.Agent.Provider)
		}

		// Check if provider is available
		if !provider.IsAvailable(ctx) {
			return nil, fmt.Errorf("LLM provider %s is not available", session.Agent.Provider)
		}

		// Add tools to LLM request
		if len(toolDefinitions) > 0 {
			llmRequest.Options["tools"] = toolDefinitions
			llmRequest.Options["tool_choice"] = req.ToolChoice
		}

		// Call LLM provider
		llmResponse, err := provider.Chat(ctx, llmRequest)
		if err != nil {
			return nil, fmt.Errorf("LLM request failed: %w", err)
		}

		// Check if the response contains tool calls
		toolCalls, err := s.parseToolCallsFromResponse(llmResponse.Content, llmResponse.Metadata)
		if err != nil {
			s.logger.Error("Failed to parse tool calls", "error", err)
			toolCalls = nil // Continue without tool calls
		}

		// If no tool calls, this is the final response
		if len(toolCalls) == 0 {
			// Save assistant message
			assistantMessage, err := s.saveAssistantMessage(ctx, session.ID, llmResponse, len(contextMessages), session.ContextStrategy, len(toolDefinitions) > 0)
			if err != nil {
				return nil, fmt.Errorf("failed to save assistant message: %w", err)
			}

			// Prepare final response
			return &models.EnhancedChatResponse{
				UserMessageID:      userMessage.ID,
				AssistantMessageID: assistantMessage.ID,
				Response:           llmResponse.Content,
				ToolCalls:          allToolCalls,
				Metadata:           assistantMessage.Metadata,
				FinishReason:       getFinishReason(llmResponse, len(toolCalls) > 0),
			}, nil
		}

		// Execute tool calls
		s.logger.Info("Executing tool calls", "count", len(toolCalls), "session_id", session.ID)
		toolResults, err := s.toolService.ExecuteToolCalls(ctx, session.ID, toolCalls)
		if err != nil {
			return nil, fmt.Errorf("failed to execute tool calls: %w", err)
		}

		// Add tool results to the conversation
		allToolCalls = append(allToolCalls, toolResults...)

		// Save assistant message with tool calls
		assistantMessage, err := s.saveAssistantMessageWithToolCalls(ctx, session.ID, llmResponse, toolCalls, toolResults, len(contextMessages), session.ContextStrategy)
		if err != nil {
			return nil, fmt.Errorf("failed to save assistant message with tool calls: %w", err)
		}

		// Add assistant message to conversation
		conversationMessages = append(conversationMessages, assistantMessage)

		// Create tool result messages and add them to conversation
		toolMessages := s.toolService.CreateToolResultMessages(toolResults)
		for _, toolMsg := range toolMessages {
			toolMessage := &models.Message{
				SessionID: session.ID,
				Role:      "tool",
				Content:   toolMsg.Content,
				Metadata: models.JSON(map[string]interface{}{
					"tool_call_id": toolMsg.ToolCallID,
					"tool_result":  true,
				}),
			}

			if err := s.repo.Message().Create(ctx, toolMessage); err != nil {
				s.logger.Error("Failed to save tool message", "error", err)
				// Continue anyway
			} else {
				conversationMessages = append(conversationMessages, toolMessage)
			}
		}
	}

	// If we exit the loop, return the last response
	return nil, fmt.Errorf("exceeded maximum tool call iterations")
}

// parseToolCallsFromResponse parses tool calls from LLM response
func (s *ChatService) parseToolCallsFromResponse(content string, metadata map[string]interface{}) ([]models.LLMToolCall, error) {
	// Check if metadata contains tool calls from Ollama
	if toolCallsData, exists := metadata["tool_calls"]; exists {
		// Handle Ollama format: []map[string]interface{}
		if toolCallsArray, ok := toolCallsData.([]map[string]interface{}); ok {
			var toolCalls []models.LLMToolCall
			for _, tcData := range toolCallsArray {
				if function, ok := tcData["function"].(map[string]interface{}); ok {
					name, _ := function["name"].(string)
					arguments, _ := function["arguments"].(map[string]interface{})
					
					// Convert arguments to JSON string as expected by models.LLMToolCall
					argumentsJSON, err := json.Marshal(arguments)
					if err != nil {
						continue
					}
					
					toolCall := models.LLMToolCall{
						ID: uuid.New().String(),
						Function: models.LLMToolCallFunction{
							Name:      name,
							Arguments: string(argumentsJSON),
						},
					}
					toolCalls = append(toolCalls, toolCall)
				}
			}
			return toolCalls, nil
		}
		
		// Handle JSON string format
		if toolCallsJSON, ok := toolCallsData.(string); ok {
			var toolCalls []models.LLMToolCall
			if err := json.Unmarshal([]byte(toolCallsJSON), &toolCalls); err == nil {
				return toolCalls, nil
			}
		}
	}

	// Look for tool calls in content (fallback for some providers)
	if strings.Contains(content, "tool_calls") {
		// This would be provider-specific parsing
		return []models.LLMToolCall{}, nil
	}

	return []models.LLMToolCall{}, nil
}

// saveAssistantMessage saves the assistant's message to the database
func (s *ChatService) saveAssistantMessage(
	ctx context.Context,
	sessionID string,
	llmResponse *llm.ChatResponse,
	contextLength int,
	strategy string,
	toolsAvailable bool,
) (*models.Message, error) {
	metadata := map[string]interface{}{
		"provider":        llmResponse.Model, // This should be provider name
		"model":           llmResponse.Model,
		"context_length":  contextLength,
		"strategy":        strategy,
		"tools_available": toolsAvailable,
		"finish_reason":   llmResponse.FinishReason,
	}

	// Add usage info if available
	if llmResponse.Usage != nil {
		metadata["usage"] = map[string]interface{}{
			"prompt_tokens":     llmResponse.Usage.PromptTokens,
			"completion_tokens": llmResponse.Usage.CompletionTokens,
			"total_tokens":      llmResponse.Usage.TotalTokens,
		}
	}

	// Add LLM metadata
	for k, v := range llmResponse.Metadata {
		metadata[k] = v
	}

	assistantMessage := &models.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResponse.Content,
		Metadata:  models.JSON(metadata),
	}

	if err := s.repo.Message().Create(ctx, assistantMessage); err != nil {
		return nil, err
	}

	return assistantMessage, nil
}

// saveAssistantMessageWithToolCalls saves assistant message with tool call information
func (s *ChatService) saveAssistantMessageWithToolCalls(
	ctx context.Context,
	sessionID string,
	llmResponse *llm.ChatResponse,
	toolCalls []models.LLMToolCall,
	toolResults []models.ToolCallResult,
	contextLength int,
	strategy string,
) (*models.Message, error) {
	metadata := map[string]interface{}{
		"provider":       llmResponse.Model, // This should be provider name
		"model":          llmResponse.Model,
		"context_length": contextLength,
		"strategy":       strategy,
		"tool_calls":     len(toolCalls),
		"finish_reason":  "tool_calls",
	}

	// Add usage info if available
	if llmResponse.Usage != nil {
		metadata["usage"] = map[string]interface{}{
			"prompt_tokens":     llmResponse.Usage.PromptTokens,
			"completion_tokens": llmResponse.Usage.CompletionTokens,
			"total_tokens":      llmResponse.Usage.TotalTokens,
		}
	}

	// Add tool call metadata
	toolCallMetadata := make([]map[string]interface{}, len(toolCalls))
	for i, call := range toolCalls {
		callMeta := map[string]interface{}{
			"id":        call.ID,
			"tool_name": call.Function.Name,
		}
		if i < len(toolResults) {
			callMeta["success"] = toolResults[i].Success
			callMeta["duration_ms"] = toolResults[i].Duration
		}
		toolCallMetadata[i] = callMeta
	}
	metadata["tool_call_details"] = toolCallMetadata

	// Add LLM metadata
	for k, v := range llmResponse.Metadata {
		metadata[k] = v
	}

	assistantMessage := &models.Message{
		SessionID: sessionID,
		Role:      "assistant",
		Content:   llmResponse.Content,
		Metadata:  models.JSON(metadata),
	}

	if err := s.repo.Message().Create(ctx, assistantMessage); err != nil {
		return nil, err
	}

	// Save individual tool calls to the database
	for i, call := range toolCalls {
		toolCall := &models.ToolCall{
			MessageID: assistantMessage.ID,
			ToolName:  call.Function.Name,
			Arguments: models.JSON(map[string]interface{}{"raw": call.Function.Arguments}),
			Success:   false,
			Duration:  0,
		}

		// Parse arguments
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(call.Function.Arguments), &args); err == nil {
			toolCall.Arguments = models.JSON(args)
		}

		// Add results if available
		if i < len(toolResults) {
			result := toolResults[i]
			toolCall.Success = result.Success
			toolCall.Duration = result.Duration
			if result.Error != "" {
				toolCall.Error = result.Error
			}
			if result.Result != nil {
				resultJSON := models.JSON(map[string]interface{}{"data": result.Result})
				toolCall.Result = &resultJSON
			}
		}

		// Save tool call (this would require extending the repository interface)
		// For now, just log it
		s.logger.Info("Tool call saved",
			"message_id", assistantMessage.ID,
			"tool_name", call.Function.Name,
			"success", toolCall.Success)
	}

	return assistantMessage, nil
}

// getFinishReason determines the finish reason for the response
func getFinishReason(llmResponse *llm.ChatResponse, hasToolCalls bool) string {
	if hasToolCalls {
		return "tool_calls"
	}
	if llmResponse.FinishReason != "" {
		return llmResponse.FinishReason
	}
	return "stop"
}