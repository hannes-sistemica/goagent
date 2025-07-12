package tools_test

import (
	"testing"

	"agent-server/internal/tools"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateInput(t *testing.T) {
	schema := tools.Schema{
		Name:        "test_tool",
		Description: "A test tool",
		Parameters: []tools.Parameter{
			{
				Name:        "required_string",
				Type:        "string",
				Description: "A required string parameter",
				Required:    true,
			},
			{
				Name:        "optional_number",
				Type:        "number",
				Description: "An optional number parameter",
				Required:    false,
				Minimum:     func() *float64 { v := 0.0; return &v }(),
				Maximum:     func() *float64 { v := 100.0; return &v }(),
			},
			{
				Name:        "enum_string",
				Type:        "string",
				Description: "A string with enum values",
				Required:    false,
				Enum:        []string{"option1", "option2", "option3"},
			},
			{
				Name:        "pattern_string",
				Type:        "string",
				Description: "A string with pattern validation",
				Required:    false,
				Pattern:     "^[a-z]+$",
			},
			{
				Name:        "boolean_param",
				Type:        "boolean",
				Description: "A boolean parameter",
				Required:    false,
			},
			{
				Name:        "object_param",
				Type:        "object",
				Description: "An object parameter",
				Required:    false,
			},
			{
				Name:        "array_param",
				Type:        "array",
				Description: "An array parameter",
				Required:    false,
			},
		},
	}

	t.Run("Valid Input", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"optional_number": 50.0,
			"enum_string":     "option2",
			"pattern_string":  "hello",
			"boolean_param":   true,
			"object_param":    map[string]interface{}{"key": "value"},
			"array_param":     []interface{}{1, 2, 3},
		}

		err := tools.ValidateInput(schema, input)
		assert.NoError(t, err)
	})

	t.Run("Missing Required Parameter", func(t *testing.T) {
		input := map[string]interface{}{
			"optional_number": 50.0,
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required parameter missing")
		assert.Contains(t, err.Error(), "required_string")
	})

	t.Run("Unknown Parameter", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string":  "test",
			"unknown_parameter": "value",
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown parameter")
		assert.Contains(t, err.Error(), "unknown_parameter")
	})

	t.Run("Invalid String Type", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": 123, // Should be string
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected string value")
	})

	t.Run("Number Out of Range", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"optional_number": 150.0, // Exceeds maximum of 100
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be <= 100")
	})

	t.Run("Number Below Minimum", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"optional_number": -10.0, // Below minimum of 0
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be >= 0")
	})

	t.Run("Invalid Enum Value", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"enum_string":     "invalid_option",
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be one of")
		assert.Contains(t, err.Error(), "option1, option2, option3")
	})

	t.Run("Pattern Mismatch", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"pattern_string":  "Hello123", // Contains uppercase and numbers
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not match pattern")
	})

	t.Run("Invalid Boolean Type", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"boolean_param":   "not_a_boolean",
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected boolean value")
	})

	t.Run("Valid Boolean String Representations", func(t *testing.T) {
		testCases := []string{"true", "false", "TRUE", "FALSE"}
		
		for _, boolStr := range testCases {
			input := map[string]interface{}{
				"required_string": "test",
				"boolean_param":   boolStr,
			}

			err := tools.ValidateInput(schema, input)
			assert.NoError(t, err, "Boolean string %s should be valid", boolStr)
		}
	})

	t.Run("Number Type Conversions", func(t *testing.T) {
		testCases := []interface{}{
			50,      // int
			50.0,    // float64
			float32(50.0), // float32
			int32(50),     // int32
			int64(50),     // int64
			"50",    // string that can be parsed
		}

		for _, numValue := range testCases {
			input := map[string]interface{}{
				"required_string": "test",
				"optional_number": numValue,
			}

			err := tools.ValidateInput(schema, input)
			assert.NoError(t, err, "Number value %v (%T) should be valid", numValue, numValue)
		}
	})

	t.Run("Invalid Object Type", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"object_param":    "not_an_object",
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected object value")
	})

	t.Run("Invalid Array Type", func(t *testing.T) {
		input := map[string]interface{}{
			"required_string": "test",
			"array_param":     "not_an_array",
		}

		err := tools.ValidateInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expected array value")
	})
}

func TestSanitizeInput(t *testing.T) {
	schema := tools.Schema{
		Parameters: []tools.Parameter{
			{
				Name:     "string_param",
				Type:     "string",
				Required: true,
			},
			{
				Name:     "number_param",
				Type:     "number",
				Required: false,
				Default:  42.0,
			},
			{
				Name:     "boolean_param",
				Type:     "boolean",
				Required: false,
			},
			{
				Name:     "optional_param",
				Type:     "string",
				Required: false,
			},
		},
	}

	t.Run("Sanitize with Defaults", func(t *testing.T) {
		input := map[string]interface{}{
			"string_param": "test",
		}

		sanitized, err := tools.SanitizeInput(schema, input)
		require.NoError(t, err)

		assert.Equal(t, "test", sanitized["string_param"])
		assert.Equal(t, 42.0, sanitized["number_param"]) // Default value applied
		assert.NotContains(t, sanitized, "boolean_param") // Optional, no default
		assert.NotContains(t, sanitized, "optional_param") // Optional, no default
	})

	t.Run("Type Conversion", func(t *testing.T) {
		input := map[string]interface{}{
			"string_param":  123,      // Should be converted to string
			"number_param":  "50",     // Should be converted to number
			"boolean_param": "true",   // Should be converted to boolean
		}

		sanitized, err := tools.SanitizeInput(schema, input)
		require.NoError(t, err)

		assert.Equal(t, "123", sanitized["string_param"])
		assert.Equal(t, 50.0, sanitized["number_param"])
		assert.Equal(t, true, sanitized["boolean_param"])
	})

	t.Run("Invalid Type Conversion", func(t *testing.T) {
		input := map[string]interface{}{
			"string_param": "test",
			"number_param": "not_a_number",
		}

		_, err := tools.SanitizeInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert to number")
	})

	t.Run("Invalid Boolean Conversion", func(t *testing.T) {
		input := map[string]interface{}{
			"string_param":  "test",
			"boolean_param": "not_a_boolean",
		}

		_, err := tools.SanitizeInput(schema, input)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot convert to boolean")
	})

	t.Run("Preserve Existing Values", func(t *testing.T) {
		input := map[string]interface{}{
			"string_param":  "test",
			"number_param":  99.5,
			"boolean_param": false,
		}

		sanitized, err := tools.SanitizeInput(schema, input)
		require.NoError(t, err)

		assert.Equal(t, "test", sanitized["string_param"])
		assert.Equal(t, 99.5, sanitized["number_param"])
		assert.Equal(t, false, sanitized["boolean_param"])
	})
}