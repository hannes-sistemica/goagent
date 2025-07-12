package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Message represents a single message in a chat session
type Message struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	SessionID string    `json:"session_id" gorm:"not null" validate:"required"`
	Role      string    `json:"role" gorm:"not null" validate:"required,oneof=user assistant system"`
	Content   string    `json:"content" gorm:"type:text;not null" validate:"required"`
	Metadata  JSON      `json:"metadata" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`

	// Relationships
	Session ChatSession `json:"session,omitempty" gorm:"foreignKey:SessionID"`
}

// BeforeCreate hook to generate UUID
func (m *Message) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// CreateMessageRequest represents the request payload for creating a message
type CreateMessageRequest struct {
	Role     string                 `json:"role" validate:"required,oneof=user assistant system"`
	Content  string                 `json:"content" validate:"required"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ChatRequest represents a request to chat with an agent
type ChatRequest struct {
	Message  string                 `json:"message" validate:"required"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Stream   bool                   `json:"stream,omitempty"`
}

// ChatResponse represents a response from the chat API
type ChatResponse struct {
	MessageID string                 `json:"message_id"`
	Response  string                 `json:"response"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// StreamChunk represents a chunk of streaming response
type StreamChunk struct {
	Delta    string                 `json:"delta"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
	Done     bool                   `json:"done"`
}

// ToMessage converts CreateMessageRequest to Message
func (r *CreateMessageRequest) ToMessage(sessionID string) *Message {
	message := &Message{
		SessionID: sessionID,
		Role:      r.Role,
		Content:   r.Content,
		Metadata:  make(JSON),
	}

	if r.Metadata != nil {
		message.Metadata = JSON(r.Metadata)
	}

	return message
}

// MessageList represents a paginated list of messages
type MessageList struct {
	Messages   []Message `json:"messages"`
	TotalCount int64     `json:"total_count"`
	Page       int       `json:"page"`
	PageSize   int       `json:"page_size"`
	HasMore    bool      `json:"has_more"`
}