package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"agent-server/internal/tools"
)

// MCPProxyTool provides access to Model Context Protocol servers
type MCPProxyTool struct {
	*tools.HTTPBaseTool
	client *http.Client
}

// NewMCPProxyTool creates a new MCP proxy tool
func NewMCPProxyTool() *MCPProxyTool {
	schema := tools.Schema{
		Name:        "mcp_proxy",
		Description: "Proxy tool for interacting with Model Context Protocol (MCP) servers to access external resources and tools",
		Parameters: []tools.Parameter{
			{
				Name:        "server_url",
				Type:        "string",
				Description: "Base URL of the MCP server",
				Required:    true,
				Pattern:     `^https?://.*`,
			},
			{
				Name:        "action",
				Type:        "string",
				Description: "MCP action to perform",
				Required:    true,
				Enum:        []string{"initialize", "list_resources", "get_resource", "list_tools", "call_tool", "search"},
			},
			{
				Name:        "resource_id",
				Type:        "string",
				Description: "Resource ID for get_resource action",
				Required:    false,
			},
			{
				Name:        "tool_name",
				Type:        "string",
				Description: "Tool name for call_tool action",
				Required:    false,
			},
			{
				Name:        "tool_arguments",
				Type:        "object",
				Description: "Arguments for tool execution",
				Required:    false,
			},
			{
				Name:        "query",
				Type:        "string",
				Description: "Search query for search action",
				Required:    false,
			},
			{
				Name:        "auth_token",
				Type:        "string",
				Description: "Authentication token for the MCP server",
				Required:    false,
			},
		},
		Examples: []tools.Example{
			{
				Description: "List available resources from MCP server",
				Input: map[string]interface{}{
					"server_url": "https://mcp.example.com",
					"action":     "list_resources",
				},
				Output: map[string]interface{}{
					"resources": []interface{}{
						map[string]interface{}{
							"id":          "doc1",
							"name":        "Documentation",
							"description": "API Documentation",
							"type":        "text",
						},
					},
				},
			},
			{
				Description: "Execute a tool via MCP server",
				Input: map[string]interface{}{
					"server_url":     "https://mcp.example.com",
					"action":         "call_tool",
					"tool_name":      "calculator",
					"tool_arguments": map[string]interface{}{"expression": "2 + 2"},
				},
				Output: map[string]interface{}{
					"result": "4",
				},
			},
		},
	}

	tool := &MCPProxyTool{
		HTTPBaseTool: tools.NewHTTPBaseTool("mcp_proxy", schema, "", 60*time.Second),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Set the execute function
	tool.BaseTool = tools.NewBaseTool("mcp_proxy", schema, tool.execute)
	
	return tool
}

func (m *MCPProxyTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	serverURL := input["server_url"].(string)
	action := input["action"].(string)

	switch action {
	case "initialize":
		return m.initialize(ctx, serverURL, input)
	case "list_resources":
		return m.listResources(ctx, serverURL, input)
	case "get_resource":
		return m.getResource(ctx, serverURL, input)
	case "list_tools":
		return m.listTools(ctx, serverURL, input)
	case "call_tool":
		return m.callTool(ctx, serverURL, input)
	case "search":
		return m.search(ctx, serverURL, input)
	default:
		return tools.ErrorResult("INVALID_ACTION", fmt.Sprintf("Unknown action: %s", action))
	}
}

func (m *MCPProxyTool) initialize(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"roots": map[string]interface{}{
					"listChanged": true,
				},
				"sampling": map[string]interface{}{},
			},
			"clientInfo": map[string]interface{}{
				"name":    "agent-server",
				"version": "1.0.0",
			},
		},
	}

	return m.makeRequest(ctx, serverURL+"/initialize", payload, input)
}

func (m *MCPProxyTool) listResources(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/list",
		"params":  map[string]interface{}{},
	}

	return m.makeRequest(ctx, serverURL+"/resources/list", payload, input)
}

func (m *MCPProxyTool) getResource(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	resourceID, ok := input["resource_id"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_RESOURCE_ID", "resource_id is required for get_resource action")
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "resources/read",
		"params": map[string]interface{}{
			"uri": resourceID,
		},
	}

	return m.makeRequest(ctx, serverURL+"/resources/read", payload, input)
}

func (m *MCPProxyTool) listTools(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]interface{}{},
	}

	return m.makeRequest(ctx, serverURL+"/tools/list", payload, input)
}

func (m *MCPProxyTool) callTool(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	toolName, ok := input["tool_name"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_TOOL_NAME", "tool_name is required for call_tool action")
	}

	arguments := make(map[string]interface{})
	if args, ok := input["tool_arguments"].(map[string]interface{}); ok {
		arguments = args
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      toolName,
			"arguments": arguments,
		},
	}

	return m.makeRequest(ctx, serverURL+"/tools/call", payload, input)
}

func (m *MCPProxyTool) search(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	query, ok := input["query"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_QUERY", "query is required for search action")
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "search",
		"params": map[string]interface{}{
			"query": query,
		},
	}

	return m.makeRequest(ctx, serverURL+"/search", payload, input)
}

func (m *MCPProxyTool) makeRequest(ctx tools.ExecutionContext, url string, payload map[string]interface{}, input map[string]interface{}) *tools.Result {
	// Marshal payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return tools.ErrorResult("JSON_ENCODING_FAILED", fmt.Sprintf("Failed to encode request: %v", err))
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "POST", url, bytes.NewReader(jsonData))
	if err != nil {
		return tools.ErrorResult("REQUEST_CREATION_FAILED", fmt.Sprintf("Failed to create request: %v", err))
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "agent-server/1.0")

	// Add auth token if provided
	if authToken, ok := input["auth_token"].(string); ok && authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}

	// Execute request
	resp, err := m.client.Do(req)
	if err != nil {
		return tools.ErrorResult("REQUEST_FAILED", fmt.Sprintf("MCP request failed: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult("RESPONSE_READ_FAILED", fmt.Sprintf("Failed to read response: %v", err))
	}

	// Parse JSON response
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return tools.ErrorResult("JSON_DECODE_FAILED", fmt.Sprintf("Failed to decode response: %v", err))
	}

	// Check for JSON-RPC error
	if errObj, ok := response["error"]; ok {
		if errMap, ok := errObj.(map[string]interface{}); ok {
			code := "UNKNOWN_ERROR"
			message := "Unknown MCP error"
			
			if c, ok := errMap["code"]; ok {
				code = fmt.Sprintf("MCP_ERROR_%v", c)
			}
			if m, ok := errMap["message"].(string); ok {
				message = m
			}
			
			return tools.ErrorResult(code, message, map[string]interface{}{
				"mcp_error": errObj,
			})
		}
	}

	// Return successful result
	result := response["result"]
	if result == nil {
		result = response
	}

	return tools.SuccessResult(result, map[string]interface{}{
		"status_code": resp.StatusCode,
		"server_url":  input["server_url"],
		"action":      payload["method"],
	})
}

// IsAvailable checks if the MCP proxy tool is available
func (m *MCPProxyTool) IsAvailable(ctx context.Context) bool {
	// The tool is available if we can make HTTP requests
	return true
}