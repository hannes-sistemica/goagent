package sqlite

import (
	"context"
	"fmt"

	"agent-server/internal/models"
	"agent-server/internal/storage"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type repository struct {
	db      *gorm.DB
	agent   storage.AgentRepository
	session storage.SessionRepository
	message storage.MessageRepository
	memory  storage.MemoryRepository
}

// NewRepository creates a new SQLite repository
func NewRepository(dbPath string) (storage.Repository, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(
		&models.Agent{}, 
		&models.ChatSession{}, 
		&models.Message{},
		&models.ToolCall{},
		&models.ToolExecutionLog{},
		&models.Memory{},
	); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	repo := &repository{
		db: db,
	}

	repo.agent = &agentRepository{db: db}
	repo.session = &sessionRepository{db: db}
	repo.message = &messageRepository{db: db}
	repo.memory = NewMemoryRepository(db)

	return repo, nil
}

func (r *repository) Agent() storage.AgentRepository {
	return r.agent
}

func (r *repository) Session() storage.SessionRepository {
	return r.session
}

func (r *repository) Message() storage.MessageRepository {
	return r.message
}

func (r *repository) Memory() storage.MemoryRepository {
	return r.memory
}

func (r *repository) Close() error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Agent repository implementation
type agentRepository struct {
	db *gorm.DB
}

func (r *agentRepository) Create(ctx context.Context, agent *models.Agent) error {
	return r.db.WithContext(ctx).Create(agent).Error
}

func (r *agentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	var agent models.Agent
	err := r.db.WithContext(ctx).First(&agent, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &agent, nil
}

func (r *agentRepository) Update(ctx context.Context, agent *models.Agent) error {
	return r.db.WithContext(ctx).Save(agent).Error
}

func (r *agentRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.Agent{}, "id = ?", id).Error
}

func (r *agentRepository) List(ctx context.Context, limit, offset int) ([]*models.Agent, int64, error) {
	var agents []*models.Agent
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Agent{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Limit(limit).
		Offset(offset).
		Order("created_at DESC").
		Find(&agents).Error

	return agents, total, err
}

// Session repository implementation
type sessionRepository struct {
	db *gorm.DB
}

func (r *sessionRepository) Create(ctx context.Context, session *models.ChatSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *sessionRepository) GetByID(ctx context.Context, id string) (*models.ChatSession, error) {
	var session models.ChatSession
	err := r.db.WithContext(ctx).Preload("Agent").First(&session, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) Update(ctx context.Context, session *models.ChatSession) error {
	return r.db.WithContext(ctx).Save(session).Error
}

func (r *sessionRepository) Delete(ctx context.Context, id string) error {
	// Delete all messages first
	if err := r.db.WithContext(ctx).Delete(&models.Message{}, "session_id = ?", id).Error; err != nil {
		return err
	}
	// Delete the session
	return r.db.WithContext(ctx).Delete(&models.ChatSession{}, "id = ?", id).Error
}

func (r *sessionRepository) ListByAgentID(ctx context.Context, agentID string, limit, offset int) ([]*models.ChatSession, int64, error) {
	var sessions []*models.ChatSession
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.ChatSession{}).Where("agent_id = ?", agentID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Where("agent_id = ?", agentID).
		Limit(limit).
		Offset(offset).
		Order("updated_at DESC").
		Find(&sessions).Error

	return sessions, total, err
}

// Message repository implementation
type messageRepository struct {
	db *gorm.DB
}

func (r *messageRepository) Create(ctx context.Context, message *models.Message) error {
	return r.db.WithContext(ctx).Create(message).Error
}

func (r *messageRepository) GetByID(ctx context.Context, id string) (*models.Message, error) {
	var message models.Message
	err := r.db.WithContext(ctx).First(&message, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &message, nil
}

func (r *messageRepository) ListBySessionID(ctx context.Context, sessionID string, limit, offset int) ([]*models.Message, int64, error) {
	var messages []*models.Message
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Message{}).Where("session_id = ?", sessionID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Limit(limit).
		Offset(offset).
		Order("created_at ASC").
		Find(&messages).Error

	return messages, total, err
}

func (r *messageRepository) DeleteBySessionID(ctx context.Context, sessionID string) error {
	return r.db.WithContext(ctx).Delete(&models.Message{}, "session_id = ?", sessionID).Error
}

func (r *messageRepository) GetLastNMessages(ctx context.Context, sessionID string, n int) ([]*models.Message, error) {
	var messages []*models.Message
	err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("created_at DESC").
		Limit(n).
		Find(&messages).Error

	if err != nil {
		return nil, err
	}

	// Reverse the slice to get chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}