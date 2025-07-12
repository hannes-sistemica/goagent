package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// JSON is a custom type for storing JSON data in the database
type JSON map[string]interface{}

func (j JSON) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, j)
}

// Agent represents an AI agent configuration
type Agent struct {
	ID           string    `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"not null" validate:"required,min=1,max=100"`
	Description  string    `json:"description" gorm:"type:text"`
	Provider     string    `json:"provider" gorm:"not null" validate:"required,oneof=openai anthropic mistral grok ollama"`
	Model        string    `json:"model" gorm:"not null" validate:"required"`
	SystemPrompt string    `json:"system_prompt" gorm:"type:text;not null" validate:"required"`
	Temperature  float32   `json:"temperature" gorm:"default:0.7" validate:"min=0,max=2"`
	MaxTokens    int       `json:"max_tokens" gorm:"default:1000" validate:"min=1,max=100000"`
	Config       JSON      `json:"config" gorm:"type:json"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Relationships
	Sessions []ChatSession `json:"sessions,omitempty" gorm:"foreignKey:AgentID"`
}

// BeforeCreate hook to generate UUID
func (a *Agent) BeforeCreate(tx *gorm.DB) error {
	if a.ID == "" {
		a.ID = uuid.New().String()
	}
	return nil
}

// CreateAgentRequest represents the request payload for creating an agent
type CreateAgentRequest struct {
	Name         string                 `json:"name" validate:"required,min=1,max=100"`
	Description  string                 `json:"description"`
	Provider     string                 `json:"provider" validate:"required,oneof=openai anthropic mistral grok ollama"`
	Model        string                 `json:"model" validate:"required"`
	SystemPrompt string                 `json:"system_prompt" validate:"required"`
	Temperature  *float32               `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxTokens    *int                   `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=100000"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// UpdateAgentRequest represents the request payload for updating an agent
type UpdateAgentRequest struct {
	Name         *string                `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
	Description  *string                `json:"description,omitempty"`
	Provider     *string                `json:"provider,omitempty" validate:"omitempty,oneof=openai anthropic mistral grok ollama"`
	Model        *string                `json:"model,omitempty"`
	SystemPrompt *string                `json:"system_prompt,omitempty"`
	Temperature  *float32               `json:"temperature,omitempty" validate:"omitempty,min=0,max=2"`
	MaxTokens    *int                   `json:"max_tokens,omitempty" validate:"omitempty,min=1,max=100000"`
	Config       map[string]interface{} `json:"config,omitempty"`
}

// ToAgent converts CreateAgentRequest to Agent
func (r *CreateAgentRequest) ToAgent() *Agent {
	agent := &Agent{
		Name:         r.Name,
		Description:  r.Description,
		Provider:     r.Provider,
		Model:        r.Model,
		SystemPrompt: r.SystemPrompt,
		Temperature:  0.7,
		MaxTokens:    1000,
		Config:       make(JSON),
	}

	if r.Temperature != nil {
		agent.Temperature = *r.Temperature
	}
	if r.MaxTokens != nil {
		agent.MaxTokens = *r.MaxTokens
	}
	if r.Config != nil {
		agent.Config = JSON(r.Config)
	}

	return agent
}

// UpdateFromRequest updates agent fields from UpdateAgentRequest
func (a *Agent) UpdateFromRequest(req *UpdateAgentRequest) {
	if req.Name != nil {
		a.Name = *req.Name
	}
	if req.Description != nil {
		a.Description = *req.Description
	}
	if req.Provider != nil {
		a.Provider = *req.Provider
	}
	if req.Model != nil {
		a.Model = *req.Model
	}
	if req.SystemPrompt != nil {
		a.SystemPrompt = *req.SystemPrompt
	}
	if req.Temperature != nil {
		a.Temperature = *req.Temperature
	}
	if req.MaxTokens != nil {
		a.MaxTokens = *req.MaxTokens
	}
	if req.Config != nil {
		a.Config = JSON(req.Config)
	}
}