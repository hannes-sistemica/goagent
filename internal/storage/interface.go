package storage

import (
	"context"

	"agent-server/internal/models"
)

// AgentRepository defines the interface for agent storage operations
type AgentRepository interface {
	Create(ctx context.Context, agent *models.Agent) error
	GetByID(ctx context.Context, id string) (*models.Agent, error)
	Update(ctx context.Context, agent *models.Agent) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context, limit, offset int) ([]*models.Agent, int64, error)
}

// SessionRepository defines the interface for session storage operations
type SessionRepository interface {
	Create(ctx context.Context, session *models.ChatSession) error
	GetByID(ctx context.Context, id string) (*models.ChatSession, error)
	Update(ctx context.Context, session *models.ChatSession) error
	Delete(ctx context.Context, id string) error
	ListByAgentID(ctx context.Context, agentID string, limit, offset int) ([]*models.ChatSession, int64, error)
}

// MessageRepository defines the interface for message storage operations
type MessageRepository interface {
	Create(ctx context.Context, message *models.Message) error
	GetByID(ctx context.Context, id string) (*models.Message, error)
	ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*models.Message, int64, error)
	DeleteBySessionID(ctx context.Context, sessionID string) error
	GetLastNMessages(ctx context.Context, sessionID string, n int) ([]*models.Message, error)
}

// Repository aggregates all repository interfaces
type Repository interface {
	Agent() AgentRepository
	Session() SessionRepository
	Message() MessageRepository
	Memory() MemoryRepository
	Close() error
}