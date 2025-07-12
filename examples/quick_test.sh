#!/bin/bash

# Quick test script - single chat interaction
# Load configuration
source "$(dirname "$0")/get_config.sh"

echo "🚀 Quick Ollama Chat Test"
echo "========================="

# Check if server is running
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    echo "❌ Server not running. Start with: make dev"
    exit 1
fi

# Create agent
echo "📝 Creating agent..."
AGENT=$(curl -s -X POST "$BASE_URL/agents" -H "Content-Type: application/json" -d '{
    "name": "Test Assistant",
    "provider": "ollama", 
    "model": "llama3.2:3b",
    "system_prompt": "You are a helpful assistant. Keep responses very brief.",
    "temperature": 0.7,
    "max_tokens": 50
}')

AGENT_ID=$(echo $AGENT | jq -r '.id')
echo "✅ Agent created: $AGENT_ID"

# Create session
echo "💬 Creating session..."
SESSION=$(curl -s -X POST "$BASE_URL/agents/$AGENT_ID/sessions" -H "Content-Type: application/json" -d '{
    "title": "Quick Test"
}')

SESSION_ID=$(echo $SESSION | jq -r '.id')
echo "✅ Session created: $SESSION_ID"

# Send chat message
echo "🗨️  Sending message..."
RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat" -H "Content-Type: application/json" -d '{
    "message": "Hello! Can you tell me what 5+3 equals?"
}')

# Check if successful
if echo "$RESPONSE" | jq -e '.response' > /dev/null; then
    echo "✅ Chat successful!"
    echo "🤖 Response: $(echo $RESPONSE | jq -r '.response')"
    
    # Show some metadata
    PROVIDER=$(echo $RESPONSE | jq -r '.metadata.provider')
    MODEL=$(echo $RESPONSE | jq -r '.metadata.model')
    echo "📊 Provider: $PROVIDER, Model: $MODEL"
else
    echo "❌ Chat failed"
    echo "Error: $RESPONSE"
fi

# Cleanup
echo "🧹 Cleaning up..."
curl -s -X DELETE "$BASE_URL/sessions/$SESSION_ID" > /dev/null
curl -s -X DELETE "$BASE_URL/agents/$AGENT_ID" > /dev/null

echo "✅ Test complete!"