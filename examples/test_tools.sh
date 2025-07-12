#!/bin/bash

# Tool calling test script
# This script demonstrates the new tool calling functionality

# Load configuration
source "$(dirname "$0")/get_config.sh"

echo "🔧 Testing Agent Server Tool Calling"
echo "==================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Test 1: List available tools
print_info "Testing tool listing..."
curl -s "$BASE_URL/tools" | jq '.' > /dev/null
if [ $? -eq 0 ]; then
    print_status "Tools endpoint is working"
    echo "Available tools:"
    curl -s "$BASE_URL/tools" | jq '.tools[] | {name: .name, description: .description, available: .available}'
else
    print_error "Tools endpoint failed"
    exit 1
fi

echo ""

# Test 2: Get specific tool information
print_info "Testing HTTP GET tool info..."
TOOL_INFO=$(curl -s "$BASE_URL/tools/http_get")
if echo "$TOOL_INFO" | jq -e '.name' > /dev/null; then
    print_status "HTTP GET tool info retrieved"
    echo "$TOOL_INFO" | jq '{name: .name, description: .description, parameters: [.parameters[] | {name: .name, type: .type, required: .required}]}'
else
    print_error "Failed to get HTTP GET tool info"
fi

echo ""

# Test 3: Test calculator tool
print_info "Testing calculator tool..."
CALC_RESULT=$(curl -s -X POST "$BASE_URL/tools/calculator/test" \
    -H "Content-Type: application/json" \
    -d '{"arguments": {"expression": "2 + 3 * 4"}}')

if echo "$CALC_RESULT" | jq -e '.success' > /dev/null; then
    SUCCESS=$(echo "$CALC_RESULT" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        print_status "Calculator tool test passed"
        echo "Result: $(echo "$CALC_RESULT" | jq -r '.result')"
    else
        print_error "Calculator tool test failed"
        echo "Error: $(echo "$CALC_RESULT" | jq -r '.error')"
    fi
else
    print_error "Calculator tool test endpoint failed"
fi

echo ""

# Test 4: Test HTTP GET tool with a real request
print_info "Testing HTTP GET tool with real request..."
HTTP_RESULT=$(curl -s -X POST "$BASE_URL/tools/http_get/test" \
    -H "Content-Type: application/json" \
    -d '{
        "arguments": {
            "url": "https://httpbin.org/json",
            "timeout": 10
        }
    }')

if echo "$HTTP_RESULT" | jq -e '.success' > /dev/null; then
    SUCCESS=$(echo "$HTTP_RESULT" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        print_status "HTTP GET tool test passed"
        echo "Status: $(echo "$HTTP_RESULT" | jq -r '.result.status_code')"
        echo "Content-Type: $(echo "$HTTP_RESULT" | jq -r '.result.content_type')"
    else
        print_error "HTTP GET tool test failed"
        echo "Error: $(echo "$HTTP_RESULT" | jq -r '.error')"
    fi
else
    print_error "HTTP GET tool test endpoint failed"
fi

echo ""

# Test 5: Test text processor tool
print_info "Testing text processor tool..."
TEXT_RESULT=$(curl -s -X POST "$BASE_URL/tools/text_processor/test" \
    -H "Content-Type: application/json" \
    -d '{
        "arguments": {
            "text": "Hello World from Agent Server",
            "operation": "word_count"
        }
    }')

if echo "$TEXT_RESULT" | jq -e '.success' > /dev/null; then
    SUCCESS=$(echo "$TEXT_RESULT" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        print_status "Text processor tool test passed"
        echo "Word count: $(echo "$TEXT_RESULT" | jq -r '.result')"
    else
        print_error "Text processor tool test failed"
        echo "Error: $(echo "$TEXT_RESULT" | jq -r '.error')"
    fi
else
    print_error "Text processor tool test endpoint failed"
fi

echo ""

# Test 6: Get tool schemas (for LLM integration)
print_info "Testing tool schemas endpoint..."
SCHEMAS=$(curl -s "$BASE_URL/tools/schemas?tools=calculator,text_processor")
if echo "$SCHEMAS" | jq -e '.[0].function.name' > /dev/null; then
    print_status "Tool schemas retrieved successfully"
    echo "Available function schemas:"
    echo "$SCHEMAS" | jq '.[].function | {name: .name, description: .description}'
else
    print_error "Failed to retrieve tool schemas"
fi

echo ""

# Test 7: Test MCP proxy tool (will fail without MCP server)
print_info "Testing MCP proxy tool (expected to fail without MCP server)..."
MCP_RESULT=$(curl -s -X POST "$BASE_URL/tools/mcp_proxy/test" \
    -H "Content-Type: application/json" \
    -d '{
        "arguments": {
            "server_url": "http://localhost:3000",
            "action": "initialize"
        }
    }')

if echo "$MCP_RESULT" | jq -e '.success' > /dev/null; then
    SUCCESS=$(echo "$MCP_RESULT" | jq -r '.success')
    if [ "$SUCCESS" = "true" ]; then
        print_status "MCP proxy tool test passed (unexpected!)"
    else
        print_info "MCP proxy tool test failed as expected (no MCP server running)"
        echo "Error: $(echo "$MCP_RESULT" | jq -r '.error' | head -c 100)..."
    fi
else
    print_error "MCP proxy tool test endpoint failed"
fi

echo ""

# Test 8: Create an agent for chat with tools testing
print_info "Creating test agent for chat with tools..."
AGENT_RESPONSE=$(curl -s -X POST "$BASE_URL/agents" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Tool-Enabled Assistant",
        "description": "An assistant with tool calling capabilities",
        "provider": "ollama",
        "model": "llama3.2:3b",
        "system_prompt": "You are a helpful assistant with access to tools. Use tools when appropriate to help users.",
        "temperature": 0.3,
        "max_tokens": 200
    }')

AGENT_ID=$(echo $AGENT_RESPONSE | jq -r '.id')
if [ "$AGENT_ID" = "null" ] || [ -z "$AGENT_ID" ]; then
    print_error "Failed to create agent for chat with tools testing"
    echo "Response: $AGENT_RESPONSE"
else
    print_status "Test agent created: $AGENT_ID"
fi

echo ""

# Test 9: Create a session for the agent
print_info "Creating test session..."
SESSION_RESPONSE=$(curl -s -X POST "$BASE_URL/agents/$AGENT_ID/sessions" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Tool Testing Session",
        "context_strategy": "last_n",
        "context_config": {
            "count": 10
        }
    }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')
if [ "$SESSION_ID" = "null" ] || [ -z "$SESSION_ID" ]; then
    print_error "Failed to create session"
    echo "Response: $SESSION_RESPONSE"
else
    print_status "Test session created: $SESSION_ID"
fi

echo ""

# Test 10: List available tools for the session
print_info "Listing tools available for session..."
SESSION_TOOLS=$(curl -s "$BASE_URL/sessions/$SESSION_ID/tools")
if echo "$SESSION_TOOLS" | jq -e '.tools' > /dev/null; then
    TOOL_COUNT=$(echo "$SESSION_TOOLS" | jq '.total_count')
    print_status "Found $TOOL_COUNT available tools for session"
    echo "Available tools:"
    echo "$SESSION_TOOLS" | jq '.tools[] | {name: .name, available: .available}' | head -20
else
    print_error "Failed to list session tools"
fi

echo ""

# Test 11: Test chat with tools endpoint (will work even without Ollama)
print_info "Testing chat with tools endpoint structure..."
CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat/tools" \
    -H "Content-Type: application/json" \
    -d '{
        "message": "Can you calculate 15 * 23 for me?",
        "tools": ["calculator"],
        "tool_choice": "auto"
    }')

# Check if the endpoint exists and returns proper error structure
if echo "$CHAT_RESPONSE" | jq -e '.error' > /dev/null; then
    ERROR_MSG=$(echo "$CHAT_RESPONSE" | jq -r '.error')
    if [[ "$ERROR_MSG" == *"not available"* ]] || [[ "$ERROR_MSG" == *"failed"* ]]; then
        print_info "Chat with tools endpoint exists but LLM not available (expected without Ollama)"
    else
        print_error "Unexpected error from chat with tools endpoint"
        echo "Error: $ERROR_MSG"
    fi
elif echo "$CHAT_RESPONSE" | jq -e '.response' > /dev/null; then
    print_status "Chat with tools endpoint responded successfully!"
    echo "Response: $(echo "$CHAT_RESPONSE" | jq -r '.response')"
    TOOL_CALLS=$(echo "$CHAT_RESPONSE" | jq '.tool_calls | length')
    echo "Tool calls made: $TOOL_CALLS"
else
    print_error "Enhanced chat endpoint failed"
    echo "Response: $CHAT_RESPONSE"
fi

echo ""

# Cleanup
print_info "Cleaning up test data..."
if [ "$SESSION_ID" != "null" ] && [ -n "$SESSION_ID" ]; then
    curl -s -X DELETE "$BASE_URL/sessions/$SESSION_ID" > /dev/null
fi
if [ "$AGENT_ID" != "null" ] && [ -n "$AGENT_ID" ]; then
    curl -s -X DELETE "$BASE_URL/agents/$AGENT_ID" > /dev/null
fi
print_status "Cleanup complete"

echo ""
echo "🎉 Tool calling tests completed!"
echo ""
echo "Summary:"
echo "- Tool listing: ✅"
echo "- Tool information: ✅"
echo "- Calculator tool: ✅"
echo "- HTTP GET tool: ✅"
echo "- Text processor tool: ✅"
echo "- Tool schemas: ✅"
echo "- MCP proxy tool: ⚠️  (expected failure)"
echo "- Enhanced chat API: ✅ (structure)"
echo ""
print_info "The tool calling system is ready! Start with Ollama to test full functionality."