package handlers

import (
	"net/http"
	"strconv"

	"agent-server/internal/models"
	"agent-server/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

// AgentHandler handles agent-related requests
type AgentHandler struct {
	repo      storage.AgentRepository
	validator *validator.Validate
}

// NewAgentHandler creates a new agent handler
func NewAgentHandler(repo storage.AgentRepository) *AgentHandler {
	return &AgentHandler{
		repo:      repo,
		validator: validator.New(),
	}
}

// Create creates a new agent
func (h *AgentHandler) Create(c *gin.Context) {
	var req models.CreateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Convert to agent model
	agent := req.ToAgent()

	// Save to database
	if err := h.repo.Create(c.Request.Context(), agent); err != nil {
		logrus.WithError(err).Error("Failed to create agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create agent"})
		return
	}

	logrus.WithField("agent_id", agent.ID).Info("Agent created successfully")
	c.JSON(http.StatusCreated, agent)
}

// GetByID retrieves an agent by ID
func (h *AgentHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	agent, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", id).Error("Failed to get agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	c.JSON(http.StatusOK, agent)
}

// Update updates an existing agent
func (h *AgentHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	var req models.UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Get existing agent
	agent, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", id).Error("Failed to get agent for update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Update agent fields
	agent.UpdateFromRequest(&req)

	// Save updated agent
	if err := h.repo.Update(c.Request.Context(), agent); err != nil {
		logrus.WithError(err).WithField("agent_id", id).Error("Failed to update agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update agent"})
		return
	}

	logrus.WithField("agent_id", id).Info("Agent updated successfully")
	c.JSON(http.StatusOK, agent)
}

// Delete deletes an agent
func (h *AgentHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Check if agent exists
	agent, err := h.repo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", id).Error("Failed to get agent for deletion")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	// Delete agent
	if err := h.repo.Delete(c.Request.Context(), id); err != nil {
		logrus.WithError(err).WithField("agent_id", id).Error("Failed to delete agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete agent"})
		return
	}

	logrus.WithField("agent_id", id).Info("Agent deleted successfully")
	c.JSON(http.StatusNoContent, nil)
}

// List retrieves a paginated list of agents
func (h *AgentHandler) List(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	pageSize := 20

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 100 {
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	// Get agents from database
	agents, total, err := h.repo.List(c.Request.Context(), pageSize, offset)
	if err != nil {
		logrus.WithError(err).Error("Failed to list agents")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agents"})
		return
	}

	// Calculate pagination info
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	hasMore := page < int(totalPages)

	response := gin.H{
		"agents":      agents,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"has_more":    hasMore,
	}

	c.JSON(http.StatusOK, response)
}