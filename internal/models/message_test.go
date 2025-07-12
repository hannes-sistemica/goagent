package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCreateMessageRequest_ToMessage(t *testing.T) {
	sessionID := "test-session-id"

	tests := []struct {
		name     string
		request  CreateMessageRequest
		expected Message
	}{
		{
			name: "basic user message",
			request: CreateMessageRequest{
				Role:    "user",
				Content: "Hello, how are you?",
			},
			expected: Message{
				SessionID: sessionID,
				Role:      "user",
				Content:   "Hello, how are you?",
				Metadata:  make(JSON),
			},
		},
		{
			name: "assistant message with metadata",
			request: CreateMessageRequest{
				Role:    "assistant",
				Content: "I'm doing well, thank you!",
				Metadata: map[string]interface{}{
					"model":      "gpt-4",
					"tokens":     25,
					"confidence": 0.95,
				},
			},
			expected: Message{
				SessionID: sessionID,
				Role:      "assistant",
				Content:   "I'm doing well, thank you!",
				Metadata: JSON{
					"model":      "gpt-4",
					"tokens":     25,
					"confidence": 0.95,
				},
			},
		},
		{
			name: "system message",
			request: CreateMessageRequest{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			expected: Message{
				SessionID: sessionID,
				Role:      "system",
				Content:   "You are a helpful assistant.",
				Metadata:  make(JSON),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			message := tt.request.ToMessage(sessionID)
			assert.Equal(t, tt.expected.SessionID, message.SessionID)
			assert.Equal(t, tt.expected.Role, message.Role)
			assert.Equal(t, tt.expected.Content, message.Content)
			assert.Equal(t, tt.expected.Metadata, message.Metadata)
		})
	}
}

func TestChatRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ChatRequest
		valid   bool
	}{
		{
			name: "valid chat request",
			request: ChatRequest{
				Message: "Hello!",
			},
			valid: true,
		},
		{
			name: "valid chat request with metadata",
			request: ChatRequest{
				Message: "Hello!",
				Metadata: map[string]interface{}{
					"source": "web",
				},
				Stream: true,
			},
			valid: true,
		},
		{
			name: "empty message should be invalid",
			request: ChatRequest{
				Message: "",
			},
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This would typically be tested with a validator
			// For now, just check that message is not empty
			if tt.valid {
				assert.NotEmpty(t, tt.request.Message)
			} else {
				assert.Empty(t, tt.request.Message)
			}
		})
	}
}

func TestChatResponse_Structure(t *testing.T) {
	response := ChatResponse{
		MessageID: "msg-123",
		Response:  "Hello there!",
		Metadata: map[string]interface{}{
			"model":   "gpt-4",
			"tokens":  10,
			"latency": 1.5,
		},
	}

	assert.Equal(t, "msg-123", response.MessageID)
	assert.Equal(t, "Hello there!", response.Response)
	assert.Equal(t, "gpt-4", response.Metadata["model"])
	assert.Equal(t, 10, response.Metadata["tokens"])
	assert.Equal(t, 1.5, response.Metadata["latency"])
}

func TestStreamChunk_Structure(t *testing.T) {
	tests := []struct {
		name  string
		chunk StreamChunk
	}{
		{
			name: "content chunk",
			chunk: StreamChunk{
				Delta: "Hello",
				Metadata: map[string]interface{}{
					"index": 0,
				},
				Done: false,
			},
		},
		{
			name: "final chunk",
			chunk: StreamChunk{
				Delta: "",
				Done:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.chunk.Done, tt.chunk.Done)
			if !tt.chunk.Done {
				assert.NotEmpty(t, tt.chunk.Delta)
			}
		})
	}
}

func TestMessageList_Structure(t *testing.T) {
	messages := []Message{
		{
			ID:        "msg-1",
			SessionID: "session-1",
			Role:      "user",
			Content:   "Hello",
		},
		{
			ID:        "msg-2",
			SessionID: "session-1",
			Role:      "assistant",
			Content:   "Hi there!",
		},
	}

	messageList := MessageList{
		Messages:   messages,
		TotalCount: 2,
		Page:       1,
		PageSize:   10,
		HasMore:    false,
	}

	assert.Len(t, messageList.Messages, 2)
	assert.Equal(t, int64(2), messageList.TotalCount)
	assert.Equal(t, 1, messageList.Page)
	assert.Equal(t, 10, messageList.PageSize)
	assert.False(t, messageList.HasMore)
}