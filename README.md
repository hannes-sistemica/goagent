# Agent Server

A production-ready AI agent management system built in Go that provides REST APIs for creating, managing, and chatting with AI agents across multiple LLM providers.

## Features

- **Multi-Provider Support**: OpenAI, Anthropic, Mistral, Grok, and Ollama
- **Agent Management**: Create, update, delete, and list AI agents
- **Session Management**: Persistent chat sessions with configurable context strategies
- **Tool Calling**: Extensible tool system with built-in tools and external API integration
- **MCP Integration**: Built-in support for Model Context Protocol (MCP) and OpenMCP
- **Streaming Support**: Real-time streaming responses via Server-Sent Events (SSE)
- **Context Strategies**: Pluggable context management (last_n, sliding_window, summarize)
- **SQLite Storage**: Lightweight, embedded database for persistence
- **RESTful API**: Clean, well-documented REST endpoints
- **Production Ready**: Structured logging, error handling, and configuration management

## Quick Start

### Prerequisites

- Go 1.21 or higher
- Make (optional, for using Makefile commands)
- Ollama (for local LLM testing) - Install from [ollama.ai](https://ollama.ai)

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd agent-server
```

2. Install dependencies:
```bash
go mod download
```

3. Set up configuration:
```bash
cp configs/config.sample.yaml configs/config.yaml
# Edit configs/config.yaml with your API keys
```

4. Start the server:
```bash
make dev
# or
go run cmd/server/main.go -config configs/config.yaml
```

The server will start on `http://localhost:8081`

### Using Ollama (Recommended for Testing)

1. Install Ollama from [ollama.ai](https://ollama.ai)
2. Pull a model:
```bash
ollama pull llama3.2:3b
```
3. Start Ollama:
```bash
ollama serve
```
4. The agent server will automatically detect Ollama on `http://localhost:11434`

## Configuration

Copy `configs/config.sample.yaml` to `configs/config.yaml` and configure:

```yaml
server:
  host: "0.0.0.0"  # Server bind address
  port: 8081       # Server port

database:
  type: sqlite
  path: "./data/agents.db"  # SQLite database file

llm:
  providers:
    openai:
      api_key: "sk-your-openai-api-key-here"
      base_url: "https://api.openai.com/v1"
    # ... other providers
    ollama:
      base_url: "http://localhost:11434"  # No API key needed

logging:
  level: info      # debug, info, warn, error
  format: json     # json or text
```

### Environment Variables

You can override configuration with environment variables:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`
- `MISTRAL_API_KEY`
- `GROK_API_KEY`

## API Usage

### Complete REST API Reference

#### Health Check
```bash
# Check server health
curl http://localhost:8081/health

# API health check (returns JSON)
curl http://localhost:8081/api/v1/health
```

#### Agent Management

##### Create an Agent
```bash
# Create a new agent (returns agent with ID)
curl -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "My Assistant",
    "description": "A helpful coding assistant",
    "provider": "ollama",
    "model": "llama3.2:3b",
    "system_prompt": "You are a helpful coding assistant.",
    "temperature": 0.7,
    "max_tokens": 1000
  }'

# Response example:
# {
#   "id": "550e8400-e29b-41d4-a716-446655440000",
#   "name": "My Assistant",
#   "provider": "ollama",
#   "model": "llama3.2:3b",
#   "created_at": "2025-07-12T10:00:00Z",
#   "updated_at": "2025-07-12T10:00:00Z"
# }
```

##### List All Agents
```bash
# Get all agents with pagination
curl "http://localhost:8081/api/v1/agents?page=1&limit=10"

# Filter by provider
curl "http://localhost:8081/api/v1/agents?provider=ollama"

# Sort by creation date
curl "http://localhost:8081/api/v1/agents?sort=created_at&order=desc"
```

##### Get Agent Details
```bash
# Get specific agent by ID
AGENT_ID="550e8400-e29b-41d4-a716-446655440000"
curl "http://localhost:8081/api/v1/agents/$AGENT_ID"
```

##### Update Agent
```bash
# Update agent configuration
curl -X PUT "http://localhost:8081/api/v1/agents/$AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Assistant",
    "temperature": 0.5,
    "system_prompt": "You are a helpful and concise coding assistant."
  }'
```

##### Delete Agent
```bash
# Delete an agent (also deletes all associated sessions)
curl -X DELETE "http://localhost:8081/api/v1/agents/$AGENT_ID"
```

#### Session Management

##### Create a Session
```bash
# Create a new chat session for an agent
SESSION_RESPONSE=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Coding Help Session",
    "context_strategy": "last_n",
    "context_config": {
      "count": 10
    }
  }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')
echo "Created session: $SESSION_ID"
```

##### List Agent Sessions
```bash
# Get all sessions for an agent
curl "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions"

# With pagination
curl "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions?page=1&limit=20"
```

##### Get Session Details
```bash
# Get session information including message count
curl "http://localhost:8081/api/v1/sessions/$SESSION_ID"
```

##### Update Session
```bash
# Update session title or settings
curl -X PUT "http://localhost:8081/api/v1/sessions/$SESSION_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Python Development Help",
    "context_config": {
      "count": 20
    }
  }'
```

##### Delete Session
```bash
# Delete a session (keeps agent)
curl -X DELETE "http://localhost:8081/api/v1/sessions/$SESSION_ID"
```

#### Chat Operations

##### Send a Simple Chat Message
```bash
# Basic chat without tools
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How do I reverse a string in Python?"
  }'
```

##### Continue a Conversation
```bash
# First message
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "I need help with Python lists"
  }'

# Follow-up message (context is maintained)
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How do I sort them in reverse order?"
  }'
```

##### Enhanced Chat with Tools
```bash
# Chat with specific tools enabled
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Calculate the compound interest on $1000 at 5% for 3 years",
    "tools": ["calculator"],
    "tool_choice": "auto"
  }'

# Force tool usage
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Store this information: I prefer morning meetings at 9 AM",
    "tools": ["memory"],
    "tool_choice": "required"
  }'
```

##### Stream Chat Response
```bash
# Real-time streaming response
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/stream" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": "Write a Python function to calculate fibonacci numbers with detailed explanation"
  }'

# With tools in streaming mode
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/stream" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{
    "message": "Calculate 15 * 23 and explain the result",
    "tools": ["calculator"]
  }'
```

#### Message History

##### Get Session Messages
```bash
# Get all messages in a session
curl "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages"

# With pagination
curl "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages?page=1&limit=50"

# Get only last N messages
curl "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages?last=10"
```

##### Get Specific Message
```bash
# Get message details including tool calls
MESSAGE_ID="msg-123456"
curl "http://localhost:8081/api/v1/messages/$MESSAGE_ID"
```

##### Search Messages
```bash
# Search across all sessions for an agent
curl "http://localhost:8081/api/v1/agents/$AGENT_ID/messages/search?q=python+functions"

# Search within a session
curl "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages/search?q=reverse+string"
```

#### Working with IDs

##### Extract IDs from Responses
```bash
# Create agent and extract ID
AGENT_ID=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Agent",
    "provider": "ollama",
    "model": "llama3.2:3b"
  }' | jq -r '.id')

# Create session and extract ID
SESSION_ID=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{"title": "Test Session"}' | jq -r '.id')

# Send message and extract response
RESPONSE=$(curl -s -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello"}' | jq -r '.response')
```

##### Batch Operations
```bash
# Create multiple agents
for i in {1..3}; do
  curl -X POST http://localhost:8081/api/v1/agents \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"Agent $i\",
      \"provider\": \"ollama\",
      \"model\": \"llama3.2:3b\"
    }"
done

# Delete multiple sessions
for session_id in $(curl -s "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" | jq -r '.sessions[].id'); do
  curl -X DELETE "http://localhost:8081/api/v1/sessions/$session_id"
done
```

### Complete Workflow Example

```bash
#!/bin/bash
# Complete example: Create agent, start session, have conversation

# 1. Create an agent
echo "Creating agent..."
AGENT_RESPONSE=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Python Helper",
    "provider": "ollama",
    "model": "llama3.2:3b",
    "system_prompt": "You are a Python programming expert. Be concise and provide code examples.",
    "temperature": 0.7
  }')

AGENT_ID=$(echo $AGENT_RESPONSE | jq -r '.id')
echo "Created agent: $AGENT_ID"

# 2. Create a session
echo "Creating session..."
SESSION_RESPONSE=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Learning Python",
    "context_strategy": "last_n",
    "context_config": {"count": 10}
  }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')
echo "Created session: $SESSION_ID"

# 3. Have a conversation
echo "Starting conversation..."

# First question
curl -s -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "How do I read a file in Python?"
  }' | jq -r '.response'

# Follow-up question
curl -s -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What about reading it line by line?"
  }' | jq -r '.response'

# Use tools
curl -s -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Calculate how many bytes are in 5 MB",
    "tools": ["calculator"],
    "tool_choice": "auto"
  }' | jq -r '.response'

# 4. Clean up (optional)
# curl -X DELETE "http://localhost:8081/api/v1/sessions/$SESSION_ID"
# curl -X DELETE "http://localhost:8081/api/v1/agents/$AGENT_ID"
```

### Error Handling

```bash
# Check for errors in responses
RESPONSE=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{"name": "Test"}')

if echo "$RESPONSE" | jq -e '.error' > /dev/null; then
  echo "Error: $(echo "$RESPONSE" | jq -r '.error')"
else
  echo "Success: $(echo "$RESPONSE" | jq -r '.id')"
fi
```

### Advanced API Examples

#### Resume a Previous Session
```bash
# List all sessions to find the one to resume
SESSIONS=$(curl -s "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions")
echo "Available sessions:"
echo "$SESSIONS" | jq -r '.sessions[] | "\(.id) - \(.title) (messages: \(.message_count))"'

# Resume a specific session
SESSION_ID="existing-session-id"
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Let'\''s continue our previous discussion"
  }'
```

#### Export Conversation History
```bash
# Get all messages from a session as JSON
curl -s "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages" | jq '.' > conversation.json

# Export as readable text
curl -s "http://localhost:8081/api/v1/sessions/$SESSION_ID/messages" | \
  jq -r '.messages[] | "[\(.role | ascii_upcase)]: \(.content)"' > conversation.txt
```

#### Multi-Turn Conversation with Memory
```bash
# Create a personal assistant that remembers across sessions
AGENT_ID=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Personal Memory Assistant",
    "provider": "ollama",
    "model": "mistral-small3.1:latest",
    "system_prompt": "You are a personal assistant that remembers everything about the user. Always use the memory tool to store and recall information."
  }' | jq -r '.id')

# Session 1: Store information
SESSION1=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{"title": "Initial Meeting"}' | jq -r '.id')

curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION1/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Hi! I'\''m John, I work at Apple as a iOS developer. I love Swift and have 2 cats named Pixel and Byte.",
    "tools": ["memory"]
  }'

# Session 2: Recall information (different session!)
SESSION2=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{"title": "Follow-up Meeting"}' | jq -r '.id')

curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION2/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Do you remember anything about me or my pets?",
    "tools": ["memory"]
  }'
```

#### Using cURL with Variables and Scripts
```bash
#!/bin/bash
# reusable-chat.sh - Reusable chat script

BASE_URL="http://localhost:8081/api/v1"
AGENT_ID="${1:-$AGENT_ID}"
SESSION_ID="${2:-$SESSION_ID}"

# Function to send a message
send_message() {
    local message="$1"
    local tools="${2:-[]}"
    
    curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat/enhanced" \
        -H "Content-Type: application/json" \
        -d "{
            \"message\": \"$message\",
            \"tools\": $tools,
            \"tool_choice\": \"auto\"
        }" | jq -r '.response'
}

# Usage
send_message "What is 25 * 4?" '["calculator"]'
send_message "Remember that my favorite number is 100" '["memory"]'
send_message "What is my favorite number?" '["memory"]'
```

#### Monitoring Active Sessions
```bash
# Get all active agents with session counts
curl -s "http://localhost:8081/api/v1/agents" | \
  jq -r '.agents[] | "\(.name) (\(.id)): \(.session_count) active sessions"'

# Get session statistics
for agent_id in $(curl -s "http://localhost:8081/api/v1/agents" | jq -r '.agents[].id'); do
    echo "Agent: $agent_id"
    curl -s "http://localhost:8081/api/v1/agents/$agent_id/sessions" | \
      jq -r '.sessions[] | "  - \(.title): \(.message_count) messages, last activity: \(.last_activity)"'
done
```

## Context Strategies

The system supports pluggable context strategies for managing chat history:

### Last N Messages (`last_n`)
Keeps the last N messages in context.
```json
{
  "context_strategy": "last_n",
  "context_config": {
    "count": 10
  }
}
```

### Sliding Window (`sliding_window`)
Maintains a sliding window of messages with overlap.
```json
{
  "context_strategy": "sliding_window", 
  "context_config": {
    "window_size": 5,
    "overlap": 2
  }
}
```

### Summarize (`summarize`)
Summarizes old messages to maintain context.
```json
{
  "context_strategy": "summarize",
  "context_config": {
    "max_context_length": 20,
    "keep_recent": 5
  }
}
```

## Tool Calling

The agent-server includes a comprehensive tool calling system that allows AI agents to interact with external APIs, services, and data sources. Tools enable agents to perform actions beyond text generation, such as calculations, web searches, API calls, and data persistence.

### How Tool Calling Works

1. **Tool Registration**: Tools are registered with the system and expose their capabilities through schemas
2. **Tool Discovery**: Agents can discover available tools and their parameters
3. **Automatic Invocation**: When agents need specific functionality, they can invoke tools automatically
4. **Result Integration**: Tool results are integrated back into the conversation context

### Built-in Tools

| Tool | Description | Key Features | Example Usage |
|------|-------------|--------------|---------------|
| `calculator` | Mathematical computations | Basic arithmetic (+, -, *, /, ^), sqrt(), abs() | `{"expression": "15 * 23 + sqrt(16)"}` |
| `memory` | Persistent memory storage | Store/recall user preferences, facts, and context | `{"action": "store", "topic": "user_info", "content": "..."}` |
| `http_get` | HTTP GET requests | Headers, query params, response parsing | `{"url": "https://api.example.com/data"}` |
| `http_post` | HTTP POST requests | JSON payloads, custom headers | `{"url": "...", "body": {...}}` |
| `web_scraper` | Web content extraction | Clean text extraction, metadata | `{"url": "https://example.com"}` |
| `text_processor` | Text manipulation | Transform, analyze, extract patterns | `{"text": "...", "operation": "..."}` |
| `json_processor` | JSON operations | Parse, transform, validate JSON | `{"json": {...}, "operation": "..."}` |
| `mcp_proxy` | Model Context Protocol | Connect to MCP servers, access resources | `{"server_url": "...", "action": "..."}` |
| `openmcp_proxy` | OpenMCP REST API | Discovery, tool execution, resources | `{"server_url": "...", "action": "..."}` |

### Memory Tool

The memory tool enables agents to overcome context limitations by storing and recalling information across conversations:

```bash
# Store a memory
curl -X POST http://localhost:8081/api/v1/tools/memory/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "action": "store",
      "topic": "user_preferences",
      "content": "User prefers concise technical explanations",
      "memory_type": "preference",
      "importance": 8,
      "tags": ["communication", "style"]
    }
  }'

# Recall memories by topic
curl -X POST http://localhost:8081/api/v1/tools/memory/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "action": "recall",
      "topic": "user_preferences",
      "limit": 5
    }
  }'

# Search memories
curl -X POST http://localhost:8081/api/v1/tools/memory/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "action": "search",
      "query": "technical",
      "memory_type": "preference"
    }
  }'
```

Memory actions: `store`, `recall`, `search`, `update`, `delete`, `stats`

### Tool Management API

```bash
# List all available tools
curl http://localhost:8081/api/v1/tools

# Get specific tool information
curl http://localhost:8081/api/v1/tools/calculator

# Test a tool
curl -X POST http://localhost:8081/api/v1/tools/calculator/test \
  -H "Content-Type: application/json" \
  -d '{"arguments": {"expression": "2 + 3 * 4"}}'

# Get tool schemas for LLM integration
curl http://localhost:8081/api/v1/tools/schemas?tools=calculator,http_get
```

### Session-Specific Tool Usage

```bash
# List tools available for a session
curl http://localhost:8081/api/v1/sessions/{session-id}/tools

# Get tool schema for session
curl http://localhost:8081/api/v1/sessions/{session-id}/tools/calculator/schema

# Test tool within session context
curl -X POST http://localhost:8081/api/v1/sessions/{session-id}/tools/calculator/test \
  -H "Content-Type: application/json" \
  -d '{"arguments": {"expression": "sqrt(16)"}}'
```

### Tool Configuration

Tools can be configured per agent or session:

```json
{
  "name": "Research Assistant",
  "provider": "ollama",
  "model": "llama3.2:3b",
  "system_prompt": "You are a research assistant with access to web tools.",
  "config": {
    "enabled_tools": ["web_scraper", "http_get", "text_processor"],
    "tool_choice": "auto",
    "max_tool_calls": 5
  }
}
```

### MCP Integration

The system includes built-in support for the Model Context Protocol (MCP):

```bash
# Initialize MCP connection
curl -X POST http://localhost:8081/api/v1/tools/mcp_proxy/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "server_url": "https://mcp-server.example.com",
      "action": "initialize"
    }
  }'

# List MCP resources
curl -X POST http://localhost:8081/api/v1/tools/mcp_proxy/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "server_url": "https://mcp-server.example.com",
      "action": "list_resources"
    }
  }'
```

### OpenMCP Integration

Connect to OpenMCP REST API servers:

```bash
# OpenMCP discovery
curl -X POST http://localhost:8081/api/v1/tools/openmcp_proxy/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "server_url": "https://openmcp-server.example.com",
      "action": "discovery"
    }
  }'

# Execute OpenMCP tool
curl -X POST http://localhost:8081/api/v1/tools/openmcp_proxy/test \
  -H "Content-Type: application/json" \
  -d '{
    "arguments": {
      "server_url": "https://openmcp-server.example.com",
      "action": "execute_tool",
      "tool_name": "weather",
      "tool_parameters": {"location": "San Francisco"}
    }
  }'
```

## LLM Providers

### Supported Providers

| Provider | Authentication | Models | Tool Calling |
|----------|---------------|---------|--------------|
| OpenAI | API Key | gpt-4, gpt-3.5-turbo, etc. | ✅ Native |
| Anthropic | API Key | claude-3-opus, claude-3-sonnet, etc. | ✅ Native |
| Mistral | API Key | mistral-large, mistral-medium, etc. | ✅ Native |
| Grok | API Key | grok-beta | ✅ Native |
| Ollama | None (local) | Any Ollama model | ✅ Supported |

### Provider Configuration

Each provider requires specific configuration in `configs/config.yaml`:

```yaml
llm:
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      base_url: "https://api.openai.com/v1"
    ollama:
      base_url: "http://localhost:11434"
```

## Development

### Project Structure
```
agent-server/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/            # HTTP handlers and routing
│   ├── config/         # Configuration management
│   ├── context/        # Context strategies
│   ├── llm/           # LLM provider interfaces
│   │   └── ollama/    # Ollama provider implementation
│   ├── models/        # Data models and DTOs
│   └── storage/       # Database repositories
├── configs/           # Configuration files
├── examples/          # Example scripts and tests
└── test/             # Integration tests
```

### Make Commands

```bash
make dev        # Run development server with live reload
make build      # Build binary
make test       # Run all tests
make test-unit  # Run unit tests only
make test-int   # Run integration tests
make clean      # Clean build artifacts
```

### Running Tests

```bash
# Unit tests
go test ./internal/...

# Integration tests  
go test ./test/...

# All tests with coverage
go test -cover ./...

# Test with real Ollama (requires Ollama running)
./examples/test_real_chat.sh
```

### Adding New Providers

1. Implement the `llm.Provider` interface:
```go
type Provider interface {
    Name() string
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req *ChatRequest) (<-chan ChatChunk, error)
    ListModels(ctx context.Context) ([]string, error)
}
```

2. Register the provider in `cmd/server/main.go`:
```go
newProvider := yourprovider.NewProvider(config)
llmRegistry.Register(newProvider)
```

### Adding New Context Strategies

1. Implement the `context.Strategy` interface:
```go
type Strategy interface {
    Name() string
    BuildContext(ctx context.Context, systemPrompt, agentPrompt string, 
                messages []*models.Message, config map[string]interface{}) ([]*models.Message, error)
}
```

2. Register the strategy in `cmd/server/main.go`:
```go
contextRegistry.Register("my_strategy", &MyStrategy{})
```

### Adding New Tools

Creating custom tools allows you to extend agent capabilities with any functionality you need. Here's how to add a new tool:

#### 1. Create Your Tool Implementation

Create a new file in `internal/tools/builtin/` (e.g., `weather.go`):

```go
package builtin

import (
    "agent-server/internal/tools"
    "encoding/json"
    "fmt"
    "net/http"
)

// WeatherTool provides weather information
type WeatherTool struct {
    *tools.BaseTool
    apiKey string
}

// NewWeatherTool creates a new weather tool
func NewWeatherTool(apiKey string) *WeatherTool {
    schema := tools.Schema{
        Name:        "weather",
        Description: "Get current weather information for a location",
        Parameters: []tools.Parameter{
            {
                Name:        "location",
                Type:        "string",
                Description: "City name or coordinates",
                Required:    true,
            },
            {
                Name:        "units",
                Type:        "string",
                Description: "Temperature units: celsius or fahrenheit",
                Required:    false,
                Default:     "celsius",
                Enum:        []string{"celsius", "fahrenheit"},
            },
        },
        Examples: []tools.Example{
            {
                Description: "Get weather in San Francisco",
                Input: map[string]interface{}{
                    "location": "San Francisco",
                    "units":    "fahrenheit",
                },
                Output: map[string]interface{}{
                    "temperature": 68,
                    "conditions":  "Partly cloudy",
                    "humidity":    65,
                },
            },
        },
    }

    tool := &WeatherTool{apiKey: apiKey}
    tool.BaseTool = tools.NewBaseTool("weather", schema, tool.execute)
    return tool
}

func (w *WeatherTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
    location, ok := input["location"].(string)
    if !ok {
        return tools.ErrorResult("INVALID_LOCATION", "location is required")
    }

    units := "celsius"
    if u, ok := input["units"].(string); ok {
        units = u
    }

    // Make API call to weather service
    // This is a simplified example
    weatherData := map[string]interface{}{
        "location":    location,
        "temperature": 22,
        "units":       units,
        "conditions":  "Sunny",
        "humidity":    45,
    }

    return tools.SuccessResult(weatherData)
}
```

#### 2. Register Your Tool

Add your tool to the registry in `internal/tools/builtin/register.go`:

```go
func RegisterBuiltinTools(registry *tools.Registry, memoryRepo storage.MemoryRepository) error {
    // ... existing tools ...
    
    // Register weather tool (configure API key from environment or config)
    weatherAPIKey := os.Getenv("WEATHER_API_KEY")
    if weatherAPIKey != "" {
        weatherTool := NewWeatherTool(weatherAPIKey)
        if err := registry.Register(weatherTool); err != nil {
            return fmt.Errorf("failed to register weather tool: %w", err)
        }
    }
    
    return nil
}
```

#### 3. Tool Best Practices

1. **Use BaseTool**: Inherit from `tools.BaseTool` for automatic validation and common functionality
2. **Clear Schema**: Define comprehensive schemas with descriptions and examples
3. **Error Handling**: Use `tools.ErrorResult()` with specific error codes
4. **Context Awareness**: Use `ExecutionContext` for session/agent-specific behavior
5. **Async Operations**: For long-running operations, consider timeouts from context
6. **Testing**: Add unit tests in `*_test.go` files

#### 4. Advanced Tool Features

**Stateful Tools**: Tools can maintain state using the repository:

```go
type StatefulTool struct {
    *tools.BaseTool
    repo storage.Repository
}

func (t *StatefulTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
    // Access session or agent-specific data
    sessionID := ctx.SessionID
    agentID := ctx.AgentID
    
    // Store or retrieve state
    // ...
}
```

**Tool Dependencies**: Tools can use other tools:

```go
func (t *MyTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
    // Get another tool from context
    if calcTool, exists := ctx.Metadata["calculator_tool"]; exists {
        // Use calculator tool
    }
}
```

**Dynamic Tool Registration**: Register tools at runtime:

```go
// In your API handler
newTool := custom.NewDynamicTool(config)
toolRegistry.Register(newTool)
```

## Examples

### Quick Test Script
```bash
# Test basic functionality
./examples/quick_test.sh
```

### Comprehensive Test
```bash
# Full workflow test with real Ollama
./examples/test_real_chat.sh
```

### Tool Calling Tests
```bash
# Test all tool calling functionality (comprehensive test suite)
./examples/test_tools.sh

# Test calculator tool with AI agent integration (starts server, creates agent, tests tool calling)
./examples/test_calculator_agent.sh

# Quick calculator tool calling test (assumes server running, fast verification)
./examples/test_tool_calling.sh

# Test memory tool with limited context window
./examples/test_memory_agent.sh
```

**test_calculator_agent.sh** - Comprehensive test that:
- Starts the server if not running
- Verifies calculator tool functionality
- Creates an AI agent with calculator tool access
- Tests enhanced chat with tool calling
- Demonstrates multi-step mathematical conversations
- Cleans up resources when done

**test_tool_calling.sh** - Quick verification test that:
- Assumes server is already running
- Tests basic calculator tool execution
- Creates agent and session with tool access
- Verifies tool calling integration
- Minimal setup for fast feedback

**test_memory_agent.sh** - Memory persistence test that:
- Creates an agent with very limited context window (2 messages)
- Demonstrates memory tool usage for information persistence
- Shows how agents can recall information that's no longer in context
- Tests memory search and retrieval functionality
- Validates personalized responses based on stored memory

### Tool Calling Examples

#### Example 1: Math Assistant with Calculator
```bash
# Create a math tutor agent
AGENT_ID=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Math Tutor",
    "provider": "ollama",
    "model": "mistral-small3.1:latest",
    "system_prompt": "You are a helpful math tutor. Use the calculator tool for all calculations.",
    "temperature": 0.3
  }' | jq -r '.id')

# Create session
SESSION_ID=$(curl -s -X POST "http://localhost:8081/api/v1/agents/$AGENT_ID/sessions" \
  -H "Content-Type: application/json" \
  -d '{"title": "Math Help"}' | jq -r '.id')

# Ask a math question - agent will automatically use calculator
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What is 15% of 340? Also calculate the square root of 144.",
    "tools": ["calculator"],
    "tool_choice": "auto"
  }'
```

#### Example 2: Personal Assistant with Memory
```bash
# Create agent with memory capabilities
AGENT_ID=$(curl -s -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Personal Assistant",
    "provider": "ollama",
    "model": "qwen2.5:14b-instruct",
    "system_prompt": "You are a personal assistant. Remember user preferences and important information using the memory tool.",
    "temperature": 0.5
  }' | jq -r '.id')

# Chat with memory storage
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "My name is Alice and I prefer morning meetings. I work at TechCorp as a software engineer.",
    "tools": ["memory"],
    "tool_choice": "auto"
  }'

# Later in conversation (even after context is lost)
curl -X POST "http://localhost:8081/api/v1/sessions/$SESSION_ID/chat/enhanced" \
  -H "Content-Type: application/json" \
  -d '{
    "message": "What do you remember about my work preferences?",
    "tools": ["memory"],
    "tool_choice": "auto"
  }'
```

#### Example 3: Research Assistant with Multiple Tools
```bash
# Create multi-tool agent
curl -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Research Assistant",
    "provider": "ollama",
    "model": "llama3.3:latest",
    "system_prompt": "You are a research assistant. Use available tools to gather information, perform calculations, and remember important findings.",
    "config": {
      "enabled_tools": ["web_scraper", "calculator", "memory", "text_processor"],
      "max_tool_calls": 10
    }
  }'
```

### Creating a Coding Assistant
```bash
# Create a specialized coding agent
curl -X POST http://localhost:8081/api/v1/agents \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Code Reviewer",
    "provider": "ollama",
    "model": "llama3.2:3b", 
    "system_prompt": "You are an expert code reviewer. Provide constructive feedback on code quality, best practices, and potential improvements.",
    "temperature": 0.3,
    "max_tokens": 500
  }'
```

## Production Deployment

### Database

For production, consider:
- Using absolute paths for SQLite: `/var/lib/agentserver/agents.db`
- Setting up automated backups
- Monitoring database size and performance

### Security

- Keep API keys in environment variables, not config files
- Use HTTPS in production
- Implement rate limiting
- Add authentication/authorization as needed

### Monitoring

The server provides structured JSON logging:
```json
{
  "level": "info",
  "msg": "Chat completed successfully", 
  "provider": "ollama",
  "model": "llama3.2:3b",
  "session_id": "abc-123",
  "time": "2025-07-12T13:23:33+02:00"
}
```

### Docker

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o agent-server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/agent-server .
COPY configs/ configs/
CMD ["./agent-server", "-config", "configs/config.yaml"]
```

## Troubleshooting

### Common Issues

**Server won't start:**
- Check if port 8081 is available: `lsof -i :8081`
- Verify configuration file exists and is valid YAML
- Check database directory permissions

**Ollama connection failed:**
- Ensure Ollama is running: `curl http://localhost:11434/api/tags`
- Check if the model is downloaded: `ollama list`
- Verify Ollama base URL in config

**API key issues:**
- Verify API keys are correctly set in config or environment
- Check API key format and permissions
- Test API keys directly with provider APIs

### Logs

Server logs are written to `server.log` and stdout in JSON format:
```bash
# Follow server logs
tail -f server.log

# Filter for errors
tail -f server.log | jq 'select(.level == "error")'
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature-name`
3. Make changes and add tests
4. Run tests: `make test`
5. Commit changes: `git commit -m "Add feature"`
6. Push to branch: `git push origin feature-name`
7. Create a Pull Request

## License

[Add your license here]

## Support

For issues and questions:
- Create an issue in the repository
- Check existing documentation and examples
- Review the troubleshooting section above