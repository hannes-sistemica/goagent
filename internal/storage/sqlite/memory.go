package sqlite

import (
	"context"
	"fmt"
	"strings"
	"time"

	"agent-server/internal/models"
	"agent-server/internal/storage"

	"gorm.io/gorm"
)

// memoryRepository implements storage.MemoryRepository using GORM
type memoryRepository struct {
	db *gorm.DB
}

// NewMemoryRepository creates a new GORM-based memory repository
func NewMemoryRepository(db *gorm.DB) storage.MemoryRepository {
	return &memoryRepository{db: db}
}

// Create stores a new memory
func (r *memoryRepository) Create(ctx context.Context, memory *models.Memory) error {
	return r.db.WithContext(ctx).Create(memory).Error
}

// GetByID retrieves a memory by its ID
func (r *memoryRepository) GetByID(ctx context.Context, id string) (*models.Memory, error) {
	var memory models.Memory
	err := r.db.WithContext(ctx).
		Where("id = ? AND (expires_at IS NULL OR expires_at > ?)", id, time.Now()).
		First(&memory).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	
	return &memory, nil
}

// Update modifies an existing memory
func (r *memoryRepository) Update(ctx context.Context, memory *models.Memory) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", memory.ID).
		Updates(memory)
	
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("memory not found")
	}
	
	return nil
}

// Delete removes a memory by ID
func (r *memoryRepository) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&models.Memory{}, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}
	
	if result.RowsAffected == 0 {
		return fmt.Errorf("memory not found")
	}
	
	return nil
}

// Search finds memories based on search criteria
func (r *memoryRepository) Search(ctx context.Context, req *models.MemorySearchRequest) ([]*models.Memory, error) {
	query := r.db.WithContext(ctx).Model(&models.Memory{}).
		Where("agent_id = ? AND (expires_at IS NULL OR expires_at > ?)", req.AgentID, time.Now())
	
	// Add optional filters
	if req.SessionID != nil {
		query = query.Where("session_id = ?", *req.SessionID)
	}
	
	if req.Topic != nil {
		query = query.Where("topic = ?", *req.Topic)
	}
	
	if req.MemoryType != nil {
		query = query.Where("memory_type = ?", *req.MemoryType)
	}
	
	if req.MinImportance != nil {
		query = query.Where("importance >= ?", *req.MinImportance)
	}
	
	if req.Query != nil {
		searchTerm := "%" + *req.Query + "%"
		query = query.Where("content LIKE ? OR topic LIKE ?", searchTerm, searchTerm)
	}
	
	// Handle tags search - this is more complex with JSON fields
	if len(req.Tags) > 0 {
		var tagConditions []string
		var tagArgs []interface{}
		
		for _, tag := range req.Tags {
			tagConditions = append(tagConditions, "tags LIKE ?")
			tagArgs = append(tagArgs, "%\""+tag+"\"%")
		}
		
		if len(tagConditions) > 0 {
			query = query.Where("("+strings.Join(tagConditions, " OR ")+")", tagArgs...)
		}
	}
	
	// Order by importance (descending) and created_at (descending)
	query = query.Order("importance DESC, created_at DESC")
	
	// Add pagination
	if req.Limit != nil {
		query = query.Limit(*req.Limit)
		
		if req.Offset != nil {
			query = query.Offset(*req.Offset)
		}
	}

	var memories []*models.Memory
	err := query.Find(&memories).Error
	return memories, err
}

// ListByAgent retrieves all memories for an agent
func (r *memoryRepository) ListByAgent(ctx context.Context, agentID string, limit, offset int) ([]*models.Memory, error) {
	req := &models.MemorySearchRequest{
		AgentID: agentID,
		Limit:   &limit,
		Offset:  &offset,
	}
	return r.Search(ctx, req)
}

// ListByTopic retrieves memories for a specific topic
func (r *memoryRepository) ListByTopic(ctx context.Context, agentID, topic string, limit, offset int) ([]*models.Memory, error) {
	req := &models.MemorySearchRequest{
		AgentID: agentID,
		Topic:   &topic,
		Limit:   &limit,
		Offset:  &offset,
	}
	return r.Search(ctx, req)
}

// GetStats returns memory usage statistics for an agent
func (r *memoryRepository) GetStats(ctx context.Context, agentID string) (*models.MemoryStats, error) {
	stats := &models.MemoryStats{
		MemoriesByType:  make(map[string]int),
		MemoriesByTopic: make(map[string]int),
	}

	baseQuery := r.db.WithContext(ctx).Model(&models.Memory{}).
		Where("agent_id = ? AND (expires_at IS NULL OR expires_at > ?)", agentID, time.Now())

	// Get total count and average importance
	var totalCount int64
	var sumImportance int64
	
	result := baseQuery.Select("COUNT(*) as count, COALESCE(SUM(importance), 0) as sum_importance").
		Row()
	
	if err := result.Scan(&totalCount, &sumImportance); err != nil {
		return nil, err
	}
	
	stats.TotalMemories = int(totalCount)
	if totalCount > 0 {
		stats.AverageImportance = float64(sumImportance) / float64(totalCount)
	}

	// Get oldest and newest memory timestamps
	var oldest, newest time.Time
	if totalCount > 0 {
		baseQuery.Select("MIN(created_at)").Row().Scan(&oldest)
		baseQuery.Select("MAX(created_at)").Row().Scan(&newest)
		stats.OldestMemory = &oldest
		stats.NewestMemory = &newest
	}

	// Get memories by type
	type TypeCount struct {
		MemoryType string
		Count      int64
	}
	
	var typeCounts []TypeCount
	err := baseQuery.Select("memory_type, COUNT(*) as count").
		Group("memory_type").
		Find(&typeCounts).Error
	if err != nil {
		return nil, err
	}
	
	for _, tc := range typeCounts {
		stats.MemoriesByType[tc.MemoryType] = int(tc.Count)
	}

	// Get memories by topic (top 10)
	type TopicCount struct {
		Topic string
		Count int64
	}
	
	var topicCounts []TopicCount
	err = baseQuery.Select("topic, COUNT(*) as count").
		Group("topic").
		Order("count DESC").
		Limit(10).
		Find(&topicCounts).Error
	if err != nil {
		return nil, err
	}
	
	for _, tc := range topicCounts {
		stats.MemoriesByTopic[tc.Topic] = int(tc.Count)
	}

	return stats, nil
}

// DeleteExpired removes expired memories
func (r *memoryRepository) DeleteExpired(ctx context.Context) (int, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at IS NOT NULL AND expires_at <= ?", time.Now()).
		Delete(&models.Memory{})
	
	if result.Error != nil {
		return 0, result.Error
	}
	
	return int(result.RowsAffected), nil
}

// DeleteByAgent removes all memories for an agent
func (r *memoryRepository) DeleteByAgent(ctx context.Context, agentID string) error {
	return r.db.WithContext(ctx).Where("agent_id = ?", agentID).Delete(&models.Memory{}).Error
}