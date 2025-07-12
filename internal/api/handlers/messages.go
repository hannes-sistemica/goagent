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

// MessageHandler handles message-related requests
type MessageHandler struct {
	repo      storage.MessageRepository
	validator *validator.Validate
}

// NewMessageHandler creates a new message handler
func NewMessageHandler(repo storage.MessageRepository) *MessageHandler {
	return &MessageHandler{
		repo:      repo,
		validator: validator.New(),
	}
}

// Create creates a new message in a session
func (h *MessageHandler) Create(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	var req models.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate request
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Validation failed", "details": err.Error()})
		return
	}

	// Convert to message model
	message := req.ToMessage(sessionID)

	// Save to database
	if err := h.repo.Create(c.Request.Context(), message); err != nil {
		logrus.WithError(err).Error("Failed to create message")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create message"})
		return
	}

	logrus.WithFields(logrus.Fields{
		"message_id": message.ID,
		"session_id": sessionID,
		"role":       message.Role,
	}).Info("Message created successfully")

	c.JSON(http.StatusCreated, message)
}

// ListBySession retrieves a paginated list of messages for a session
func (h *MessageHandler) ListBySession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Parse pagination parameters
	page := 1
	pageSize := 50

	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if parsed, err := strconv.Atoi(ps); err == nil && parsed > 0 && parsed <= 200 {
			pageSize = parsed
		}
	}

	offset := (page - 1) * pageSize

	// Get messages from database
	messages, total, err := h.repo.ListBySessionID(c.Request.Context(), sessionID, pageSize, offset)
	if err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to list messages")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve messages"})
		return
	}

	// Calculate pagination info
	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	hasMore := page < int(totalPages)

	response := models.MessageList{
		Messages:   make([]models.Message, len(messages)),
		TotalCount: total,
		Page:       page,
		PageSize:   pageSize,
		HasMore:    hasMore,
	}

	// Convert pointer slice to value slice
	for i, msg := range messages {
		response.Messages[i] = *msg
	}

	c.JSON(http.StatusOK, response)
}

// DeleteBySession deletes all messages in a session
func (h *MessageHandler) DeleteBySession(c *gin.Context) {
	sessionID := c.Param("id")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Session ID is required"})
		return
	}

	// Delete all messages for the session
	if err := h.repo.DeleteBySessionID(c.Request.Context(), sessionID); err != nil {
		logrus.WithError(err).WithField("session_id", sessionID).Error("Failed to delete messages")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete messages"})
		return
	}

	logrus.WithField("session_id", sessionID).Info("Messages deleted successfully")
	c.JSON(http.StatusNoContent, nil)
}