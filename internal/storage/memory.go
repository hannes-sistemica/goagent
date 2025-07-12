package storage

import (
	"context"
	"agent-server/internal/models"
)

// MemoryRepository defines the interface for memory storage operations
type MemoryRepository interface {
	// Create stores a new memory
	Create(ctx context.Context, memory *models.Memory) error
	
	// GetByID retrieves a memory by its ID
	GetByID(ctx context.Context, id string) (*models.Memory, error)
	
	// Update modifies an existing memory
	Update(ctx context.Context, memory *models.Memory) error
	
	// Delete removes a memory by ID
	Delete(ctx context.Context, id string) error
	
	// Search finds memories based on search criteria
	Search(ctx context.Context, req *models.MemorySearchRequest) ([]*models.Memory, error)
	
	// ListByAgent retrieves all memories for an agent
	ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]*models.Memory, error)
	
	// ListByTopic retrieves memories for a specific topic
	ListByTopic(ctx context.Context, agentID, topic string, limit, offset int) ([]*models.Memory, error)
	
	// GetStats returns memory usage statistics for an agent
	GetStats(ctx context.Context, agentID string) (*models.MemoryStats, error)
	
	// DeleteExpired removes expired memories
	DeleteExpired(ctx context.Context) (int, error)
	
	// DeleteByAgent removes all memories for an agent
	DeleteByAgent(ctx context.Context, agentID string) error
}