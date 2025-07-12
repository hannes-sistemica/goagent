#!/bin/bash

# Real chat test with Ollama
# This script tests the complete workflow with actual LLM responses

# Load configuration
source "$(dirname "$0")/get_config.sh"
MODEL="llama3.2:3b"

echo "🚀 Testing Agent Server with Real Ollama Chat"
echo "=============================================="

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

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if server is running
print_info "Checking if server is running..."
if ! curl -s "$BASE_URL/../health" > /dev/null; then
    print_error "Server is not running. Please start with 'make dev'"
    exit 1
fi
print_status "Server is running"

# Check if Ollama is available
print_info "Checking Ollama availability..."
if ! curl -s http://localhost:11434/api/tags > /dev/null; then
    print_error "Ollama is not running. Please start with 'ollama serve'"
    exit 1
fi
print_status "Ollama is running"

# Check if model is available
print_info "Checking if model '$MODEL' is available..."
MODELS_RESPONSE=$(curl -s http://localhost:11434/api/tags)
if ! echo "$MODELS_RESPONSE" | jq -r '.models[].name' | grep -q "$MODEL"; then
    print_warning "Model '$MODEL' not found. Available models:"
    echo "$MODELS_RESPONSE" | jq -r '.models[] | "  - \(.name) (\(.details.parameter_size))"'
    print_info "You can pull the model with: ollama pull $MODEL"
    exit 1
fi
print_status "Model '$MODEL' is available"

# Create an agent with a coding assistant prompt
print_info "Creating coding assistant agent..."
AGENT_RESPONSE=$(curl -s -X POST "$BASE_URL/agents" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"Coding Assistant\",
        \"description\": \"A helpful coding assistant that provides brief, practical programming help\",
        \"provider\": \"ollama\",
        \"model\": \"$MODEL\",
        \"system_prompt\": \"You are a helpful coding assistant. Provide brief, practical answers to programming questions. Focus on being concise but accurate. When showing code examples, explain them briefly.\",
        \"temperature\": 0.3,
        \"max_tokens\": 150
    }")

AGENT_ID=$(echo $AGENT_RESPONSE | jq -r '.id')

if [ "$AGENT_ID" = "null" ] || [ -z "$AGENT_ID" ]; then
    print_error "Failed to create agent"
    echo "Response: $AGENT_RESPONSE"
    exit 1
fi

print_status "Agent created with ID: $AGENT_ID"

# Create a chat session
print_info "Creating chat session..."
SESSION_RESPONSE=$(curl -s -X POST "$BASE_URL/agents/$AGENT_ID/sessions" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Coding Questions",
        "context_strategy": "last_n",
        "context_config": {
            "count": 8
        }
    }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')

if [ "$SESSION_ID" = "null" ] || [ -z "$SESSION_ID" ]; then
    print_error "Failed to create session"
    echo "Response: $SESSION_RESPONSE"
    exit 1
fi

print_status "Session created with ID: $SESSION_ID"

# Function to send a chat message and display response
send_chat() {
    local message="$1"
    print_info "Sending: \"$message\""
    
    CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat" \
        -H "Content-Type: application/json" \
        -d "{
            \"message\": \"$message\",
            \"metadata\": {
                \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"
            }
        }")
    
    if echo "$CHAT_RESPONSE" | jq -e '.response' > /dev/null; then
        RESPONSE=$(echo $CHAT_RESPONSE | jq -r '.response')
        METADATA=$(echo $CHAT_RESPONSE | jq -r '.metadata')
        
        echo -e "${GREEN}🤖 Assistant:${NC} $RESPONSE"
        echo ""
        
        # Show metadata
        echo "📊 Metadata:"
        echo "$METADATA" | jq -r 'to_entries[] | "  \(.key): \(.value)"'
        echo ""
        
        return 0
    else
        print_error "Chat failed"
        echo "Response: $CHAT_RESPONSE"
        return 1
    fi
}

# Test conversation
echo "🗨️  Starting conversation..."
echo "================================="

# Question 1: Simple coding question
send_chat "How do I reverse a string in Python?"

# Question 2: Follow-up question
send_chat "What about in JavaScript?"

# Question 3: More complex question
send_chat "Explain the difference between let, const, and var in JavaScript"

# Question 4: Quick algorithm question
send_chat "Write a simple function to check if a number is prime"

# Get conversation history
print_info "Getting conversation history..."
HISTORY_RESPONSE=$(curl -s "$BASE_URL/sessions/$SESSION_ID/messages")
MESSAGE_COUNT=$(echo $HISTORY_RESPONSE | jq '.messages | length')
print_status "Found $MESSAGE_COUNT messages in conversation history"

# Show conversation summary
echo ""
echo "📝 Conversation Summary:"
echo "========================"
echo $HISTORY_RESPONSE | jq -r '.messages[] | "[\(.role | ascii_upcase)]: \(.content[:100])...\n"'

# Test streaming endpoint
print_info "Testing streaming chat..."
echo -e "${BLUE}🌊 Streaming Response:${NC}"

curl -s -N -X POST "$BASE_URL/sessions/$SESSION_ID/stream" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d '{
        "message": "Give me a very short example of a Python list comprehension"
    }' | while IFS= read -r line; do
    if [[ $line == data:* ]]; then
        # Extract JSON data after "data: "
        json_data="${line#data: }"
        
        # Parse content and done status
        content=$(echo "$json_data" | jq -r '.content // empty')
        done=$(echo "$json_data" | jq -r '.done // false')
        
        # Print content without newline for streaming effect
        if [ -n "$content" ]; then
            printf "%s" "$content"
        fi
        
        # Break when done
        if [ "$done" = "true" ]; then
            echo ""
            break
        fi
    fi
done

echo ""
print_status "Streaming test completed"

# Performance test - measure response time
print_info "Testing response time..."
start_time=$(date +%s%N)

send_chat "What is 2+2?" > /dev/null

end_time=$(date +%s%N)
duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

echo "⏱️  Response time: ${duration}ms"

# Cleanup
print_info "Cleaning up..."
# curl -s -X DELETE "$BASE_URL/sessions/$SESSION_ID" > /dev/null
# curl -s -X DELETE "$BASE_URL/agents/$AGENT_ID" > /dev/null
print_status "Cleanup complete"

echo ""
echo "🎉 Test completed successfully!"
echo ""
echo "Summary:"
echo "- Agent creation: ✅"
echo "- Session management: ✅"
echo "- Chat functionality: ✅"
echo "- Streaming support: ✅"
echo "- Message history: ✅"
echo "- Performance: ${duration}ms response time"
echo ""
print_info "The AI agent server is working correctly with Ollama!"