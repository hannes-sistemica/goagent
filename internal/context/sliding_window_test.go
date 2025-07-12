package context

import (
	"context"
	"testing"

	"agent-server/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSlidingWindowStrategy_Name(t *testing.T) {
	strategy := &SlidingWindowStrategy{}
	assert.Equal(t, "sliding_window", strategy.Name())
}

func TestSlidingWindowStrategy_DefaultConfig(t *testing.T) {
	strategy := &SlidingWindowStrategy{}
	config := strategy.DefaultConfig()
	assert.Equal(t, 5, config["window_size"])
	assert.Equal(t, 2, config["overlap"])
}

func TestSlidingWindowStrategy_BuildContext(t *testing.T) {
	strategy := &SlidingWindowStrategy{}
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
		{Role: "assistant", Content: "Response 6"},
	}

	tests := []struct {
		name          string
		systemPrompt  string
		config        map[string]interface{}
		expectedCount int
		expectError   bool
	}{
		{
			name:         "default config",
			systemPrompt: "You are helpful",
			config: map[string]interface{}{
				"window_size": 5,
				"overlap":     2,
			},
			expectedCount: 8, // 1 system + 7 messages (5 + 2 overlap)
		},
		{
			name:         "window size 3 overlap 1",
			systemPrompt: "You are helpful",
			config: map[string]interface{}{
				"window_size": 3,
				"overlap":     1,
			},
			expectedCount: 5, // 1 system + 4 messages (3 + 1 overlap)
		},
		{
			name:         "window larger than messages",
			systemPrompt: "You are helpful",
			config: map[string]interface{}{
				"window_size": 20,
				"overlap":     5,
			},
			expectedCount: 13, // 1 system + all 12 messages
		},
		{
			name:         "no overlap",
			systemPrompt: "You are helpful",
			config: map[string]interface{}{
				"window_size": 4,
				"overlap":     0,
			},
			expectedCount: 5, // 1 system + 4 last messages
		},
		{
			name: "zero window size",
			config: map[string]interface{}{
				"window_size": 0,
				"overlap":     1,
			},
			expectError: true,
		},
		{
			name: "negative overlap",
			config: map[string]interface{}{
				"window_size": 5,
				"overlap":     -1,
			},
			expectError: true,
		},
		{
			name: "overlap >= window_size",
			config: map[string]interface{}{
				"window_size": 5,
				"overlap":     5,
			},
			expectError: true,
		},
		{
			name:         "float values",
			systemPrompt: "You are helpful",
			config: map[string]interface{}{
				"window_size": 4.0,
				"overlap":     1.0,
			},
			expectedCount: 6, // 1 system + 5 messages (4 + 1 overlap)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := strategy.BuildContext(ctx, tt.systemPrompt, "", messages, tt.config)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.expectedCount)

			// First message should always be system
			assert.Equal(t, "system", result[0].Role)
			assert.Equal(t, tt.systemPrompt, result[0].Content)

			// Verify the sliding window logic
			if len(result) > 1 {
				windowSize := 5
				overlap := 2

				if ws, ok := tt.config["window_size"]; ok {
					if wsInt, ok := ws.(int); ok {
						windowSize = wsInt
					} else if wsFloat, ok := ws.(float64); ok {
						windowSize = int(wsFloat)
					}
				}

				if o, ok := tt.config["overlap"]; ok {
					if oInt, ok := o.(int); ok {
						overlap = oInt
					} else if oFloat, ok := o.(float64); ok {
						overlap = int(oFloat)
					}
				}

				actualMessageCount := len(result) - 1

				// If we have fewer messages than window size, should get all messages
				if len(messages) <= windowSize {
					assert.Equal(t, len(messages), actualMessageCount)
				} else {
					// Should get window size + overlap (if overlap is possible)
					if len(messages) > windowSize+overlap {
						assert.Equal(t, windowSize+overlap, actualMessageCount)
					} else {
						assert.Equal(t, len(messages), actualMessageCount)
					}
				}
			}
		})
	}
}

func TestSlidingWindowStrategy_BuildContext_EmptyMessages(t *testing.T) {
	strategy := &SlidingWindowStrategy{}
	ctx := context.Background()

	result, err := strategy.BuildContext(ctx, "System prompt", "", []*models.Message{}, map[string]interface{}{
		"window_size": 5,
		"overlap":     2,
	})

	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "system", result[0].Role)
	assert.Equal(t, "System prompt", result[0].Content)
}

func TestSlidingWindowStrategy_BuildContext_FewMessages(t *testing.T) {
	strategy := &SlidingWindowStrategy{}
	ctx := context.Background()

	messages := []*models.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi there!"},
	}

	result, err := strategy.BuildContext(ctx, "System prompt", "", messages, map[string]interface{}{
		"window_size": 5,
		"overlap":     2,
	})

	require.NoError(t, err)
	assert.Len(t, result, 3) // 1 system + 2 messages
	assert.Equal(t, "system", result[0].Role)
	assert.Equal(t, "user", result[1].Role)
	assert.Equal(t, "assistant", result[2].Role)
}