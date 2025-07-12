package models

import (
	"time"
	"gorm.io/gorm"
	"github.com/google/uuid"
)

// Memory represents a stored memory entry
type Memory struct {
	ID          string     `json:"id" gorm:"type:varchar(36);primaryKey"`
	AgentID     string     `json:"agent_id" gorm:"type:varchar(36);not null;index"`
	SessionID   *string    `json:"session_id,omitempty" gorm:"type:varchar(36);index"`
	Topic       string     `json:"topic" gorm:"type:varchar(255);not null;index"`
	Content     string     `json:"content" gorm:"type:text;not null"`
	MemoryType  string     `json:"memory_type" gorm:"type:varchar(50);not null;index"` // preference, fact, conversation, behavior
	Importance  int        `json:"importance" gorm:"type:integer;not null;index"`      // 1-10 scale
	Tags        JSON       `json:"tags" gorm:"type:text"`                             // searchable tags
	Metadata    JSON       `json:"metadata" gorm:"type:text"`                         // additional context
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" gorm:"index"`                 // optional expiration
}

// BeforeCreate hook to generate UUID if not provided
func (m *Memory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == "" {
		m.ID = uuid.New().String()
	}
	return nil
}

// MemorySearchRequest represents a search request for memories
type MemorySearchRequest struct {
	AgentID     string   `json:"agent_id"`
	SessionID   *string  `json:"session_id,omitempty"`
	Topic       *string  `json:"topic,omitempty"`
	MemoryType  *string  `json:"memory_type,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Query       *string  `json:"query,omitempty"`        // content search
	MinImportance *int   `json:"min_importance,omitempty"`
	Limit       *int     `json:"limit,omitempty"`
	Offset      *int     `json:"offset,omitempty"`
}

// MemoryStoreRequest represents a request to store a memory
type MemoryStoreRequest struct {
	Topic       string            `json:"topic"`
	Content     string            `json:"content"`
	MemoryType  string            `json:"memory_type"`
	Importance  *int              `json:"importance,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
}

// MemoryUpdateRequest represents a request to update a memory
type MemoryUpdateRequest struct {
	ID          string            `json:"id"`
	Content     *string           `json:"content,omitempty"`
	Importance  *int              `json:"importance,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ExpiresAt   *time.Time        `json:"expires_at,omitempty"`
}

// MemoryStats represents memory usage statistics
type MemoryStats struct {
	TotalMemories     int                    `json:"total_memories"`
	MemoriesByType    map[string]int         `json:"memories_by_type"`
	MemoriesByTopic   map[string]int         `json:"memories_by_topic"`
	AverageImportance float64                `json:"average_importance"`
	OldestMemory      *time.Time             `json:"oldest_memory,omitempty"`
	NewestMemory      *time.Time             `json:"newest_memory,omitempty"`
}