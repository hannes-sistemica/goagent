package builtin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"agent-server/internal/models"
	"agent-server/internal/storage"
	"agent-server/internal/tools"
)

// MemoryTool provides memory storage and recall capabilities for AI agents
type MemoryTool struct {
	*tools.BaseTool
	memoryRepo storage.MemoryRepository
}

// NewMemoryTool creates a new memory tool
func NewMemoryTool(memoryRepo storage.MemoryRepository) *MemoryTool {
	schema := tools.Schema{
		Name:        "memory",
		Description: "Store and recall memories for adaptive behavior and personalized interactions",
		Parameters: []tools.Parameter{
			{
				Name:        "action",
				Type:        "string",
				Description: "Action to perform: store, recall, search, update, delete, stats",
				Required:    true,
				Enum:        []string{"store", "recall", "search", "update", "delete", "stats"},
			},
			{
				Name:        "topic",
				Type:        "string",
				Description: "Memory topic or category (required for store, recall, update)",
				Required:    false,
			},
			{
				Name:        "content",
				Type:        "string",
				Description: "Memory content to store (required for store, update)",
				Required:    false,
			},
			{
				Name:        "memory_type",
				Type:        "string",
				Description: "Type of memory: preference, fact, conversation, behavior",
				Required:    false,
				Enum:        []string{"preference", "fact", "conversation", "behavior"},
			},
			{
				Name:        "importance",
				Type:        "number",
				Description: "Memory importance level (1-10, default: 5)",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
				Maximum:     func() *float64 { v := 10.0; return &v }(),
			},
			{
				Name:        "tags",
				Type:        "array",
				Description: "Searchable tags for the memory (comma-separated)",
				Required:    false,
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query for content or topic (for search action)",
				Required:    false,
			},
			{
				Name:        "memory_id",
				Type:        "string",
				Description: "Memory ID for update/delete operations",
				Required:    false,
			},
			{
				Name:        "limit",
				Type:        "number",
				Description: "Maximum number of results to return (default: 10)",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
				Maximum:     func() *float64 { v := 100.0; return &v }(),
			},
			{
				Name:        "expires_in_days",
				Type:        "number",
				Description: "Number of days until memory expires (optional)",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
			},
		},
		Examples: []tools.Example{
			{
				Description: "Store a user preference",
				Input: map[string]interface{}{
					"action":      "store",
					"topic":       "user_preferences",
					"content":     "User prefers brief responses and technical explanations",
					"memory_type": "preference",
					"importance":  8,
					"tags":        []string{"communication", "style"},
				},
				Output: map[string]interface{}{
					"success":   true,
					"memory_id": "abc-123",
					"message":   "Memory stored successfully",
				},
			},
			{
				Description: "Recall memories about a topic",
				Input: map[string]interface{}{
					"action": "recall",
					"topic":  "user_preferences",
					"limit":  5,
				},
				Output: map[string]interface{}{
					"success":  true,
					"memories": []map[string]interface{}{},
					"count":    1,
				},
			},
			{
				Description: "Search memories by content",
				Input: map[string]interface{}{
					"action": "search",
					"query":  "technical explanations",
					"limit":  3,
				},
				Output: map[string]interface{}{
					"success":  true,
					"memories": []map[string]interface{}{},
					"count":    1,
				},
			},
		},
	}

	tool := &MemoryTool{memoryRepo: memoryRepo}
	tool.BaseTool = tools.NewBaseTool("memory", schema, tool.execute)

	return tool
}

func (m *MemoryTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	action, ok := input["action"].(string)
	if !ok {
		return tools.ErrorResult("INVALID_ACTION", "action parameter is required")
	}

	switch action {
	case "store":
		return m.handleStore(ctx, input)
	case "recall":
		return m.handleRecall(ctx, input)
	case "search":
		return m.handleSearch(ctx, input)
	case "update":
		return m.handleUpdate(ctx, input)
	case "delete":
		return m.handleDelete(ctx, input)
	case "stats":
		return m.handleStats(ctx, input)
	default:
		return tools.ErrorResult("UNKNOWN_ACTION", fmt.Sprintf("Unknown action: %s", action))
	}
}

// handleStore stores a new memory
func (m *MemoryTool) handleStore(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	topic, ok := input["topic"].(string)
	if !ok || topic == "" {
		return tools.ErrorResult("MISSING_TOPIC", "topic is required for store action")
	}

	content, ok := input["content"].(string)
	if !ok || content == "" {
		return tools.ErrorResult("MISSING_CONTENT", "content is required for store action")
	}

	memoryType := "fact" // default
	if mt, ok := input["memory_type"].(string); ok && mt != "" {
		memoryType = mt
	}

	importance := 5 // default
	if imp, ok := input["importance"]; ok {
		switch v := imp.(type) {
		case float64:
			importance = int(v)
		case int:
			importance = v
		case string:
			if parsed, err := strconv.Atoi(v); err == nil {
				importance = parsed
			}
		}
	}

	// Validate importance range
	if importance < 1 || importance > 10 {
		importance = 5
	}

	// Parse tags
	var tags []string
	if tagsInput, ok := input["tags"]; ok {
		switch v := tagsInput.(type) {
		case []interface{}:
			for _, tag := range v {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, strings.TrimSpace(tagStr))
				}
			}
		case []string:
			tags = v
		case string:
			// Support comma-separated tags
			for _, tag := range strings.Split(v, ",") {
				tags = append(tags, strings.TrimSpace(tag))
			}
		}
	}

	// Handle expiration
	var expiresAt *time.Time
	if expireDays, ok := input["expires_in_days"]; ok {
		if days, ok := expireDays.(float64); ok && days > 0 {
			expireTime := time.Now().AddDate(0, 0, int(days))
			expiresAt = &expireTime
		}
	}

	var sessionIDPtr *string
	if ctx.SessionID != "" {
		sessionIDPtr = &ctx.SessionID
	}

	// Convert tags to JSON format that models.JSON expects
	tagsData := make(map[string]interface{})
	tagsData["tags"] = tags

	// Create metadata map
	metadataMap := map[string]interface{}{"tool": "memory", "session_id": ctx.SessionID}

	memory := &models.Memory{
		AgentID:     ctx.AgentID,
		SessionID:   sessionIDPtr,
		Topic:       topic,
		Content:     content,
		MemoryType:  memoryType,
		Importance:  importance,
		Tags:        models.JSON(tagsData),
		Metadata:    models.JSON(metadataMap),
		ExpiresAt:   expiresAt,
	}

	if err := m.memoryRepo.Create(context.Background(), memory); err != nil {
		return tools.ErrorResult("STORE_FAILED", fmt.Sprintf("Failed to store memory: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":   true,
		"memory_id": memory.ID,
		"message":   "Memory stored successfully",
		"topic":     topic,
		"importance": importance,
	})
}

// handleRecall retrieves memories by topic
func (m *MemoryTool) handleRecall(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	topic, ok := input["topic"].(string)
	if !ok || topic == "" {
		return tools.ErrorResult("MISSING_TOPIC", "topic is required for recall action")
	}

	limit := 10 // default
	if lim, ok := input["limit"]; ok {
		if l, ok := lim.(float64); ok {
			limit = int(l)
		}
	}

	memories, err := m.memoryRepo.ListByTopic(context.Background(), ctx.AgentID, topic, limit, 0)
	if err != nil {
		return tools.ErrorResult("RECALL_FAILED", fmt.Sprintf("Failed to recall memories: %v", err))
	}

	memoriesData := make([]map[string]interface{}, len(memories))
	for i, memory := range memories {
		memoriesData[i] = map[string]interface{}{
			"id":          memory.ID,
			"topic":       memory.Topic,
			"content":     memory.Content,
			"memory_type": memory.MemoryType,
			"importance":  memory.Importance,
			"tags":        memory.Tags,
			"created_at":  memory.CreatedAt,
			"updated_at":  memory.UpdatedAt,
		}
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":  true,
		"memories": memoriesData,
		"count":    len(memories),
		"topic":    topic,
	})
}

// handleSearch searches memories by content or other criteria
func (m *MemoryTool) handleSearch(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	searchReq := &models.MemorySearchRequest{
		AgentID: ctx.AgentID,
	}

	// Set search query
	if query, ok := input["query"].(string); ok && query != "" {
		searchReq.Query = &query
	}

	// Set topic filter
	if topic, ok := input["topic"].(string); ok && topic != "" {
		searchReq.Topic = &topic
	}

	// Set memory type filter
	if memoryType, ok := input["memory_type"].(string); ok && memoryType != "" {
		searchReq.MemoryType = &memoryType
	}

	// Set minimum importance
	if importance, ok := input["importance"]; ok {
		if imp, ok := importance.(float64); ok {
			minImp := int(imp)
			searchReq.MinImportance = &minImp
		}
	}

	// Set limit
	limit := 10
	if lim, ok := input["limit"]; ok {
		if l, ok := lim.(float64); ok {
			limit = int(l)
		}
	}
	searchReq.Limit = &limit

	// Parse tags
	if tagsInput, ok := input["tags"]; ok {
		switch v := tagsInput.(type) {
		case []interface{}:
			for _, tag := range v {
				if tagStr, ok := tag.(string); ok {
					searchReq.Tags = append(searchReq.Tags, strings.TrimSpace(tagStr))
				}
			}
		case []string:
			searchReq.Tags = v
		case string:
			for _, tag := range strings.Split(v, ",") {
				searchReq.Tags = append(searchReq.Tags, strings.TrimSpace(tag))
			}
		}
	}

	memories, err := m.memoryRepo.Search(context.Background(), searchReq)
	if err != nil {
		return tools.ErrorResult("SEARCH_FAILED", fmt.Sprintf("Failed to search memories: %v", err))
	}

	memoriesData := make([]map[string]interface{}, len(memories))
	for i, memory := range memories {
		memoriesData[i] = map[string]interface{}{
			"id":          memory.ID,
			"topic":       memory.Topic,
			"content":     memory.Content,
			"memory_type": memory.MemoryType,
			"importance":  memory.Importance,
			"tags":        memory.Tags,
			"created_at":  memory.CreatedAt,
			"updated_at":  memory.UpdatedAt,
		}
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":  true,
		"memories": memoriesData,
		"count":    len(memories),
		"query":    searchReq.Query,
	})
}

// handleUpdate updates an existing memory
func (m *MemoryTool) handleUpdate(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	memoryID, ok := input["memory_id"].(string)
	if !ok || memoryID == "" {
		return tools.ErrorResult("MISSING_MEMORY_ID", "memory_id is required for update action")
	}

	// Get existing memory
	memory, err := m.memoryRepo.GetByID(context.Background(), memoryID)
	if err != nil {
		return tools.ErrorResult("UPDATE_FAILED", fmt.Sprintf("Failed to get memory: %v", err))
	}
	if memory == nil {
		return tools.ErrorResult("MEMORY_NOT_FOUND", "Memory not found")
	}

	// Check ownership
	if memory.AgentID != ctx.AgentID {
		return tools.ErrorResult("ACCESS_DENIED", "Cannot update memory belonging to another agent")
	}

	// Update fields if provided
	if content, ok := input["content"].(string); ok && content != "" {
		memory.Content = content
	}

	if importance, ok := input["importance"]; ok {
		if imp, ok := importance.(float64); ok {
			memory.Importance = int(imp)
		}
	}

	// Update tags if provided
	if tagsInput, ok := input["tags"]; ok {
		var tags []string
		switch v := tagsInput.(type) {
		case []interface{}:
			for _, tag := range v {
				if tagStr, ok := tag.(string); ok {
					tags = append(tags, strings.TrimSpace(tagStr))
				}
			}
		case []string:
			tags = v
		case string:
			for _, tag := range strings.Split(v, ",") {
				tags = append(tags, strings.TrimSpace(tag))
			}
		}
		tagsData := make(map[string]interface{})
		tagsData["tags"] = tags
		memory.Tags = models.JSON(tagsData)
	}

	if err := m.memoryRepo.Update(context.Background(), memory); err != nil {
		return tools.ErrorResult("UPDATE_FAILED", fmt.Sprintf("Failed to update memory: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":   true,
		"memory_id": memory.ID,
		"message":   "Memory updated successfully",
	})
}

// handleDelete deletes a memory
func (m *MemoryTool) handleDelete(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	memoryID, ok := input["memory_id"].(string)
	if !ok || memoryID == "" {
		return tools.ErrorResult("MISSING_MEMORY_ID", "memory_id is required for delete action")
	}

	// Verify memory exists and belongs to this agent
	memory, err := m.memoryRepo.GetByID(context.Background(), memoryID)
	if err != nil {
		return tools.ErrorResult("DELETE_FAILED", fmt.Sprintf("Failed to get memory: %v", err))
	}
	if memory == nil {
		return tools.ErrorResult("MEMORY_NOT_FOUND", "Memory not found")
	}
	if memory.AgentID != ctx.AgentID {
		return tools.ErrorResult("ACCESS_DENIED", "Cannot delete memory belonging to another agent")
	}

	if err := m.memoryRepo.Delete(context.Background(), memoryID); err != nil {
		return tools.ErrorResult("DELETE_FAILED", fmt.Sprintf("Failed to delete memory: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":   true,
		"memory_id": memoryID,
		"message":   "Memory deleted successfully",
	})
}

// handleStats returns memory statistics for the agent
func (m *MemoryTool) handleStats(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	stats, err := m.memoryRepo.GetStats(context.Background(), ctx.AgentID)
	if err != nil {
		return tools.ErrorResult("STATS_FAILED", fmt.Sprintf("Failed to get memory stats: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"success":              true,
		"total_memories":       stats.TotalMemories,
		"memories_by_type":     stats.MemoriesByType,
		"memories_by_topic":    stats.MemoriesByTopic,
		"average_importance":   stats.AverageImportance,
		"oldest_memory":        stats.OldestMemory,
		"newest_memory":        stats.NewestMemory,
	})
}