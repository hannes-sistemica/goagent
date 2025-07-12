package context

import (
	"agent-server/internal/models"
	"context"
	"fmt"
)

// SlidingWindowStrategy maintains a sliding window of messages with overlap
type SlidingWindowStrategy struct{}

func (s *SlidingWindowStrategy) Name() string {
	return "sliding_window"
}

func (s *SlidingWindowStrategy) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"window_size": 5,
		"overlap":     2,
	}
}

func (s *SlidingWindowStrategy) BuildContext(ctx context.Context, systemPrompt, agentPrompt string, messages []*models.Message, config map[string]interface{}) ([]*models.Message, error) {
	// Get configuration
	windowSize := 5
	overlap := 2

	if ws, ok := config["window_size"]; ok {
		if wsInt, ok := ws.(int); ok {
			windowSize = wsInt
		} else if wsFloat, ok := ws.(float64); ok {
			windowSize = int(wsFloat)
		}
	}

	if o, ok := config["overlap"]; ok {
		if oInt, ok := o.(int); ok {
			overlap = oInt
		} else if oFloat, ok := o.(float64); ok {
			overlap = int(oFloat)
		}
	}

	if windowSize <= 0 {
		return []*models.Message{}, fmt.Errorf("window_size must be positive")
	}
	if overlap < 0 || overlap >= windowSize {
		return []*models.Message{}, fmt.Errorf("overlap must be between 0 and window_size-1")
	}

	// Create system message
	contextMessages := []*models.Message{
		{
			Role:    "system",
			Content: buildSystemMessage(systemPrompt, agentPrompt),
		},
	}

	// If we have fewer messages than window size, return all
	if len(messages) <= windowSize {
		contextMessages = append(contextMessages, messages...)
		return contextMessages, nil
	}

	// Calculate the start of the sliding window
	// We want to keep the most recent messages, so we slide from the end
	start := len(messages) - windowSize
	
	// Include overlap from previous window if available
	if start > overlap {
		start -= overlap
		windowSize += overlap
	} else {
		start = 0
		windowSize = len(messages)
	}

	// Take messages from the sliding window
	contextMessages = append(contextMessages, messages[start:start+windowSize]...)

	return contextMessages, nil
}