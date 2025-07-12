package builtin

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"agent-server/internal/storage"
	"agent-server/internal/tools"
)

// CalculatorTool provides basic arithmetic calculations
type CalculatorTool struct {
	*tools.BaseTool
}

// NewCalculatorTool creates a new calculator tool
func NewCalculatorTool() *CalculatorTool {
	schema := tools.Schema{
		Name:        "calculator",
		Description: "Performs basic arithmetic calculations and mathematical operations",
		Parameters: []tools.Parameter{
			{
				Name:        "expression",
				Type:        "string",
				Description: "Mathematical expression to evaluate (supports +, -, *, /, ^, sqrt, abs)",
				Required:    true,
				Pattern:     `^[0-9+\-*/().\s^%a-z,]+$`,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Simple arithmetic",
				Input:       map[string]interface{}{"expression": "2 + 3 * 4"},
				Output:      map[string]interface{}{"result": 14, "expression": "2 + 3 * 4"},
			},
			{
				Description: "Square root calculation",
				Input:       map[string]interface{}{"expression": "sqrt(16)"},
				Output:      map[string]interface{}{"result": 4, "expression": "sqrt(16)"},
			},
		},
	}

	tool := &CalculatorTool{}
	tool.BaseTool = tools.NewBaseTool("calculator", schema, tool.execute)
	
	return tool
}

func (c *CalculatorTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	expression := input["expression"].(string)
	
	// Simple expression evaluator (basic implementation)
	result, err := evaluateExpression(expression)
	if err != nil {
		return tools.ErrorResult("CALCULATION_ERROR", fmt.Sprintf("Failed to evaluate expression: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"result":     result,
		"expression": expression,
	})
}

// TextProcessorTool provides text manipulation capabilities
type TextProcessorTool struct {
	*tools.BaseTool
}

// NewTextProcessorTool creates a new text processor tool
func NewTextProcessorTool() *TextProcessorTool {
	schema := tools.Schema{
		Name:        "text_processor",
		Description: "Processes and manipulates text with various operations",
		Parameters: []tools.Parameter{
			{
				Name:        "text",
				Type:        "string",
				Description: "The text to process",
				Required:    true,
			},
			{
				Name:        "operation",
				Type:        "string",
				Description: "Text operation to perform",
				Required:    true,
				Enum:        []string{"uppercase", "lowercase", "title_case", "word_count", "char_count", "reverse", "trim", "extract_emails", "extract_urls"},
			},
			{
				Name:        "pattern",
				Type:        "string",
				Description: "Pattern for regex operations (optional)",
				Required:    false,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Convert text to uppercase",
				Input: map[string]interface{}{
					"text":      "hello world",
					"operation": "uppercase",
				},
				Output: map[string]interface{}{
					"result":    "HELLO WORLD",
					"operation": "uppercase",
				},
			},
			{
				Description: "Count words in text",
				Input: map[string]interface{}{
					"text":      "The quick brown fox",
					"operation": "word_count",
				},
				Output: map[string]interface{}{
					"result":    4,
					"operation": "word_count",
				},
			},
		},
	}

	tool := &TextProcessorTool{}
	tool.BaseTool = tools.NewBaseTool("text_processor", schema, tool.execute)
	
	return tool
}

func (t *TextProcessorTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	text := input["text"].(string)
	operation := input["operation"].(string)

	var result interface{}
	var err error

	switch operation {
	case "uppercase":
		result = strings.ToUpper(text)
	case "lowercase":
		result = strings.ToLower(text)
	case "title_case":
		result = strings.Title(strings.ToLower(text))
	case "word_count":
		words := strings.Fields(text)
		result = len(words)
	case "char_count":
		result = len(text)
	case "reverse":
		result = reverseString(text)
	case "trim":
		result = strings.TrimSpace(text)
	case "extract_emails":
		result, err = extractEmails(text)
	case "extract_urls":
		result, err = extractURLs(text)
	default:
		return tools.ErrorResult("INVALID_OPERATION", fmt.Sprintf("Unknown operation: %s", operation))
	}

	if err != nil {
		return tools.ErrorResult("PROCESSING_ERROR", fmt.Sprintf("Failed to process text: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"result":    result,
		"operation": operation,
		"original":  text,
	})
}

// JSONProcessorTool provides JSON manipulation capabilities
type JSONProcessorTool struct {
	*tools.BaseTool
}

// NewJSONProcessorTool creates a new JSON processor tool
func NewJSONProcessorTool() *JSONProcessorTool {
	schema := tools.Schema{
		Name:        "json_processor",
		Description: "Processes and manipulates JSON data",
		Parameters: []tools.Parameter{
			{
				Name:        "json_data",
				Type:        "string",
				Description: "JSON string to process",
				Required:    true,
			},
			{
				Name:        "operation",
				Type:        "string",
				Description: "JSON operation to perform",
				Required:    true,
				Enum:        []string{"validate", "pretty_print", "minify", "extract_keys", "get_value"},
			},
			{
				Name:        "path",
				Type:        "string",
				Description: "JSON path for get_value operation (e.g., 'user.name')",
				Required:    false,
			},
		},
		Examples: []tools.Example{
			{
				Description: "Pretty print JSON",
				Input: map[string]interface{}{
					"json_data": `{"name":"John","age":30}`,
					"operation": "pretty_print",
				},
				Output: map[string]interface{}{
					"result": "{\n  \"name\": \"John\",\n  \"age\": 30\n}",
				},
			},
		},
	}

	tool := &JSONProcessorTool{}
	tool.BaseTool = tools.NewBaseTool("json_processor", schema, tool.execute)
	
	return tool
}

func (j *JSONProcessorTool) execute(ctx tools.ExecutionContext, input map[string]interface{}) *tools.Result {
	jsonData := input["json_data"].(string)
	operation := input["operation"].(string)

	// Parse JSON first
	var data interface{}
	if err := json.Unmarshal([]byte(jsonData), &data); err != nil {
		return tools.ErrorResult("INVALID_JSON", fmt.Sprintf("Invalid JSON: %v", err))
	}

	var result interface{}
	var err error

	switch operation {
	case "validate":
		result = map[string]interface{}{
			"valid":   true,
			"message": "JSON is valid",
		}
	case "pretty_print":
		var prettyJSON []byte
		prettyJSON, err = json.MarshalIndent(data, "", "  ")
		if err == nil {
			result = string(prettyJSON)
		}
	case "minify":
		var minifiedJSON []byte
		minifiedJSON, err = json.Marshal(data)
		if err == nil {
			result = string(minifiedJSON)
		}
	case "extract_keys":
		result = extractJSONKeys(data)
	case "get_value":
		path, ok := input["path"].(string)
		if !ok {
			return tools.ErrorResult("MISSING_PATH", "path is required for get_value operation")
		}
		result, err = getJSONValue(data, path)
	default:
		return tools.ErrorResult("INVALID_OPERATION", fmt.Sprintf("Unknown operation: %s", operation))
	}

	if err != nil {
		return tools.ErrorResult("PROCESSING_ERROR", fmt.Sprintf("Failed to process JSON: %v", err))
	}

	return tools.SuccessResult(map[string]interface{}{
		"result":    result,
		"operation": operation,
	})
}

// Helper functions

func evaluateExpression(expr string) (float64, error) {
	// Enhanced expression evaluator with support for basic operations
	// Trim leading/trailing spaces but preserve spaces around operators for splitting
	expr = strings.TrimSpace(expr)
	
	// Handle functions first (these may contain spaces)
	if strings.HasPrefix(expr, "sqrt(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[5 : len(expr)-1])
		val, err := strconv.ParseFloat(inner, 64)
		if err != nil {
			return 0, err
		}
		return math.Sqrt(val), nil
	}
	
	if strings.HasPrefix(expr, "abs(") && strings.HasSuffix(expr, ")") {
		inner := strings.TrimSpace(expr[4 : len(expr)-1])
		val, err := strconv.ParseFloat(inner, 64)
		if err != nil {
			return 0, err
		}
		return math.Abs(val), nil
	}
	
	// Handle power operations (^)
	if strings.Contains(expr, "^") {
		parts := strings.Split(expr, "^")
		if len(parts) == 2 {
			base, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			exp, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return math.Pow(base, exp), nil
			}
		}
	}
	
	// Handle multiplication (*) - but avoid splitting negative numbers
	if strings.Contains(expr, "*") && !strings.HasPrefix(expr, "-") {
		parts := strings.Split(expr, "*")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a * b, nil
			}
		}
	}
	
	// Handle division (/)
	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				if b == 0 {
					return 0, fmt.Errorf("division by zero")
				}
				return a / b, nil
			}
		}
	}
	
	// Handle addition (+)
	if strings.Contains(expr, "+") {
		parts := strings.Split(expr, "+")
		if len(parts) == 2 {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a + b, nil
			}
		}
	}
	
	// Handle subtraction (-) - be careful with negative numbers
	if strings.Contains(expr, "-") && !strings.HasPrefix(expr, "-") {
		parts := strings.Split(expr, "-")
		if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
			a, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			b, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
			if err1 == nil && err2 == nil {
				return a - b, nil
			}
		}
	}
	
	// Try to parse as a single number (including negative numbers)
	return strconv.ParseFloat(strings.TrimSpace(expr), 64)
}

func reverseString(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func extractEmails(text string) ([]string, error) {
	emailRegex := regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`)
	return emailRegex.FindAllString(text, -1), nil
}

func extractURLs(text string) ([]string, error) {
	urlRegex := regexp.MustCompile(`https?://[^\s]+`)
	return urlRegex.FindAllString(text, -1), nil
}

func extractJSONKeys(data interface{}) []string {
	var keys []string
	if obj, ok := data.(map[string]interface{}); ok {
		for key := range obj {
			keys = append(keys, key)
		}
	}
	return keys
}

func getJSONValue(data interface{}, path string) (interface{}, error) {
	keys := strings.Split(path, ".")
	current := data
	
	for _, key := range keys {
		if obj, ok := current.(map[string]interface{}); ok {
			if value, exists := obj[key]; exists {
				current = value
			} else {
				return nil, fmt.Errorf("key '%s' not found", key)
			}
		} else {
			return nil, fmt.Errorf("cannot access key '%s' on non-object", key)
		}
	}
	
	return current, nil
}

// RegisterBuiltinTools registers all built-in tools with the registry
func RegisterBuiltinTools(registry *tools.Registry, memoryRepo storage.MemoryRepository) error {
	builtinTools := []tools.Tool{
		NewHTTPGetTool(),
		NewHTTPPostTool(),
		NewWebScraperTool(),
		NewCalculatorTool(),
		NewTextProcessorTool(),
		NewJSONProcessorTool(),
		NewMCPProxyTool(),
		NewOpenMCPProxyTool(),
		NewMemoryTool(memoryRepo),
	}

	for _, tool := range builtinTools {
		if err := registry.Register(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}

	return nil
}