package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ChatSession represents a conversation session with an agent
type ChatSession struct {
	ID              string    `json:"id" gorm:"primaryKey"`
	AgentID         string    `json:"agent_id" gorm:"not null" validate:"required"`
	Title           string    `json:"title"`
	ContextStrategy string    `json:"context_strategy" gorm:"default:last_n" validate:"oneof=last_n summarize sliding_window"`
	ContextConfig   JSON      `json:"context_config" gorm:"type:json"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`

	// Relationships
	Agent    Agent     `json:"agent,omitempty" gorm:"foreignKey:AgentID"`
	Messages []Message `json:"messages,omitempty" gorm:"foreignKey:SessionID"`
}

// BeforeCreate hook to generate UUID
func (s *ChatSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	return nil
}

// CreateSessionRequest represents the request payload for creating a session
type CreateSessionRequest struct {
	Title           string                 `json:"title"`
	ContextStrategy string                 `json:"context_strategy,omitempty" validate:"omitempty,oneof=last_n summarize sliding_window"`
	ContextConfig   map[string]interface{} `json:"context_config,omitempty"`
}

// UpdateSessionRequest represents the request payload for updating a session
type UpdateSessionRequest struct {
	Title           *string                `json:"title,omitempty"`
	ContextStrategy *string                `json:"context_strategy,omitempty" validate:"omitempty,oneof=last_n summarize sliding_window"`
	ContextConfig   map[string]interface{} `json:"context_config,omitempty"`
}

// ToSession converts CreateSessionRequest to ChatSession
func (r *CreateSessionRequest) ToSession(agentID string) *ChatSession {
	session := &ChatSession{
		AgentID:         agentID,
		Title:           r.Title,
		ContextStrategy: "last_n",
		ContextConfig:   make(JSON),
	}

	if r.ContextStrategy != "" {
		session.ContextStrategy = r.ContextStrategy
	}
	if r.ContextConfig != nil {
		session.ContextConfig = JSON(r.ContextConfig)
	}

	return session
}

// UpdateFromRequest updates session fields from UpdateSessionRequest
func (s *ChatSession) UpdateFromRequest(req *UpdateSessionRequest) {
	if req.Title != nil {
		s.Title = *req.Title
	}
	if req.ContextStrategy != nil {
		s.ContextStrategy = *req.ContextStrategy
	}
	if req.ContextConfig != nil {
		s.ContextConfig = JSON(req.ContextConfig)
	}
}