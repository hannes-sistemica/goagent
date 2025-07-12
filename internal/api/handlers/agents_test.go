package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"agent-server/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockAgentRepository is a mock implementation of AgentRepository
type MockAgentRepository struct {
	mock.Mock
}

func (m *MockAgentRepository) Create(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) GetByID(ctx context.Context, id string) (*models.Agent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Agent), args.Error(1)
}

func (m *MockAgentRepository) Update(ctx context.Context, agent *models.Agent) error {
	args := m.Called(ctx, agent)
	return args.Error(0)
}

func (m *MockAgentRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentRepository) List(ctx context.Context, limit, offset int) ([]*models.Agent, int64, error) {
	args := m.Called(ctx, limit, offset)
	return args.Get(0).([]*models.Agent), args.Get(1).(int64), args.Error(2)
}

func TestAgentHandler_Create(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    models.CreateAgentRequest
		setupMock      func(*MockAgentRepository)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name: "successful creation",
			requestBody: models.CreateAgentRequest{
				Name:         "Test Agent",
				Description:  "A test agent",
				Provider:     "ollama",
				Model:        "llama2",
				SystemPrompt: "You are helpful",
			},
			setupMock: func(repo *MockAgentRepository) {
				repo.On("Create", mock.Anything, mock.AnythingOfType("*models.Agent")).Run(func(args mock.Arguments) {
					agent := args.Get(1).(*models.Agent)
					agent.ID = "test-agent-id" // Set ID for testing
				}).Return(nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var agent models.Agent
				err := json.Unmarshal(w.Body.Bytes(), &agent)
				require.NoError(t, err)
				assert.Equal(t, "Test Agent", agent.Name)
				assert.Equal(t, "ollama", agent.Provider)
				assert.Equal(t, "llama2", agent.Model)
				assert.Equal(t, "test-agent-id", agent.ID)
			},
		},
		{
			name: "invalid provider",
			requestBody: models.CreateAgentRequest{
				Name:         "Test Agent",
				Provider:     "invalid",
				Model:        "test",
				SystemPrompt: "You are helpful",
			},
			setupMock:      func(repo *MockAgentRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, "Validation failed", response["error"])
			},
		},
		{
			name: "missing required fields",
			requestBody: models.CreateAgentRequest{
				Name: "Test Agent",
				// Missing Provider, Model, SystemPrompt
			},
			setupMock:      func(repo *MockAgentRepository) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Contains(t, response, "error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockAgentRepository)
			tt.setupMock(mockRepo)

			handler := NewAgentHandler(mockRepo)

			// Create request
			body, _ := json.Marshal(tt.requestBody)
			req := httptest.NewRequest("POST", "/agents", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req

			// Execute
			handler.Create(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAgentHandler_GetByID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		agentID        string
		setupMock      func(*MockAgentRepository)
		expectedStatus int
		checkResponse  func(*testing.T, *httptest.ResponseRecorder)
	}{
		{
			name:    "successful get",
			agentID: "test-id",
			setupMock: func(repo *MockAgentRepository) {
				agent := &models.Agent{
					ID:           "test-id",
					Name:         "Test Agent",
					Provider:     "ollama",
					Model:        "llama2",
					SystemPrompt: "You are helpful",
				}
				repo.On("GetByID", mock.Anything, "test-id").Return(agent, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var agent models.Agent
				err := json.Unmarshal(w.Body.Bytes(), &agent)
				require.NoError(t, err)
				assert.Equal(t, "test-id", agent.ID)
				assert.Equal(t, "Test Agent", agent.Name)
			},
		},
		{
			name:    "agent not found",
			agentID: "nonexistent",
			setupMock: func(repo *MockAgentRepository) {
				repo.On("GetByID", mock.Anything, "nonexistent").Return(nil, nil)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, "Agent not found", response["error"])
			},
		},
		{
			name:    "empty agent ID",
			agentID: "",
			setupMock: func(repo *MockAgentRepository) {
				// No mock setup needed as validation happens before repo call
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, w *httptest.ResponseRecorder) {
				var response map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &response)
				assert.Equal(t, "Agent ID is required", response["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mockRepo := new(MockAgentRepository)
			tt.setupMock(mockRepo)

			handler := NewAgentHandler(mockRepo)

			// Create request
			req := httptest.NewRequest("GET", "/agents/"+tt.agentID, nil)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = req
			c.Params = gin.Params{{Key: "id", Value: tt.agentID}}

			// Execute
			handler.GetByID(c)

			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			tt.checkResponse(t, w)

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestAgentHandler_List(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockRepo := new(MockAgentRepository)
	agents := []*models.Agent{
		{
			ID:       "agent-1",
			Name:     "Agent 1",
			Provider: "ollama",
			Model:    "llama2",
		},
		{
			ID:       "agent-2",
			Name:     "Agent 2",
			Provider: "ollama",
			Model:    "codellama",
		},
	}

	mockRepo.On("List", mock.Anything, 20, 0).Return(agents, int64(2), nil)

	handler := NewAgentHandler(mockRepo)

	// Create request
	req := httptest.NewRequest("GET", "/agents", nil)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// Execute
	handler.List(c)

	// Assert
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	assert.Equal(t, float64(2), response["total_count"])
	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(20), response["page_size"])
	assert.False(t, response["has_more"].(bool))

	agentsData := response["agents"].([]interface{})
	assert.Len(t, agentsData, 2)

	mockRepo.AssertExpectations(t)
}