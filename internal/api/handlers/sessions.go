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

// SessionHandler handles session-related requests
type SessionHandler struct {
	sessionRepo storage.SessionRepository
	agentRepo   storage.AgentRepository
	validator   *validator.Validate
}

// NewSessionHandler creates a new session handler
func NewSessionHandler(sessionRepo storage.SessionRepository, agentRepo storage.AgentRepository) *SessionHandler {
	return &SessionHandler{
		sessionRepo: sessionRepo,
		agentRepo:   agentRepo,
		validator:   validator.New(),
	}
}

// Create creates a new chat session for an agent
func (h *SessionHandler) Create(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Check if agent exists
	agent, err := h.agentRepo.GetByID(c.Request.Context(), agentID)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", agentID).Error("Failed to get agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

	var req models.CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Convert to session model
	session := req.ToSession(agentID)

	// Save to database
	if err := h.sessionRepo.Create(c.Request.Context(), session); err != nil {
		logrus.WithError(err).Error("Failed to create session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create session"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"session_id": session.ID,
		"agent_id":   agentID,
	}).Info("Session created successfully")

	c.JSON(http.StatusCreated, session)
}

// GetByID retrieves a session by ID
func (h *SessionHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	session, err := h.sessionRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("session_id", id).Error("Failed to get session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	c.JSON(http.StatusOK, session)
}

// Update updates an existing session
func (h *SessionHandler) Update(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req models.UpdateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Get existing session
	session, err := h.sessionRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("session_id", id).Error("Failed to get session for update")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Update session fields
	session.UpdateFromRequest(&req)

	// Save updated session
	if err := h.sessionRepo.Update(c.Request.Context(), session); err != nil {
		logrus.WithError(err).WithField("session_id", id).Error("Failed to update session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update session"})
		return
	}

	logrus.WithField("session_id", id).Info("Session updated successfully")
	c.JSON(http.StatusOK, session)
}

// Delete deletes a session
func (h *SessionHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Check if session exists
	session, err := h.sessionRepo.GetByID(c.Request.Context(), id)
	if err != nil {
		logrus.WithError(err).WithField("session_id", id).Error("Failed to get session for deletion")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve session"})
		return
	}

	if session == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Delete session (this will also delete associated messages due to cascade)
	if err := h.sessionRepo.Delete(c.Request.Context(), id); err != nil {
		logrus.WithError(err).WithField("session_id", id).Error("Failed to delete session")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete session"})
		return
	}

	logrus.WithField("session_id", id).Info("Session deleted successfully")
	c.JSON(http.StatusNoContent, nil)
}

// ListByAgent retrieves a paginated list of sessions for an agent
func (h *SessionHandler) ListByAgent(c *gin.Context) {
	agentID := c.Param("id")
	if agentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Agent ID is required"})
		return
	}

	// Check if agent exists
	agent, err := h.agentRepo.GetByID(c.Request.Context(), agentID)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", agentID).Error("Failed to get agent")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve agent"})
		return
	}

	if agent == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Agent not found"})
		return
	}

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

	// Get sessions from database
	sessions, total, err := h.sessionRepo.ListByAgentID(c.Request.Context(), agentID, pageSize, offset)
	if err != nil {
		logrus.WithError(err).WithField("agent_id", agentID).Error("Failed to list sessions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve sessions"})
		return
	}

	// Calculate pagination info
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	hasMore := page < int(totalPages)

	response := gin.H{
		"sessions":    sessions,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": totalPages,
		"has_more":    hasMore,
	}

	c.JSON(http.StatusOK, response)
}