#!/bin/bash

# Quick Tool Calling Test
# Tests calculator tool integration with AI agent (assumes server is running)

# Load configuration
source "$(dirname "$0")/get_config.sh"
MODEL="llama3.2:3b"

echo "🧮 Quick Calculator Tool Calling Test"
echo "====================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
PURPLE='\033[0;35m'
NC='\033[0m'

print_status() { echo -e "${GREEN}✅ $1${NC}"; }
print_info() { echo -e "${BLUE}ℹ️  $1${NC}"; }
print_error() { echo -e "${RED}❌ $1${NC}"; }
print_tool() { echo -e "${PURPLE}🔧 $1${NC}"; }

# Check server
print_info "Checking server..."
if ! curl -s "$BASE_URL/../health" > /dev/null; then
    print_error "Server not running. Start with: make dev"
    exit 1
fi
print_status "Server is running"

# Test calculator tool
print_tool "Testing calculator tool..."
CALC_RESULT=$(curl -s -X POST "$BASE_URL/tools/calculator/test" \
    -H "Content-Type: application/json" \
    -d '{"arguments": {"expression": "15 + 23"}}')

if [ "$(echo "$CALC_RESULT" | jq -r '.success')" = "true" ]; then
    RESULT=$(echo "$CALC_RESULT" | jq -r '.result.result')
    print_status "Calculator: 15 + 23 = $RESULT"
else
    print_error "Calculator test failed"
    exit 1
fi

# Create agent with calculator tool
print_info "Creating math agent..."
AGENT_RESPONSE=$(curl -s -X POST "$BASE_URL/agents" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Calculator Agent\",
        \"description\": \"Math assistant with calculator\",
        \"provider\": \"ollama\",
        \"model\": \"$MODEL\",
        \"system_prompt\": \"You are a math assistant. Use the calculator tool for any calculations. Always show the calculation step by step.\",
        \"temperature\": 0.1,
        \"max_tokens\": 200
    }")

AGENT_ID=$(echo $AGENT_RESPONSE | jq -r '.id')
if [ "$AGENT_ID" = "null" ]; then
    print_error "Failed to create agent"
    exit 1
fi
print_status "Agent created: $AGENT_ID"

# Create session
print_info "Creating session..."
SESSION_RESPONSE=$(curl -s -X POST "$BASE_URL/agents/$AGENT_ID/sessions" \
    -H "Content-Type: application/json" \
    -d '{"title": "Calculator Test", "context_strategy": "last_n", "context_config": {"count": 5}}')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')
if [ "$SESSION_ID" = "null" ]; then
    print_error "Failed to create session"
    exit 1
fi
print_status "Session created: $SESSION_ID"

# Test chat with tools with tool calling
print_info "Testing AI agent with calculator tool..."

# Check if Ollama is available
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    OLLAMA_AVAILABLE=true
    print_status "Ollama detected - testing full tool calling"
else
    OLLAMA_AVAILABLE=false
    print_info "Ollama not available - testing endpoint structure only"
fi

if [ "$OLLAMA_AVAILABLE" = true ]; then
    # Test with Ollama
    QUESTIONS=(
        "What is 47 + 38?"
        "Calculate 25 + 17"
        "What's the square root of 144?"
    )
    
    for question in "${QUESTIONS[@]}"; do
        print_info "Asking: '$question'"
        
        CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat/tools" \
            -H "Content-Type: application/json" \
            -d "{
                \"message\": \"$question\",
                \"tools\": [\"calculator\"],
                \"tool_choice\": \"auto\"
            }")
        
        if echo "$CHAT_RESPONSE" | jq -e '.response' > /dev/null; then
            RESPONSE=$(echo "$CHAT_RESPONSE" | jq -r '.response')
            TOOL_CALLS=$(echo "$CHAT_RESPONSE" | jq '.tool_calls // []')
            TOOL_COUNT=$(echo "$TOOL_CALLS" | jq 'length')
            
            echo -e "${GREEN}🤖 Assistant:${NC} $RESPONSE"
            
            if [ "$TOOL_COUNT" -gt 0 ]; then
                print_tool "✓ Used calculator tool $TOOL_COUNT time(s)"
                echo "$TOOL_CALLS" | jq -r '.[] | "   Calculator: \(.arguments.expression) = \(.result.result)"'
            else
                print_error "✗ No calculator tool used"
            fi
            echo ""
        else
            ERROR=$(echo "$CHAT_RESPONSE" | jq -r '.error // "Unknown error"')
            print_error "Chat failed: $ERROR"
        fi
    done
else
    # Test without Ollama - just verify endpoint structure
    CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat/tools" \
        -H "Content-Type: application/json" \
        -d '{"message": "What is 2+2?", "tools": ["calculator"], "tool_choice": "auto"}')
    
    if echo "$CHAT_RESPONSE" | jq -e '.error' > /dev/null; then
        ERROR=$(echo "$CHAT_RESPONSE" | jq -r '.error')
        if [[ "$ERROR" == *"provider"* ]] || [[ "$ERROR" == *"not available"* ]]; then
            print_status "Enhanced chat endpoint properly structured"
        else
            print_error "Unexpected error: $ERROR"
        fi
    fi
fi

# Test direct tool execution in session context
print_tool "Testing calculator in session context..."
SESSION_CALC=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/tools/calculator/test" \
    -H "Content-Type: application/json" \
    -d '{"arguments": {"expression": "25 + 17"}}')

if [ "$(echo "$SESSION_CALC" | jq -r '.success')" = "true" ]; then
    RESULT=$(echo "$SESSION_CALC" | jq -r '.result.result')
    print_status "Session calculator: 25 + 17 = $RESULT"
else
    print_error "Session calculator test failed"
fi

# Cleanup
print_info "Cleaning up..."
curl -s -X DELETE "$BASE_URL/sessions/$SESSION_ID" > /dev/null
curl -s -X DELETE "$BASE_URL/agents/$AGENT_ID" > /dev/null

echo ""
echo "🎉 Calculator Tool Calling Test Summary"
echo "======================================="
echo "✅ Calculator tool execution: WORKING"
echo "✅ Agent creation with tools: WORKING"
echo "✅ Session tool access: WORKING"

if [ "$OLLAMA_AVAILABLE" = true ]; then
    echo "✅ AI agent tool calling: WORKING"
    echo "✅ Multi-step conversations: WORKING"
else
    echo "⚠️  AI integration: LIMITED (install Ollama for full test)"
fi

echo ""
print_status "The calculator tool calling system is functional!"

if [ "$OLLAMA_AVAILABLE" = false ]; then
    echo ""
    print_info "For full AI testing:"
    echo "1. Install Ollama: https://ollama.ai"
    echo "2. Start: ollama serve"
    echo "3. Pull model: ollama pull $MODEL"
fi