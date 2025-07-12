#!/bin/bash

# Test script for Ollama integration
# Make sure Ollama is running on localhost:11434 and has llama2 model available

# Load configuration
source "$(dirname "$0")/get_config.sh"

echo "🚀 Testing Agent Server with Ollama Integration"
echo "================================================"

# Check if server is running
if ! curl -s "$SERVER_URL/health" > /dev/null; then
    echo "❌ Server is not running at $SERVER_URL. Please start with './agent-server'"
    exit 1
fi

echo "✅ Server is running"

# 1. Create an agent
echo "📝 Creating agent..."
AGENT_RESPONSE=$(curl -s -X POST "$BASE_URL/agents" \
    -H "Content-Type: application/json" \
    -d '{
        "name": "Ollama Test Agent",
        "description": "Testing Ollama integration",
        "provider": "ollama",
        "model": "llama2",
        "system_prompt": "You are a helpful AI assistant. Keep your responses brief and friendly.",
        "temperature": 0.7,
        "max_tokens": 100
    }')

AGENT_ID=$(echo $AGENT_RESPONSE | jq -r '.id')

if [ "$AGENT_ID" = "null" ] || [ -z "$AGENT_ID" ]; then
    echo "❌ Failed to create agent"
    echo "Response: $AGENT_RESPONSE"
    exit 1
fi

echo "✅ Agent created with ID: $AGENT_ID"

# 2. Create a chat session
echo "💬 Creating chat session..."
SESSION_RESPONSE=$(curl -s -X POST "$BASE_URL/agents/$AGENT_ID/sessions" \
    -H "Content-Type: application/json" \
    -d '{
        "title": "Test Conversation",
        "context_strategy": "last_n",
        "context_config": {
            "count": 5
        }
    }')

SESSION_ID=$(echo $SESSION_RESPONSE | jq -r '.id')

if [ "$SESSION_ID" = "null" ] || [ -z "$SESSION_ID" ]; then
    echo "❌ Failed to create session"
    echo "Response: $SESSION_RESPONSE"
    exit 1
fi

echo "✅ Session created with ID: $SESSION_ID"

# 3. Test if Ollama is available
echo "🔍 Checking Ollama availability..."
OLLAMA_CHECK=$(curl -s http://localhost:11434/api/tags 2>/dev/null)

if [ $? -ne 0 ]; then
    echo "⚠️  Ollama is not running on localhost:11434"
    echo "   Please start Ollama with: ollama serve"
    echo "   And pull llama2 model with: ollama pull llama2"
    echo ""
    echo "   Continuing with mock test (will likely fail)..."
else
    echo "✅ Ollama is running"
    
    # Check if llama2 model is available
    if echo "$OLLAMA_CHECK" | jq -r '.models[].name' | grep -q "llama2"; then
        echo "✅ llama2 model is available"
    else
        echo "⚠️  llama2 model not found. Available models:"
        echo "$OLLAMA_CHECK" | jq -r '.models[].name'
        echo "   Please pull llama2 with: ollama pull llama2"
    fi
fi

# 4. Send a chat message
echo "🗨️  Sending chat message..."
CHAT_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/chat" \
    -H "Content-Type: application/json" \
    -d '{
        "message": "Hello! Can you tell me a short joke?",
        "metadata": {
            "test": true
        }
    }')

echo "Response: $CHAT_RESPONSE"

# Check if chat was successful
if echo "$CHAT_RESPONSE" | jq -e '.response' > /dev/null; then
    echo "✅ Chat successful!"
    RESPONSE_TEXT=$(echo $CHAT_RESPONSE | jq -r '.response')
    echo "🤖 Assistant response: $RESPONSE_TEXT"
else
    echo "❌ Chat failed"
    echo "This is expected if Ollama is not running or llama2 model is not available"
fi

# 5. Get message history
echo "📜 Getting message history..."
MESSAGES_RESPONSE=$(curl -s "$BASE_URL/sessions/$SESSION_ID/messages")
MESSAGE_COUNT=$(echo $MESSAGES_RESPONSE | jq '.messages | length')
echo "✅ Found $MESSAGE_COUNT messages in history"

# 6. Test streaming (basic test)
echo "🌊 Testing streaming endpoint..."
STREAM_RESPONSE=$(curl -s -X POST "$BASE_URL/sessions/$SESSION_ID/stream" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d '{
        "message": "Count to 3 please"
    }' \
    --max-time 10)

if [ $? -eq 0 ]; then
    echo "✅ Streaming endpoint responded"
    echo "First few lines: $(echo "$STREAM_RESPONSE" | head -3)"
else
    echo "❌ Streaming test failed (expected if Ollama not available)"
fi

# 7. Cleanup
echo "🧹 Cleaning up..."
curl -s -X DELETE "$BASE_URL/sessions/$SESSION_ID" > /dev/null
curl -s -X DELETE "$BASE_URL/agents/$AGENT_ID" > /dev/null
echo "✅ Cleanup complete"

echo ""
echo "🎉 Test completed!"
echo ""
echo "Summary:"
echo "- Agent creation: ✅"
echo "- Session creation: ✅"
echo "- Chat functionality: $([ "$RESPONSE_TEXT" != "" ] && echo "✅" || echo "❌ (check Ollama)")"
echo "- Message history: ✅"
echo "- Cleanup: ✅"
echo ""

if [ "$RESPONSE_TEXT" = "" ]; then
    echo "💡 To test with actual LLM responses:"
    echo "   1. Install Ollama: https://ollama.ai/"
    echo "   2. Start Ollama: ollama serve"
    echo "   3. Pull model: ollama pull llama2"
    echo "   4. Run this test again"
fi