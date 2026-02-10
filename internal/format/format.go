package format

import (
	// Standard library dependencies
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	// Internal dependencies
	"CloudCutter/models"
)

// FormatEvent formats the event based on the given format
func FormatEvent(event models.PurviewEvent, format string) string {
	switch format {
	case "log":
		return logFormat(event)
	default:
		return logFormat(event)
	}
}

// Log format
func logFormat(event models.PurviewEvent) string {
	var builder strings.Builder
	value := reflect.ValueOf(event)
	valueType := value.Type()

	var ignoreFields = []string{
		"SourceFile",
		"Timestamp",
		"RawData",
		"AuditData",
		"Flattened",
	}

	for i := 0; i < value.NumField(); i++ {
		field := valueType.Field(i)
		fieldValue := value.Field(i)
		stringValue := fmt.Sprintf("%v", fieldValue.Interface())

		if shouldIgnore(field.Name, ignoreFields) || stringValue == "" || stringValue == "{}" || stringValue == "[]" {
			continue
		}

		// Try to pretty print JSON fields
		if field.Name == "Folders" || field.Name == "AffectedItems" || field.Name == "OperationProperties" {
			var js interface{}
			if err := json.Unmarshal([]byte(stringValue), &js); err == nil {
				if pretty, err := json.MarshalIndent(js, "", "  "); err == nil {
					stringValue = string(pretty)
				}
			}
		}

		fmt.Fprintf(&builder, "%-20s: %v\n", field.Name, stringValue)
	}

	builder.WriteString("-----------------------")

	return builder.String()
}

// Helper to check slice containment
func shouldIgnore(fieldName string, ignoreList []string) bool {
	for _, ignore := range ignoreList {
		if fieldName == ignore {
			return true
		}
	}

	return false
}
