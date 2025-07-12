package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateSessionRequest_ToSession(t *testing.T) {
	agentID := "test-agent-id"

	tests := []struct {
		name     string
		request  CreateSessionRequest
		expected ChatSession
	}{
		{
			name: "basic session creation",
			request: CreateSessionRequest{
				Title: "Test Session",
			},
			expected: ChatSession{
				AgentID:         agentID,
				Title:           "Test Session",
				ContextStrategy: "last_n",
				ContextConfig:   make(JSON),
			},
		},
		{
			name: "session with custom strategy",
			request: CreateSessionRequest{
				Title:           "Custom Session",
				ContextStrategy: "sliding_window",
				ContextConfig: map[string]interface{}{
					"window_size": 5,
					"overlap":     2,
				},
			},
			expected: ChatSession{
				AgentID:         agentID,
				Title:           "Custom Session",
				ContextStrategy: "sliding_window",
				ContextConfig: JSON{
					"window_size": 5,
					"overlap":     2,
				},
			},
		},
		{
			name: "session with empty title",
			request: CreateSessionRequest{
				ContextStrategy: "summarize",
			},
			expected: ChatSession{
				AgentID:         agentID,
				Title:           "",
				ContextStrategy: "summarize",
				ContextConfig:   make(JSON),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := tt.request.ToSession(agentID)
			assert.Equal(t, tt.expected.AgentID, session.AgentID)
			assert.Equal(t, tt.expected.Title, session.Title)
			assert.Equal(t, tt.expected.ContextStrategy, session.ContextStrategy)
			assert.Equal(t, tt.expected.ContextConfig, session.ContextConfig)
		})
	}
}

func TestChatSession_UpdateFromRequest(t *testing.T) {
	session := &ChatSession{
		AgentID:         "agent-id",
		Title:           "Original Title",
		ContextStrategy: "last_n",
		ContextConfig:   JSON{"count": 10},
	}

	tests := []struct {
		name    string
		request UpdateSessionRequest
		check   func(t *testing.T, session *ChatSession)
	}{
		{
			name: "update title only",
			request: UpdateSessionRequest{
				Title: func() *string { s := "Updated Title"; return &s }(),
			},
			check: func(t *testing.T, session *ChatSession) {
				assert.Equal(t, "Updated Title", session.Title)
				assert.Equal(t, "last_n", session.ContextStrategy)
			},
		},
		{
			name: "update strategy and config",
			request: UpdateSessionRequest{
				ContextStrategy: func() *string { s := "sliding_window"; return &s }(),
				ContextConfig: map[string]interface{}{
					"window_size": 8,
					"overlap":     3,
				},
			},
			check: func(t *testing.T, session *ChatSession) {
				assert.Equal(t, "sliding_window", session.ContextStrategy)
				expected := JSON{
					"window_size": 8,
					"overlap":     3,
				}
				assert.Equal(t, expected, session.ContextConfig)
			},
		},
		{
			name: "update all fields",
			request: UpdateSessionRequest{
				Title:           func() *string { s := "New Title"; return &s }(),
				ContextStrategy: func() *string { s := "summarize"; return &s }(),
				ContextConfig: map[string]interface{}{
					"max_context_length": 15,
					"keep_recent":        3,
				},
			},
			check: func(t *testing.T, session *ChatSession) {
				assert.Equal(t, "New Title", session.Title)
				assert.Equal(t, "summarize", session.ContextStrategy)
				expected := JSON{
					"max_context_length": 15,
					"keep_recent":        3,
				}
				assert.Equal(t, expected, session.ContextConfig)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the original session
			testSession := *session
			testSession.ContextConfig = make(JSON)
			for k, v := range session.ContextConfig {
				testSession.ContextConfig[k] = v
			}

			testSession.UpdateFromRequest(&tt.request)
			tt.check(t, &testSession)
		})
	}
}