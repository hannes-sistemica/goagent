package context

import (
	"agent-server/internal/models"
	"context"
	"fmt"
	"strings"
)

// SummarizeStrategy summarizes older messages and keeps recent ones
type SummarizeStrategy struct{}

func (s *SummarizeStrategy) Name() string {
	return "summarize"
}

func (s *SummarizeStrategy) DefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"max_context_length": 20,
		"keep_recent":        5,
		"summary_model":      "gpt-3.5-turbo",
	}
}

func (s *SummarizeStrategy) BuildContext(ctx context.Context, systemPrompt, agentPrompt string, messages []*models.Message, config map[string]interface{}) ([]*models.Message, error) {
	// Get configuration
	maxContextLength := 20
	keepRecent := 5

	if mcl, ok := config["max_context_length"]; ok {
		if mclInt, ok := mcl.(int); ok {
			maxContextLength = mclInt
		} else if mclFloat, ok := mcl.(float64); ok {
			maxContextLength = int(mclFloat)
		}
	}

	if kr, ok := config["keep_recent"]; ok {
		if krInt, ok := kr.(int); ok {
			keepRecent = krInt
		} else if krFloat, ok := kr.(float64); ok {
			keepRecent = int(krFloat)
		}
	}

	if maxContextLength <= 0 || keepRecent <= 0 {
		return []*models.Message{}, fmt.Errorf("max_context_length and keep_recent must be positive")
	}

	// Create system message
	contextMessages := []*models.Message{
		{
			Role:    "system",
			Content: buildSystemMessage(systemPrompt, agentPrompt),
		},
	}

	// If we have fewer messages than max context length, return all
	if len(messages) <= maxContextLength {
		contextMessages = append(contextMessages, messages...)
		return contextMessages, nil
	}

	// Calculate how many messages to summarize
	messagesToSummarize := len(messages) - keepRecent
	if messagesToSummarize <= 0 {
		contextMessages = append(contextMessages, messages...)
		return contextMessages, nil
	}

	// Get messages to summarize
	oldMessages := messages[:messagesToSummarize]
	recentMessages := messages[messagesToSummarize:]

	// Create summary (simplified version - in production you'd call an LLM)
	summary := s.createSimpleSummary(oldMessages)

	// Add summary as a system message
	if summary != "" {
		contextMessages = append(contextMessages, &models.Message{
			Role:    "system",
			Content: fmt.Sprintf("Previous conversation summary: %s", summary),
		})
	}

	// Add recent messages
	contextMessages = append(contextMessages, recentMessages...)

	return contextMessages, nil
}

// createSimpleSummary creates a basic summary of messages
// In production, this would call an LLM service
func (s *SummarizeStrategy) createSimpleSummary(messages []*models.Message) string {
	if len(messages) == 0 {
		return ""
	}

	var summary strings.Builder
	userMessages := 0
	assistantMessages := 0

	for _, msg := range messages {
		switch msg.Role {
		case "user":
			userMessages++
		case "assistant":
			assistantMessages++
		}
	}

	summary.WriteString(fmt.Sprintf("The conversation included %d user messages and %d assistant responses", userMessages, assistantMessages))

	// Add topic hints based on content
	topics := s.extractTopics(messages)
	if len(topics) > 0 {
		summary.WriteString(fmt.Sprintf(". Topics discussed: %s", strings.Join(topics, ", ")))
	}

	summary.WriteString(".")

	return summary.String()
}

// extractTopics extracts basic topics from message content
func (s *SummarizeStrategy) extractTopics(messages []*models.Message) []string {
	// This is a very simplified topic extraction
	// In production, you'd use NLP techniques or LLM-based extraction
	topicWords := map[string]bool{
		"code":        false,
		"programming": false,
		"bug":         false,
		"error":       false,
		"help":        false,
		"question":    false,
		"problem":     false,
		"solution":    false,
	}

	for _, msg := range messages {
		content := strings.ToLower(msg.Content)
		for topic := range topicWords {
			if strings.Contains(content, topic) {
				topicWords[topic] = true
			}
		}
	}

	var topics []string
	for topic, found := range topicWords {
		if found {
			topics = append(topics, topic)
		}
	}

	return topics
}