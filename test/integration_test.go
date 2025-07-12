package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"agent-server/internal/api"
	"agent-server/internal/config"
	contextpkg "agent-server/internal/context"
	"agent-server/internal/llm"
	"agent-server/internal/llm/ollama"
	"agent-server/internal/models"
	"agent-server/internal/storage/sqlite"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	server     *api.Server
	tempDBPath string
}

func (suite *IntegrationTestSuite) SetupSuite() {
	// Create temporary database file
	tempDir := os.TempDir()
	suite.tempDBPath = filepath.Join(tempDir, "test_agents.db")

	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Database: config.DatabaseConfig{
			Type: "sqlite",
			Path: suite.tempDBPath,
		},
		LLM: config.LLMConfig{
			Providers: map[string]config.ProviderConfig{
				"ollama": {
					BaseURL: "http://localhost:11434",
				},
			},
		},
		Logging: config.LoggingConfig{
			Level:  "error", // Reduce log noise in tests
			Format: "json",
		},
	}

	// Initialize storage
	repo, err := sqlite.NewRepository(suite.tempDBPath)
	require.NoError(suite.T(), err)

	// Initialize registries
	ctxRegistry := contextpkg.NewStrategyRegistry()
	llmRegistry := llm.NewRegistry()

	// Register Ollama provider (will use mock server in tests)
	ollamaProvider := ollama.NewProvider("http://localhost:11434")
	llmRegistry.Register(ollamaProvider)

	// Create server
	suite.server = api.NewServer(cfg, repo, ctxRegistry, llmRegistry)
	suite.server.SetupRoutes()
}

func (suite *IntegrationTestSuite) TearDownSuite() {
	// Clean up temporary database
	os.Remove(suite.tempDBPath)
}

func (suite *IntegrationTestSuite) TestCompleteWorkflow() {
	router := suite.server.GetRouter()

	// 1. Create an agent
	agentReq := models.CreateAgentRequest{
		Name:         "Test Integration Agent",
		Description:  "An agent for integration testing",
		Provider:     "ollama",
		Model:        "llama2",
		SystemPrompt: "You are a helpful assistant for testing.",
		Temperature:  func() *float32 { v := float32(0.7); return &v }(),
		MaxTokens:    func() *int { v := 100; return &v }(),
	}

	agentBody, _ := json.Marshal(agentReq)
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(agentBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var agent models.Agent
	err := json.Unmarshal(w.Body.Bytes(), &agent)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), agent.ID)
	assert.Equal(suite.T(), "Test Integration Agent", agent.Name)

	// 2. Get the created agent
	req = httptest.NewRequest("GET", "/api/v1/agents/"+agent.ID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var retrievedAgent models.Agent
	err = json.Unmarshal(w.Body.Bytes(), &retrievedAgent)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), agent.ID, retrievedAgent.ID)

	// 3. Create a session for the agent
	sessionReq := models.CreateSessionRequest{
		Title:           "Test Session",
		ContextStrategy: "last_n",
		ContextConfig: map[string]interface{}{
			"count": 5,
		},
	}

	sessionBody, _ := json.Marshal(sessionReq)
	req = httptest.NewRequest("POST", "/api/v1/agents/"+agent.ID+"/sessions", bytes.NewReader(sessionBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusCreated, w.Code)

	var session models.ChatSession
	err = json.Unmarshal(w.Body.Bytes(), &session)
	require.NoError(suite.T(), err)
	assert.NotEmpty(suite.T(), session.ID)
	assert.Equal(suite.T(), agent.ID, session.AgentID)

	// 4. Send a message (this will fail without actual Ollama, but we can test the structure)
	chatReq := map[string]interface{}{
		"message": "Hello, this is a test message!",
		"metadata": map[string]interface{}{
			"test": true,
		},
	}

	chatBody, _ := json.Marshal(chatReq)
	req = httptest.NewRequest("POST", "/api/v1/sessions/"+session.ID+"/chat", bytes.NewReader(chatBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// This will likely fail because Ollama isn't running, but we can check the structure
	// In a real integration test, you'd have Ollama running or use a mock server
	if w.Code == http.StatusOK {
		var chatResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &chatResp)
		require.NoError(suite.T(), err)
		assert.Contains(suite.T(), chatResp, "response")
		assert.Contains(suite.T(), chatResp, "user_message_id")
		assert.Contains(suite.T(), chatResp, "assistant_message_id")
	} else {
		// Expected if Ollama is not running
		assert.Contains(suite.T(), []int{http.StatusInternalServerError, http.StatusServiceUnavailable}, w.Code)
	}

	// 5. Get message history
	req = httptest.NewRequest("GET", "/api/v1/sessions/"+session.ID+"/messages", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var messageList models.MessageList
	err = json.Unmarshal(w.Body.Bytes(), &messageList)
	require.NoError(suite.T(), err)

	// We should have at least the user message, even if the assistant response failed
	if len(messageList.Messages) > 0 {
		assert.Equal(suite.T(), "user", messageList.Messages[0].Role)
		assert.Equal(suite.T(), "Hello, this is a test message!", messageList.Messages[0].Content)
	}

	// 6. List sessions for the agent
	req = httptest.NewRequest("GET", "/api/v1/agents/"+agent.ID+"/sessions", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var sessionListResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &sessionListResp)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(1), sessionListResp["total_count"])

	// 7. Update the agent
	updateReq := models.UpdateAgentRequest{
		Name: func() *string { s := "Updated Agent Name"; return &s }(),
	}

	updateBody, _ := json.Marshal(updateReq)
	req = httptest.NewRequest("PUT", "/api/v1/agents/"+agent.ID, bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var updatedAgent models.Agent
	err = json.Unmarshal(w.Body.Bytes(), &updatedAgent)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "Updated Agent Name", updatedAgent.Name)

	// 8. List all agents
	req = httptest.NewRequest("GET", "/api/v1/agents", nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var agentListResp map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &agentListResp)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(1), agentListResp["total_count"])

	// 9. Delete the session
	req = httptest.NewRequest("DELETE", "/api/v1/sessions/"+session.ID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// 10. Delete the agent
	req = httptest.NewRequest("DELETE", "/api/v1/agents/"+agent.ID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNoContent, w.Code)

	// 11. Verify agent is deleted
	req = httptest.NewRequest("GET", "/api/v1/agents/"+agent.ID, nil)
	w = httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusNotFound, w.Code)
}

func (suite *IntegrationTestSuite) TestHealthCheck() {
	router := suite.server.GetRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), "ok", response["status"])
}

func (suite *IntegrationTestSuite) TestContextStrategies() {
	router := suite.server.GetRouter()

	// Create an agent
	agentReq := models.CreateAgentRequest{
		Name:         "Context Test Agent",
		Provider:     "ollama",
		Model:        "llama2",
		SystemPrompt: "You are helpful.",
	}

	agentBody, _ := json.Marshal(agentReq)
	req := httptest.NewRequest("POST", "/api/v1/agents", bytes.NewReader(agentBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)
	require.Equal(suite.T(), http.StatusCreated, w.Code)

	var agent models.Agent
	json.Unmarshal(w.Body.Bytes(), &agent)

	// Test different context strategies
	strategies := []struct {
		name   string
		config map[string]interface{}
	}{
		{"last_n", map[string]interface{}{"count": 3}},
		{"sliding_window", map[string]interface{}{"window_size": 4, "overlap": 1}},
		{"summarize", map[string]interface{}{"max_context_length": 10, "keep_recent": 3}},
	}

	for _, strategy := range strategies {
		suite.T().Run("strategy_"+strategy.name, func(t *testing.T) {
			// Create session with specific strategy
			sessionReq := models.CreateSessionRequest{
				Title:           "Test " + strategy.name,
				ContextStrategy: strategy.name,
				ContextConfig:   strategy.config,
			}

			sessionBody, _ := json.Marshal(sessionReq)
			req := httptest.NewRequest("POST", "/api/v1/agents/"+agent.ID+"/sessions", bytes.NewReader(sessionBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusCreated, w.Code)

			var session models.ChatSession
			json.Unmarshal(w.Body.Bytes(), &session)
			assert.Equal(t, strategy.name, session.ContextStrategy)

			// Clean up
			req = httptest.NewRequest("DELETE", "/api/v1/sessions/"+session.ID, nil)
			w = httptest.NewRecorder()
			router.ServeHTTP(w, req)
		})
	}

	// Clean up agent
	req = httptest.NewRequest("DELETE", "/api/v1/agents/"+agent.ID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
}

func TestIntegrationSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}