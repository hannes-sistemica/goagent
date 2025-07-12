package context

import (
	"agent-server/internal/models"
	"context"
	"fmt"
)

// LastNStrategy takes the last N messages for context
type LastNStrategy struct{}

func (s *LastNStrategy) Name() string {
	return "last_n"
}

func (s *LastNStrategy) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"count": 10,
	}
}

func (s *LastNStrategy) BuildContext(ctx context.Context, systemPrompt, agentPrompt string, messages []*models.Message, config map[string]interface{}) ([]*models.Message, error) {
	// Get count from config
	count := 10
	if c, ok := config["count"]; ok {
		if countInt, ok := c.(int); ok {
			count = countInt
		} else if countFloat, ok := c.(float64); ok {
			count = int(countFloat)
		}
	}

	if count <= 0 {
		return []*models.Message{}, fmt.Errorf("count must be positive")
	}

	// Create system message
	contextMessages := []*models.Message{
		{
			Role:    "system",
			Content: buildSystemMessage(systemPrompt, agentPrompt),
		},
	}

	// Take the last N messages
	startIndex := 0
	if len(messages) > count {
		startIndex = len(messages) - count
	}

	// Append the last N messages
	contextMessages = append(contextMessages, messages[startIndex:]...)

	return contextMessages, nil
}

// buildSystemMessage combines system prompt and agent prompt
func buildSystemMessage(systemPrompt, agentPrompt string) string {
	if systemPrompt == "" && agentPrompt == "" {
		return "You are a helpful AI assistant."
	}
	
	if systemPrompt == "" {
		return agentPrompt
	}
	
	if agentPrompt == "" {
		return systemPrompt
	}
	
	return systemPrompt + "\n\n" + agentPrompt
}