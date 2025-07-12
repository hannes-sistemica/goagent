package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateAgentRequest_ToAgent(t *testing.T) {
	tests := []struct {
		name     string
		request  CreateAgentRequest
		expected Agent
	}{
		{
			name: "basic agent creation",
			request: CreateAgentRequest{
				Name:         "Test Agent",
				Description:  "A test agent",
				Provider:     "openai",
				Model:        "gpt-4",
				SystemPrompt: "You are a helpful assistant",
			},
			expected: Agent{
				Name:         "Test Agent",
				Description:  "A test agent",
				Provider:     "openai",
				Model:        "gpt-4",
				SystemPrompt: "You are a helpful assistant",
				Temperature:  0.7,
				MaxTokens:    1000,
				Config:       make(JSON),
			},
		},
		{
			name: "agent with custom parameters",
			request: CreateAgentRequest{
				Name:         "Custom Agent",
				Provider:     "ollama",
				Model:        "llama2",
				SystemPrompt: "Custom prompt",
				Temperature:  func() *float32 { v := float32(0.5); return &v }(),
				MaxTokens:    func() *int { v := 2000; return &v }(),
				Config: map[string]interface{}{
					"custom_key": "custom_value",
				},
			},
			expected: Agent{
				Name:         "Custom Agent",
				Provider:     "ollama",
				Model:        "llama2",
				SystemPrompt: "Custom prompt",
				Temperature:  0.5,
				MaxTokens:    2000,
				Config: JSON{
					"custom_key": "custom_value",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agent := tt.request.ToAgent()
			assert.Equal(t, tt.expected.Name, agent.Name)
			assert.Equal(t, tt.expected.Description, agent.Description)
			assert.Equal(t, tt.expected.Provider, agent.Provider)
			assert.Equal(t, tt.expected.Model, agent.Model)
			assert.Equal(t, tt.expected.SystemPrompt, agent.SystemPrompt)
			assert.Equal(t, tt.expected.Temperature, agent.Temperature)
			assert.Equal(t, tt.expected.MaxTokens, agent.MaxTokens)
			assert.Equal(t, tt.expected.Config, agent.Config)
		})
	}
}

func TestAgent_UpdateFromRequest(t *testing.T) {
	agent := &Agent{
		Name:         "Original Agent",
		Description:  "Original description",
		Provider:     "openai",
		Model:        "gpt-3.5-turbo",
		SystemPrompt: "Original prompt",
		Temperature:  0.7,
		MaxTokens:    1000,
		Config:       JSON{"key": "value"},
	}

	tests := []struct {
		name    string
		request UpdateAgentRequest
		check   func(t *testing.T, agent *Agent)
	}{
		{
			name: "update name only",
			request: UpdateAgentRequest{
				Name: func() *string { s := "Updated Name"; return &s }(),
			},
			check: func(t *testing.T, agent *Agent) {
				assert.Equal(t, "Updated Name", agent.Name)
				assert.Equal(t, "Original description", agent.Description)
			},
		},
		{
			name: "update multiple fields",
			request: UpdateAgentRequest{
				Name:        func() *string { s := "New Name"; return &s }(),
				Temperature: func() *float32 { v := float32(0.9); return &v }(),
				MaxTokens:   func() *int { v := 2000; return &v }(),
			},
			check: func(t *testing.T, agent *Agent) {
				assert.Equal(t, "New Name", agent.Name)
				assert.Equal(t, float32(0.9), agent.Temperature)
				assert.Equal(t, 2000, agent.MaxTokens)
				assert.Equal(t, "openai", agent.Provider) // unchanged
			},
		},
		{
			name: "update config",
			request: UpdateAgentRequest{
				Config: map[string]interface{}{
					"new_key": "new_value",
					"number":  42,
				},
			},
			check: func(t *testing.T, agent *Agent) {
				expected := JSON{
					"new_key": "new_value",
					"number":  42,
				}
				assert.Equal(t, expected, agent.Config)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy of the original agent
			testAgent := *agent
			testAgent.Config = make(JSON)
			for k, v := range agent.Config {
				testAgent.Config[k] = v
			}

			testAgent.UpdateFromRequest(&tt.request)
			tt.check(t, &testAgent)
		})
	}
}

func TestJSON_ValueAndScan(t *testing.T) {
	tests := []struct {
		name string
		json JSON
	}{
		{
			name: "empty json",
			json: JSON{},
		},
		{
			name: "simple json",
			json: JSON{
				"key":    "value",
				"number": float64(42), // JSON unmarshaling converts numbers to float64
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Value method
			value, err := tt.json.Value()
			require.NoError(t, err)

			// Test Scan method
			var scanned JSON
			err = scanned.Scan(value)
			require.NoError(t, err)
			assert.Equal(t, tt.json, scanned)
		})
	}
}

func TestJSON_ScanNil(t *testing.T) {
	var j JSON
	err := j.Scan(nil)
	require.NoError(t, err)
	assert.Equal(t, JSON{}, j)
}