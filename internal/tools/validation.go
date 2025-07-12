package tools

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// ValidateInput validates input parameters against a schema
func ValidateInput(schema Schema, input map[string]interface{}) error {
	// Check for required parameters
	for _, param := range schema.Parameters {
		if param.Required {
			if _, exists := input[param.Name]; !exists {
				return NewValidationError(param.Name, "required parameter missing", nil)
			}
		}
	}
	
	// Validate each provided parameter
	for key, value := range input {
		param := findParameter(schema.Parameters, key)
		if param == nil {
			return NewValidationError(key, "unknown parameter", value)
		}
		
		if err := validateParameter(*param, value); err != nil {
			return err
		}
	}
	
	return nil
}

// findParameter finds a parameter by name in the schema
func findParameter(parameters []Parameter, name string) *Parameter {
	for _, param := range parameters {
		if param.Name == name {
			return &param
		}
	}
	return nil
}

// validateParameter validates a single parameter value
func validateParameter(param Parameter, value interface{}) error {
	// Handle nil values
	if value == nil {
		if param.Required {
			return NewValidationError(param.Name, "required parameter cannot be nil", value)
		}
		return nil
	}
	
	switch param.Type {
	case "string":
		return validateStringParameter(param, value)
	case "number":
		return validateNumberParameter(param, value)
	case "boolean":
		return validateBooleanParameter(param, value)
	case "object":
		return validateObjectParameter(param, value)
	case "array":
		return validateArrayParameter(param, value)
	default:
		return NewValidationError(param.Name, fmt.Sprintf("unsupported parameter type: %s", param.Type), value)
	}
}

// validateStringParameter validates string parameters
func validateStringParameter(param Parameter, value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return NewValidationError(param.Name, "expected string value", value)
	}
	
	// Check enum values
	if len(param.Enum) > 0 {
		found := false
		for _, enumValue := range param.Enum {
			if str == enumValue {
				found = true
				break
			}
		}
		if !found {
			return NewValidationError(param.Name, 
				fmt.Sprintf("value must be one of: %s", strings.Join(param.Enum, ", ")), value)
		}
	}
	
	// Check pattern
	if param.Pattern != "" {
		matched, err := regexp.MatchString(param.Pattern, str)
		if err != nil {
			return NewValidationError(param.Name, fmt.Sprintf("invalid regex pattern: %s", param.Pattern), value)
		}
		if !matched {
			return NewValidationError(param.Name, fmt.Sprintf("value does not match pattern: %s", param.Pattern), value)
		}
	}
	
	return nil
}

// validateNumberParameter validates number parameters
func validateNumberParameter(param Parameter, value interface{}) error {
	var num float64
	var ok bool
	
	// Handle different number types
	switch v := value.(type) {
	case float64:
		num = v
		ok = true
	case float32:
		num = float64(v)
		ok = true
	case int:
		num = float64(v)
		ok = true
	case int32:
		num = float64(v)
		ok = true
	case int64:
		num = float64(v)
		ok = true
	case string:
		// Try to parse string as number
		if parsed, err := strconv.ParseFloat(v, 64); err == nil {
			num = parsed
			ok = true
		}
	}
	
	if !ok {
		return NewValidationError(param.Name, "expected number value", value)
	}
	
	// Check minimum
	if param.Minimum != nil && num < *param.Minimum {
		return NewValidationError(param.Name, fmt.Sprintf("value must be >= %g", *param.Minimum), value)
	}
	
	// Check maximum
	if param.Maximum != nil && num > *param.Maximum {
		return NewValidationError(param.Name, fmt.Sprintf("value must be <= %g", *param.Maximum), value)
	}
	
	return nil
}

// validateBooleanParameter validates boolean parameters
func validateBooleanParameter(param Parameter, value interface{}) error {
	switch v := value.(type) {
	case bool:
		return nil
	case string:
		// Allow string representations of booleans
		lower := strings.ToLower(v)
		if lower == "true" || lower == "false" {
			return nil
		}
	}
	
	return NewValidationError(param.Name, "expected boolean value", value)
}

// validateObjectParameter validates object parameters
func validateObjectParameter(param Parameter, value interface{}) error {
	// Check if it's a map or struct
	rv := reflect.ValueOf(value)
	if rv.Kind() == reflect.Map {
		return nil // Basic validation - could be extended
	}
	
	if rv.Kind() == reflect.Struct {
		return nil // Basic validation - could be extended
	}
	
	// Check if it's a map[string]interface{} from JSON
	if _, ok := value.(map[string]interface{}); ok {
		return nil
	}
	
	return NewValidationError(param.Name, "expected object value", value)
}

// validateArrayParameter validates array parameters
func validateArrayParameter(param Parameter, value interface{}) error {
	rv := reflect.ValueOf(value)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array {
		return NewValidationError(param.Name, "expected array value", value)
	}
	
	return nil
}

// SanitizeInput sanitizes and converts input parameters
func SanitizeInput(schema Schema, input map[string]interface{}) (map[string]interface{}, error) {
	sanitized := make(map[string]interface{})
	
	for _, param := range schema.Parameters {
		value, exists := input[param.Name]
		
		// Use default value if parameter is missing and has default
		if !exists && param.Default != nil {
			sanitized[param.Name] = param.Default
			continue
		}
		
		// Skip missing optional parameters
		if !exists && !param.Required {
			continue
		}
		
		// Skip missing required parameters (validation will catch this)
		if !exists {
			continue
		}
		
		// Convert and sanitize the value
		converted, err := convertValue(param.Type, value)
		if err != nil {
			return nil, NewValidationError(param.Name, err.Error(), value)
		}
		
		sanitized[param.Name] = converted
	}
	
	return sanitized, nil
}

// convertValue converts a value to the expected type
func convertValue(expectedType string, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, nil
	}
	
	switch expectedType {
	case "string":
		if str, ok := value.(string); ok {
			return str, nil
		}
		return fmt.Sprintf("%v", value), nil
		
	case "number":
		switch v := value.(type) {
		case float64, float32, int, int32, int64:
			return v, nil
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed, nil
			}
		}
		return nil, fmt.Errorf("cannot convert to number")
		
	case "boolean":
		switch v := value.(type) {
		case bool:
			return v, nil
		case string:
			lower := strings.ToLower(v)
			if lower == "true" {
				return true, nil
			}
			if lower == "false" {
				return false, nil
			}
		}
		return nil, fmt.Errorf("cannot convert to boolean")
		
	case "object", "array":
		return value, nil
		
	default:
		return value, nil
	}
}