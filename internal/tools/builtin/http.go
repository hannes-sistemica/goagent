package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-server/internal/tools"
)

// HTTPGetTool provides HTTP GET functionality
type HTTPGetTool struct {
	*tools.HTTPBaseTool
	client *http.Client
}

// NewHTTPGetTool creates a new HTTP GET tool
func NewHTTPGetTool() *HTTPGetTool {
	schema := tools.Schema{
		Name:        "http_get",
		Description: "Performs an HTTP GET request to retrieve data from a URL",
		Parameters: []tools.Parameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to send the GET request to",
				Required:    true,
				Pattern:     `^https?://.*`,
			},
			{
				Name:        "headers",
				Type:        "object",
				Description: "Optional HTTP headers to include in the request",
				Required:    false,
			},
			{
				Name:        "timeout",
				Type:        "number",
				Description: "Request timeout in seconds (default: 30)",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
				Maximum:     func() *float64 { v := 300.0; return &v }(),
				Default:     30,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Get JSON data from an API",
				Input: map[string]interface{}{
					"url": "https://api.example.com/data",
					"headers": map[string]interface{}{
						"Accept": "application/json",
					},
				},
				Output: map[string]interface{}{
					"status_code": 200,
					"data":        "response body",
					"headers":     map[string]interface{}{},
				},
			},
		},
	}

	tool := &HTTPGetTool{
		HTTPBaseTool: tools.NewHTTPBaseTool("http_get", schema, "", 30*time.Second),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set the execute function
	tool.BaseTool = tools.NewBaseTool("http_get", schema, tool.execute)
	
	return tool
}

func (h *HTTPGetTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	// Extract parameters
	urlStr := input["url"].(string)
	
	// Validate URL
	if _, err := url.Parse(urlStr); err != nil {
		return tools.ErrorResult("INVALID_URL", fmt.Sprintf("Invalid URL: %v", err))
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "GET", urlStr, nil)
	if err != nil {
		return tools.ErrorResult("REQUEST_CREATION_FAILED", fmt.Sprintf("Failed to create request: %v", err))
	}

	// Add headers
	if headers, ok := input["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}

	// Set timeout if provided
	timeout := 30 * time.Second
	if timeoutVal, ok := input["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
		h.client.Timeout = timeout
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return tools.ErrorResult("REQUEST_FAILED", fmt.Sprintf("HTTP request failed: %v", err))
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult("RESPONSE_READ_FAILED", fmt.Sprintf("Failed to read response: %v", err))
	}

	// Prepare response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		responseHeaders[key] = strings.Join(values, ", ")
	}

	// Try to parse JSON if content type suggests it
	var responseData interface{}
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(body, &jsonData); err == nil {
			responseData = jsonData
		} else {
			responseData = string(body)
		}
	} else {
		responseData = string(body)
	}

	return tools.SuccessResult(map[string]interface{}{
		"status_code": resp.StatusCode,
		"data":        responseData,
		"headers":     responseHeaders,
		"content_type": contentType,
	}, map[string]interface{}{
		"url":           urlStr,
		"response_size": len(body),
	})
}

// HTTPPostTool provides HTTP POST functionality
type HTTPPostTool struct {
	*tools.HTTPBaseTool
	client *http.Client
}

// NewHTTPPostTool creates a new HTTP POST tool
func NewHTTPPostTool() *HTTPPostTool {
	schema := tools.Schema{
		Name:        "http_post",
		Description: "Performs an HTTP POST request to send data to a URL",
		Parameters: []tools.Parameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL to send the POST request to",
				Required:    true,
				Pattern:     `^https?://.*`,
			},
			{
				Name:        "data",
				Type:        "object",
				Description: "The data to send in the request body (will be JSON encoded)",
				Required:    false,
			},
			{
				Name:        "headers",
				Type:        "object",
				Description: "Optional HTTP headers to include in the request",
				Required:    false,
			},
			{
				Name:        "content_type",
				Type:        "string",
				Description: "Content-Type header (default: application/json)",
				Required:    false,
				Default:     "application/json",
			},
			{
				Name:        "timeout",
				Type:        "number",
				Description: "Request timeout in seconds (default: 30)",
				Required:    false,
				Minimum:     func() *float64 { v := 1.0; return &v }(),
				Maximum:     func() *float64 { v := 300.0; return &v }(),
				Default:     30,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Post JSON data to an API",
				Input: map[string]interface{}{
					"url": "https://api.example.com/users",
					"data": map[string]interface{}{
						"name":  "John Doe",
						"email": "john@example.com",
					},
					"headers": map[string]interface{}{
						"Authorization": "Bearer token123",
					},
				},
				Output: map[string]interface{}{
					"status_code": 201,
					"data":        map[string]interface{}{"id": 123},
				},
			},
		},
	}

	tool := &HTTPPostTool{
		HTTPBaseTool: tools.NewHTTPBaseTool("http_post", schema, "", 30*time.Second),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set the execute function
	tool.BaseTool = tools.NewBaseTool("http_post", schema, tool.execute)
	
	return tool
}

func (h *HTTPPostTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	// Extract parameters
	urlStr := input["url"].(string)
	
	// Validate URL
	if _, err := url.Parse(urlStr); err != nil {
		return tools.ErrorResult("INVALID_URL", fmt.Sprintf("Invalid URL: %v", err))
	}

	// Prepare request body
	var body io.Reader
	var contentType string = "application/json"
	
	if ctVal, ok := input["content_type"].(string); ok {
		contentType = ctVal
	}

	if data, ok := input["data"]; ok && data != nil {
		if contentType == "application/json" {
			jsonData, err := json.Marshal(data)
			if err != nil {
				return tools.ErrorResult("JSON_ENCODING_FAILED", fmt.Sprintf("Failed to encode JSON: %v", err))
			}
			body = bytes.NewReader(jsonData)
		} else {
			// For other content types, convert to string
			body = strings.NewReader(fmt.Sprintf("%v", data))
		}
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx.Context, "POST", urlStr, body)
	if err != nil {
		return tools.ErrorResult("REQUEST_CREATION_FAILED", fmt.Sprintf("Failed to create request: %v", err))
	}

	// Set content type
	req.Header.Set("Content-Type", contentType)

	// Add additional headers
	if headers, ok := input["headers"].(map[string]interface{}); ok {
		for key, value := range headers {
			req.Header.Set(key, fmt.Sprintf("%v", value))
		}
	}

	// Set timeout if provided
	timeout := 30 * time.Second
	if timeoutVal, ok := input["timeout"].(float64); ok {
		timeout = time.Duration(timeoutVal) * time.Second
		h.client.Timeout = timeout
	}

	// Execute request
	resp, err := h.client.Do(req)
	if err != nil {
		return tools.ErrorResult("REQUEST_FAILED", fmt.Sprintf("HTTP request failed: %v", err))
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult("RESPONSE_READ_FAILED", fmt.Sprintf("Failed to read response: %v", err))
	}

	// Prepare response headers
	responseHeaders := make(map[string]string)
	for key, values := range resp.Header {
		responseHeaders[key] = strings.Join(values, ", ")
	}

	// Try to parse JSON if content type suggests it
	var responseData interface{}
	respContentType := resp.Header.Get("Content-Type")
	if strings.Contains(respContentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(respBody, &jsonData); err == nil {
			responseData = jsonData
		} else {
			responseData = string(respBody)
		}
	} else {
		responseData = string(respBody)
	}

	return tools.SuccessResult(map[string]interface{}{
		"status_code": resp.StatusCode,
		"data":        responseData,
		"headers":     responseHeaders,
		"content_type": respContentType,
	}, map[string]interface{}{
		"url":           urlStr,
		"response_size": len(respBody),
	})
}

// WebScraperTool provides basic web scraping functionality
type WebScraperTool struct {
	*tools.HTTPBaseTool
	client *http.Client
}

// NewWebScraperTool creates a new web scraper tool
func NewWebScraperTool() *WebScraperTool {
	schema := tools.Schema{
		Name:        "web_scraper",
		Description: "Scrapes text content from web pages",
		Parameters: []tools.Parameter{
			{
				Name:        "url",
				Type:        "string",
				Description: "The URL of the web page to scrape",
				Required:    true,
				Pattern:     `^https?://.*`,
			},
			{
				Name:        "selector",
				Type:        "string",
				Description: "CSS selector to extract specific content (optional)",
				Required:    false,
			},
			{
				Name:        "max_length",
				Type:        "number",
				Description: "Maximum length of extracted text (default: 10000)",
				Required:    false,
				Minimum:     func() *float64 { v := 100.0; return &v }(),
				Maximum:     func() *float64 { v := 100000.0; return &v }(),
				Default:     10000,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Scrape title and content from a webpage",
				Input: map[string]interface{}{
					"url":        "https://example.com/article",
					"selector":   "article",
					"max_length": 5000,
				},
				Output: map[string]interface{}{
					"title":   "Article Title",
					"content": "Article content...",
					"url":     "https://example.com/article",
				},
			},
		},
	}

	tool := &WebScraperTool{
		HTTPBaseTool: tools.NewHTTPBaseTool("web_scraper", schema, "", 30*time.Second),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Set the execute function
	tool.BaseTool = tools.NewBaseTool("web_scraper", schema, tool.execute)
	
	return tool
}

func (w *WebScraperTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	// Extract parameters
	urlStr := input["url"].(string)
	
	// Validate URL
	if _, err := url.Parse(urlStr); err != nil {
		return tools.ErrorResult("INVALID_URL", fmt.Sprintf("Invalid URL: %v", err))
	}

	// Create request with user agent
	req, err := http.NewRequestWithContext(ctx.Context, "GET", urlStr, nil)
	if err != nil {
		return tools.ErrorResult("REQUEST_CREATION_FAILED", fmt.Sprintf("Failed to create request: %v", err))
	}

	req.Header.Set("User-Agent", "AgentServer/1.0 (+https://github.com/agent-server)")

	// Execute request
	resp, err := w.client.Do(req)
	if err != nil {
		return tools.ErrorResult("REQUEST_FAILED", fmt.Sprintf("HTTP request failed: %v", err))
	}
	defer resp.Body.Close()

	// Check if response is HTML
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		return tools.ErrorResult("NOT_HTML", "URL does not return HTML content")
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return tools.ErrorResult("RESPONSE_READ_FAILED", fmt.Sprintf("Failed to read response: %v", err))
	}

	// Basic text extraction (simplified - in production, use a proper HTML parser)
	content := string(body)
	
	// Extract title
	title := "No title found"
	if titleStart := strings.Index(content, "<title>"); titleStart != -1 {
		titleStart += 7
		if titleEnd := strings.Index(content[titleStart:], "</title>"); titleEnd != -1 {
			title = content[titleStart : titleStart+titleEnd]
		}
	}

	// Simple text extraction (remove HTML tags)
	text := stripHTMLTags(content)
	
	// Apply max length
	maxLength := 10000
	if maxLenVal, ok := input["max_length"].(float64); ok {
		maxLength = int(maxLenVal)
	}
	
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	return tools.SuccessResult(map[string]interface{}{
		"title":   title,
		"content": text,
		"url":     urlStr,
		"length":  len(text),
	}, map[string]interface{}{
		"status_code":    resp.StatusCode,
		"content_type":   contentType,
		"original_size":  len(body),
	})
}

// stripHTMLTags removes HTML tags from text (basic implementation)
func stripHTMLTags(html string) string {
	// This is a very basic implementation
	// In production, use a proper HTML parser like golang.org/x/net/html
	result := ""
	inTag := false
	
	for _, char := range html {
		if char == '<' {
			inTag = true
		} else if char == '>' {
			inTag = false
		} else if !inTag {
			result += string(char)
		}
	}
	
	// Clean up whitespace
	result = strings.ReplaceAll(result, "\n", " ")
	result = strings.ReplaceAll(result, "\t", " ")
	for strings.Contains(result, "  ") {
		result = strings.ReplaceAll(result, "  ", " ")
	}
	
	return strings.TrimSpace(result)
}