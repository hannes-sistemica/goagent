package context

import (
	"context"
	"testing"

	"agent-server/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLastNStrategy_Name(t *testing.T) {
	strategy := &LastNStrategy{}
	assert.Equal(t, "last_n", strategy.Name())
}

func TestLastNStrategy_DefaultConfig(t *testing.T) {
	strategy := &LastNStrategy{}
	config := strategy.DefaultConfig()
	assert.Equal(t, 10, config["count"])
}

func TestLastNStrategy_BuildContext(t *testing.T) {
	strategy := &LastNStrategy{}
	ctx := context.Background()

	// Create test messages
	messages := []*models.Message{
		{Role: "user", Content: "Message 1"},
		{Role: "assistant", Content: "Response 1"},
		{Role: "user", Content: "Message 2"},
		{Role: "assistant", Content: "Response 2"},
		{Role: "user", Content: "Message 3"},
		{Role: "assistant", Content: "Response 3"},
		{Role: "user", Content: "Message 4"},
		{Role: "assistant", Content: "Response 4"},
		{Role: "user", Content: "Message 5"},
		{Role: "assistant", Content: "Response 5"},
		{Role: "user", Content: "Message 6"},
	}

	tests := []struct {
		name           string
		systemPrompt   string
		agentPrompt    string
		config         map[string]interface{}
		expectedCount  int
		expectedSystem string
		expectError    bool
	}{
		{
			name:         "default count",
			systemPrompt: "You are helpful",
			config:       map[string]interface{}{"count": 10},
			expectedCount: 11, // 1 system + 10 last messages
			expectedSystem: "You are helpful",
		},
		{
			name:         "count of 5",
			systemPrompt: "You are helpful",
			config:       map[string]interface{}{"count": 5},
			expectedCount: 6, // 1 system + 5 last messages
			expectedSystem: "You are helpful",
		},
		{
			name:         "count of 3",
			systemPrompt: "System prompt",
			agentPrompt:  "Agent prompt",
			config:       map[string]interface{}{"count": 3},
			expectedCount: 4, // 1 system + 3 last messages
			expectedSystem: "System prompt\n\nAgent prompt",
		},
		{
			name:         "count larger than available",
			systemPrompt: "You are helpful",
			config:       map[string]interface{}{"count": 20},
			expectedCount: 12, // 1 system + 11 messages (all available)
		},
		{
			name:        "zero count",
			config:      map[string]interface{}{"count": 0},
			expectError: true,
		},
		{
			name:        "negative count",
			config:      map[string]interface{}{"count": -1},
			expectError: true,
		},
		{
			name:         "count as float",
			systemPrompt: "You are helpful",
			config:       map[string]interface{}{"count": 5.0},
			expectedCount: 6, // 1 system + 5 last messages
		},
		{
			name:         "no config uses default",
			systemPrompt: "You are helpful",
			config:       map[string]interface{}{},
			expectedCount: 11, // 1 system + 10 last messages (default count is 10)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strategy.BuildContext(ctx, tt.systemPrompt, tt.agentPrompt, messages, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)

			// First message should always be system
			assert.Equal(t, "system", result[0].Role)
			if tt.expectedSystem != "" {
				assert.Equal(t, tt.expectedSystem, result[0].Content)
			}

			// Rest should be from the original messages (last N)
			if len(result) > 1 {
				expectedStartIndex := len(messages) - (tt.expectedCount - 1)
				if expectedStartIndex < 0 {
					expectedStartIndex = 0
				}

				for i := 1; i < len(result) && i-1+expectedStartIndex < len(messages); i++ {
					originalIndex := expectedStartIndex + i - 1
					if originalIndex >= 0 && originalIndex < len(messages) {
						assert.Equal(t, messages[originalIndex].Role, result[i].Role, "Role mismatch at index %d", i)
						assert.Equal(t, messages[originalIndex].Content, result[i].Content, "Content mismatch at index %d", i)
					}
				}
			}
		})
	}
}

func TestLastNStrategy_BuildContext_EmptyMessages(t *testing.T) {
	strategy := &LastNStrategy{}
	ctx := context.Background()

	result, err := strategy.BuildContext(ctx, "System prompt", "", []*models.Message{}, map[string]interface{}{"count": 5})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "system", result[0].Role)
	assert.Equal(t, "System prompt", result[0].Content)
}

func TestBuildSystemMessage(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		agentPrompt  string
		expected     string
	}{
		{
			name:         "both prompts",
			systemPrompt: "You are helpful",
			agentPrompt:  "Answer briefly",
			expected:     "You are helpful\n\nAnswer briefly",
		},
		{
			name:         "system prompt only",
			systemPrompt: "You are helpful",
			agentPrompt:  "",
			expected:     "You are helpful",
		},
		{
			name:         "agent prompt only",
			systemPrompt: "",
			agentPrompt:  "Answer briefly",
			expected:     "Answer briefly",
		},
		{
			name:         "no prompts",
			systemPrompt: "",
			agentPrompt:  "",
			expected:     "You are a helpful AI assistant.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSystemMessage(tt.systemPrompt, tt.agentPrompt)
			assert.Equal(t, tt.expected, result)
		})
	}
}