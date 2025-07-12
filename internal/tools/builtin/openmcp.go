package builtin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"agent-server/internal/tools"
)

// OpenMCPProxyTool provides access to OpenMCP REST API servers
type OpenMCPProxyTool struct {
	*tools.HTTPBaseTool
	client *http.Client
}

// NewOpenMCPProxyTool creates a new OpenMCP proxy tool
func NewOpenMCPProxyTool() *OpenMCPProxyTool {
	schema := tools.Schema{
		Name:        "openmcp_proxy",
		Description: "Proxy tool for interacting with OpenMCP REST API servers to access external resources and tools via standardized REST endpoints",
		Parameters: []tools.Parameter{
			{
				Name:        "server_url",
				Type:        "string",
				Description: "Base URL of the OpenMCP server",
				Required:    true,
				Pattern:     `^https?://.*`,
			},
			{
				Name:        "action",
				Type:        "string",
				Description: "OpenMCP action to perform",
				Required:    true,
				Enum:        []string{"discovery", "list_resources", "get_resource", "list_tools", "execute_tool", "stream_completion", "health_check"},
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
				Description: "Tool name for execute_tool action",
				Required:    false,
			},
			{
				Name:        "tool_parameters",
				Type:        "object",
				Description: "Parameters for tool execution",
				Required:    false,
			},
			{
				Name:        "prompt",
				Type:        "string",
				Description: "Prompt for completion request",
				Required:    false,
			},
			{
				Name:        "model",
				Type:        "string",
				Description: "Model to use for completion",
				Required:    false,
			},
			{
				Name:        "max_tokens",
				Type:        "number",
				Description: "Maximum tokens for completion",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
				Maximum:     func() *float64 { v := 8192.0; return &v }(),
			},
			{
				Name:        "temperature",
				Type:        "number",
				Description: "Temperature for completion",
				Required:    false,
				Minimum:     func() *float64 { v := 0.0; return &v }(),
				Maximum:     func() *float64 { v := 2.0; return &v }(),
			},
			{
				Name:        "api_key",
				Type:        "string",
				Description: "API key for authentication",
				Required:    false,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Discover OpenMCP server capabilities",
				Input: map[string]interface{}{
					"server_url": "https://openmcp.example.com",
					"action":     "discovery",
				},
				Output: map[string]interface{}{
					"name":         "Example OpenMCP Server",
					"version":      "1.0.0",
					"capabilities": []string{"resources", "tools", "sampling"},
				},
			},
			{
				Description: "Execute a tool via OpenMCP server",
				Input: map[string]interface{}{
					"server_url":      "https://openmcp.example.com",
					"action":          "execute_tool",
					"tool_name":       "weather",
					"tool_parameters": map[string]interface{}{"location": "San Francisco"},
				},
				Output: map[string]interface{}{
					"success": true,
					"result":  map[string]interface{}{"temperature": "72°F", "condition": "Sunny"},
				},
			},
		},
	}

	tool := &OpenMCPProxyTool{
		HTTPBaseTool: tools.NewHTTPBaseTool("openmcp_proxy", schema, "", 60*time.Second),
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}

	// Set the execute function
	tool.BaseTool = tools.NewBaseTool("openmcp_proxy", schema, tool.execute)
	
	return tool
}

func (o *OpenMCPProxyTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	serverURL := input["server_url"].(string)
	action := input["action"].(string)

	switch action {
	case "discovery":
		return o.discovery(ctx, serverURL, input)
	case "list_resources":
		return o.listResources(ctx, serverURL, input)
	case "get_resource":
		return o.getResource(ctx, serverURL, input)
	case "list_tools":
		return o.listTools(ctx, serverURL, input)
	case "execute_tool":
		return o.executeTool(ctx, serverURL, input)
	case "stream_completion":
		return o.streamCompletion(ctx, serverURL, input)
	case "health_check":
		return o.healthCheck(ctx, serverURL, input)
	default:
		return tools.ErrorResult("INVALID_ACTION", fmt.Sprintf("Unknown action: %s", action))
	}
}

func (o *OpenMCPProxyTool) discovery(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	url := fmt.Sprintf("%s/discovery", serverURL)
	return o.makeGetRequest(ctx, url, input)
}

func (o *OpenMCPProxyTool) listResources(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	url := fmt.Sprintf("%s/resources", serverURL)
	return o.makeGetRequest(ctx, url, input)
}

func (o *OpenMCPProxyTool) getResource(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	resourceID, ok := input["resource_id"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_RESOURCE_ID", "resource_id is required for get_resource action")
	}

	// URL encode the resource ID
	encodedID := url.QueryEscape(resourceID)
	url := fmt.Sprintf("%s/resources/%s", serverURL, encodedID)
	return o.makeGetRequest(ctx, url, input)
}

func (o *OpenMCPProxyTool) listTools(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	url := fmt.Sprintf("%s/tools", serverURL)
	return o.makeGetRequest(ctx, url, input)
}

func (o *OpenMCPProxyTool) executeTool(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	toolName, ok := input["tool_name"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_TOOL_NAME", "tool_name is required for execute_tool action")
	}

	parameters := make(map[string]interface{})
	if params, ok := input["tool_parameters"].(map[string]interface{}); ok {
		parameters = params
	}

	payload := map[string]interface{}{
		"parameters": parameters,
	}

	// URL encode the tool name
	encodedName := url.QueryEscape(toolName)
	url := fmt.Sprintf("%s/tools/%s/execute", serverURL, encodedName)
	return o.makePostRequest(ctx, url, payload, input)
}

func (o *OpenMCPProxyTool) streamCompletion(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	prompt, ok := input["prompt"].(string)
	if !ok {
		return tools.ErrorResult("MISSING_PROMPT", "prompt is required for stream_completion action")
	}

	payload := map[string]interface{}{
		"prompt": prompt,
		"stream": false, // For simplicity, we'll use non-streaming for now
	}

	if model, ok := input["model"].(string); ok {
		payload["model"] = model
	}

	if maxTokens, ok := input["max_tokens"].(float64); ok {
		payload["max_tokens"] = int(maxTokens)
	}

	if temperature, ok := input["temperature"].(float64); ok {
		payload["temperature"] = temperature
	}

	url := fmt.Sprintf("%s/sampling/stream", serverURL)
	return o.makePostRequest(ctx, url, payload, input)
}

func (o *OpenMCPProxyTool) healthCheck(ctx tools.ExecutionContext, serverURL string, input map[string]interface{}) *tools.Result {
	url := fmt.Sprintf("%s/health", serverURL)
	return o.makeGetRequest(ctx, url, input)
}

func (o *OpenMCPProxyTool) makeGetRequest(ctx tools.ExecutionContext, url string, input map[string]interface{}) *tools.Result {
	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "GET", url, nil)
	if err != nil {
		return tools.ErrorResult("REQUEST_CREATION_FAILED", fmt.Sprintf("Failed to create request: %v", err))
	}

	return o.executeRequest(req, input)
}

func (o *OpenMCPProxyTool) makePostRequest(ctx tools.ExecutionContext, url string, payload map[string]interface{}, input map[string]interface{}) *tools.Result {
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

	req.Header.Set("Content-Type", "application/json")
	return o.executeRequest(req, input)
}

func (o *OpenMCPProxyTool) executeRequest(req *http.Request, input map[string]interface{}) *tools.Result {
	// Set common headers
	req.Header.Set("User-Agent", "agent-server/1.0")
	req.Header.Set("Accept", "application/json")

	// Add API key if provided
	if apiKey, ok := input["api_key"].(string); ok && apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	// Execute request
	resp, err := o.client.Do(req)
	if err != nil {
		return tools.ErrorResult("REQUEST_FAILED", fmt.Sprintf("OpenMCP request failed: %v", err))
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult("RESPONSE_READ_FAILED", fmt.Sprintf("Failed to read response: %v", err))
	}

	// Handle non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errorResponse map[string]interface{}
		if json.Unmarshal(body, &errorResponse) == nil {
			if errorMsg, ok := errorResponse["error"].(string); ok {
				return tools.ErrorResult("OPENMCP_ERROR", errorMsg, map[string]interface{}{
					"status_code": resp.StatusCode,
					"response":    errorResponse,
				})
			}
		}
		
		return tools.ErrorResult("HTTP_ERROR", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)), map[string]interface{}{
			"status_code": resp.StatusCode,
		})
	}

	// Parse JSON response
	var response interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		// If not JSON, return as string
		response = string(body)
	}

	return tools.SuccessResult(response, map[string]interface{}{
		"status_code":   resp.StatusCode,
		"server_url":    input["server_url"],
		"content_type":  resp.Header.Get("Content-Type"),
		"response_size": len(body),
	})
}

// IsAvailable checks if the OpenMCP proxy tool is available
func (o *OpenMCPProxyTool) IsAvailable(ctx context.Context) bool {
	// The tool is available if we can make HTTP requests
	return true
}