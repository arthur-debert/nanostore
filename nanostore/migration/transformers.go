package migration

import (
	"fmt"
	"strconv"
	"strings"
)

// Transformer is a function that transforms a value
type Transformer func(value interface{}) (interface{}, error)

// TransformerRegistry maps transformer names to their implementations
var TransformerRegistry = map[string]Transformer{
	"toString":    ToString,
	"toInt":       ToInt,
	"toFloat":     ToFloat,
	"toBool":      ToBool,
	"toLowerCase": ToLowerCase,
	"toUpperCase": ToUpperCase,
	"trim":        Trim,
}

// ToString converts any value to string
func ToString(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}
	return fmt.Sprintf("%v", value), nil
}

// ToInt converts a value to integer
func ToInt(value interface{}) (interface{}, error) {
	if value == nil {
		return 0, nil
	}

	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case int32:
		return int(v), nil
	case float64:
		return int(v), nil
	case float32:
		return int(v), nil
	case string:
		return strconv.Atoi(strings.TrimSpace(v))
	case bool:
		if v {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("cannot convert %T to int", value)
	}
}

// ToFloat converts a value to float64
func ToFloat(value interface{}) (interface{}, error) {
	if value == nil {
		return 0.0, nil
	}

	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case int32:
		return float64(v), nil
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	case bool:
		if v {
			return 1.0, nil
		}
		return 0.0, nil
	default:
		return 0.0, fmt.Errorf("cannot convert %T to float", value)
	}
}

// ToBool converts a value to boolean
func ToBool(value interface{}) (interface{}, error) {
	if value == nil {
		return false, nil
	}

	switch v := value.(type) {
	case bool:
		return v, nil
	case string:
		s := strings.ToLower(strings.TrimSpace(v))
		switch s {
		case "true", "yes", "1", "on":
			return true, nil
		case "false", "no", "0", "off", "":
			return false, nil
		default:
			return false, fmt.Errorf("cannot convert %q to bool", v)
		}
	case int:
		return v != 0, nil
	case int64:
		return int64(v) != 0, nil
	case int32:
		return int32(v) != 0, nil
	case float64:
		return v != 0.0, nil
	case float32:
		return float32(v) != 0.0, nil
	default:
		return false, fmt.Errorf("cannot convert %T to bool", value)
	}
}

// ToLowerCase converts a string to lowercase
func ToLowerCase(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		// Convert to string first
		str = fmt.Sprintf("%v", value)
	}

	return strings.ToLower(str), nil
}

// ToUpperCase converts a string to uppercase
func ToUpperCase(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		// Convert to string first
		str = fmt.Sprintf("%v", value)
	}

	return strings.ToUpper(str), nil
}

// Trim removes leading and trailing whitespace from a string
func Trim(value interface{}) (interface{}, error) {
	if value == nil {
		return "", nil
	}

	str, ok := value.(string)
	if !ok {
		// Convert to string first
		str = fmt.Sprintf("%v", value)
	}

	return strings.TrimSpace(str), nil
}
