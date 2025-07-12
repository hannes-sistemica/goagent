package builtin_test

import (
	"context"
	"testing"

	"agent-server/internal/storage/sqlite"
	"agent-server/internal/tools"
	"agent-server/internal/tools/builtin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCalculatorTool(t *testing.T) {
	calculator := builtin.NewCalculatorTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "calculator", calculator.Name())
		
		schema := calculator.Schema()
		assert.Equal(t, "calculator", schema.Name)
		assert.Contains(t, schema.Description, "arithmetic")
		assert.Len(t, schema.Parameters, 1)
		assert.Equal(t, "expression", schema.Parameters[0].Name)
		assert.Equal(t, "string", schema.Parameters[0].Type)
		assert.True(t, schema.Parameters[0].Required)
	})

	t.Run("IsAvailable", func(t *testing.T) {
		ctx := context.Background()
		assert.True(t, calculator.IsAvailable(ctx))
	})

	t.Run("Valid Expression", func(t *testing.T) {
		input := map[string]interface{}{
			"expression": "2 + 3",
		}

		err := calculator.Validate(input)
		assert.NoError(t, err)
	})

	t.Run("Invalid Expression - Missing", func(t *testing.T) {
		input := map[string]interface{}{}

		err := calculator.Validate(input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required parameter missing")
	})

	t.Run("Execute Simple Addition", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"expression": "2 + 3",
		}

		result := calculator.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 5.0, data["result"])
		assert.Equal(t, "2 + 3", data["expression"])
	})

	t.Run("Execute Square Root", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"expression": "sqrt(16)",
		}

		result := calculator.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 4.0, data["result"])
	})

	t.Run("Execute Invalid Expression", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"expression": "invalid expression",
		}

		result := calculator.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Failed to evaluate expression")
	})
}

func TestTextProcessorTool(t *testing.T) {
	processor := builtin.NewTextProcessorTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "text_processor", processor.Name())
		
		schema := processor.Schema()
		assert.Equal(t, "text_processor", schema.Name)
		assert.Contains(t, schema.Description, "text")
		assert.Len(t, schema.Parameters, 3)
		
		// Check parameters
		params := make(map[string]tools.Parameter)
		for _, p := range schema.Parameters {
			params[p.Name] = p
		}
		
		assert.True(t, params["text"].Required)
		assert.True(t, params["operation"].Required)
		assert.False(t, params["pattern"].Required)
		assert.Len(t, params["operation"].Enum, 9) // 9 operations
	})

	t.Run("Uppercase Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "hello world",
			"operation": "uppercase",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "HELLO WORLD", data["result"])
		assert.Equal(t, "uppercase", data["operation"])
		assert.Equal(t, "hello world", data["original"])
	})

	t.Run("Word Count Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "The quick brown fox",
			"operation": "word_count",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 4, data["result"])
	})

	t.Run("Character Count Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "hello",
			"operation": "char_count",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, 5, data["result"])
	})

	t.Run("Reverse Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "hello",
			"operation": "reverse",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "olleh", data["result"])
	})

	t.Run("Extract Emails Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "Contact us at info@example.com or support@test.org",
			"operation": "extract_emails",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		emails, ok := data["result"].([]string)
		assert.True(t, ok)
		assert.Len(t, emails, 2)
		assert.Contains(t, emails, "info@example.com")
		assert.Contains(t, emails, "support@test.org")
	})

	t.Run("Extract URLs Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "Visit https://example.com or http://test.org for more info",
			"operation": "extract_urls",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		urls, ok := data["result"].([]string)
		assert.True(t, ok)
		assert.Len(t, urls, 2)
		assert.Contains(t, urls, "https://example.com")
		assert.Contains(t, urls, "http://test.org")
	})

	t.Run("Invalid Operation", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"text":      "hello",
			"operation": "invalid_operation",
		}

		result := processor.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Unknown operation")
	})
}

func TestJSONProcessorTool(t *testing.T) {
	processor := builtin.NewJSONProcessorTool()

	t.Run("Tool Metadata", func(t *testing.T) {
		assert.Equal(t, "json_processor", processor.Name())
		
		schema := processor.Schema()
		assert.Equal(t, "json_processor", schema.Name)
		assert.Contains(t, schema.Description, "JSON")
		assert.Len(t, schema.Parameters, 3)
	})

	t.Run("Validate JSON", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"name": "John", "age": 30}`,
			"operation": "validate",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		resultMap, ok := data["result"].(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, true, resultMap["valid"])
	})

	t.Run("Pretty Print JSON", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"name":"John","age":30}`,
			"operation": "pretty_print",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		prettyJSON, ok := data["result"].(string)
		assert.True(t, ok)
		assert.Contains(t, prettyJSON, "  \"name\": \"John\"")
		assert.Contains(t, prettyJSON, "  \"age\": 30")
	})

	t.Run("Minify JSON", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{
  "name": "John",
  "age": 30
}`,
			"operation": "minify",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		minifiedJSON, ok := data["result"].(string)
		assert.True(t, ok)
		assert.Equal(t, `{"age":30,"name":"John"}`, minifiedJSON)
	})

	t.Run("Extract Keys", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"name": "John", "age": 30, "city": "New York"}`,
			"operation": "extract_keys",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		
		keys, ok := data["result"].([]string)
		assert.True(t, ok)
		assert.Len(t, keys, 3)
		assert.Contains(t, keys, "name")
		assert.Contains(t, keys, "age")
		assert.Contains(t, keys, "city")
	})

	t.Run("Get Value by Path", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"user": {"name": "John", "age": 30}}`,
			"operation": "get_value",
			"path":      "user.name",
		}

		result := processor.Execute(ctx, input)
		assert.True(t, result.Success)
		
		data, ok := result.Data.(map[string]interface{})
		assert.True(t, ok)
		assert.Equal(t, "John", data["result"])
	})

	t.Run("Get Value Missing Path", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"name": "John"}`,
			"operation": "get_value",
		}

		result := processor.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "path is required")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		ctx := tools.ExecutionContext{
			Context:   context.Background(),
			SessionID: "test-session",
		}

		input := map[string]interface{}{
			"json_data": `{"name": John"}`, // Invalid JSON
			"operation": "validate",
		}

		result := processor.Execute(ctx, input)
		assert.False(t, result.Success)
		assert.Contains(t, result.Error, "Invalid JSON")
	})
}

func TestRegisterBuiltinTools(t *testing.T) {
	t.Run("Register All Built-in Tools", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Create in-memory repository for memory tool
		repo, err := sqlite.NewRepository(":memory:")
		require.NoError(t, err)
		
		err = builtin.RegisterBuiltinTools(registry, repo.Memory())
		require.NoError(t, err)
		
		// Should have 9 built-in tools (including memory)
		assert.Equal(t, 9, registry.Count())
		
		// Check that all expected tools are registered
		expectedTools := []string{
			"http_get",
			"http_post", 
			"web_scraper",
			"calculator",
			"text_processor",
			"json_processor",
			"mcp_proxy",
			"openmcp_proxy",
			"memory",
		}
		
		registeredTools := registry.List()
		for _, expected := range expectedTools {
			assert.Contains(t, registeredTools, expected, "Tool %s should be registered", expected)
		}
		
		// Verify each tool is available
		ctx := context.Background()
		for _, toolName := range expectedTools {
			tool, exists := registry.Get(toolName)
			assert.True(t, exists, "Tool %s should exist", toolName)
			assert.True(t, tool.IsAvailable(ctx), "Tool %s should be available", toolName)
		}
	})

	t.Run("Tool Schemas are Valid", func(t *testing.T) {
		registry := tools.NewRegistry()
		
		// Create in-memory repository for memory tool
		repo, err := sqlite.NewRepository(":memory:")
		require.NoError(t, err)
		
		err = builtin.RegisterBuiltinTools(registry, repo.Memory())
		require.NoError(t, err)
		
		for _, toolName := range registry.List() {
			tool, _ := registry.Get(toolName)
			schema := tool.Schema()
			
			// Basic schema validation
			assert.NotEmpty(t, schema.Name, "Tool %s should have a name", toolName)
			assert.NotEmpty(t, schema.Description, "Tool %s should have a description", toolName)
			assert.Equal(t, toolName, schema.Name, "Tool name should match schema name")
			
			// Parameter validation
			for _, param := range schema.Parameters {
				assert.NotEmpty(t, param.Name, "Parameter should have a name")
				assert.NotEmpty(t, param.Type, "Parameter should have a type")
				assert.Contains(t, []string{"string", "number", "boolean", "object", "array"}, 
					param.Type, "Parameter type should be valid")
			}
		}
	})
}