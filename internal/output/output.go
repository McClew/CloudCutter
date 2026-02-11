package output

import (
	// Standard library dependencies
	"encoding/csv"
	"fmt"
	"os"
	"reflect"
	"strings"

	// Internal dependencies
	"CloudCutter/internal/parser"
	"CloudCutter/models"
)

// ExportToCSV writes a slice of PurviewEvents to a CSV file
func ExportToCSV(events []models.PurviewEvent, filePath string, includeSigma bool) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Get headers
	headers := parser.GetPurviewEventColumns(events, includeSigma)
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write CSV headers: %v", err)
	}

	// Write data
	for _, event := range events {
		var row []string
		for _, header := range headers {
			row = append(row, resolveCSVValue(header, event))
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %v", err)
		}
	}

	return nil
}

// resolveCSVValue extracts a value from a PurviewEvent based on the header name
func resolveCSVValue(header string, event models.PurviewEvent) string {
	// Handle complex/nested fields explicitly if needed, otherwise use map or reflection
	if strings.Contains(header, ".") {
		parts := strings.Split(header, ".")
		val := resolveRecursive(parts, event)
		return fmt.Sprintf("%v", val)
	}

	// Check the standard fields first
	v := reflect.ValueOf(event)
	// Try to find the field by name (case-insensitive)
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if strings.EqualFold(fieldType.Name, header) {
			return fmt.Sprintf("%v", v.Field(i).Interface())
		}
	}

	// Fallback to RawData or Flattened if not a struct field
	if val, ok := event.Flattened[header]; ok {
		return fmt.Sprintf("%v", val)
	}

	return ""
}

// resolveRecursive is a helper to traverse nested structures (similar to search logic)
func resolveRecursive(parts []string, data any) any {
	if len(parts) == 0 {
		return data
	}

	currentPart := parts[0]
	remainingParts := parts[1:]

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if strings.EqualFold(v.Type().Field(i).Name, currentPart) {
				return resolveRecursive(remainingParts, v.Field(i).Interface())
			}
		}
	case reflect.Map:
		for _, key := range v.MapKeys() {
			if strings.EqualFold(fmt.Sprintf("%v", key.Interface()), currentPart) {
				return resolveRecursive(remainingParts, v.MapIndex(key).Interface())
			}
		}
	case reflect.Slice:
		// For exporting, we might just want a string representation of the whole slice if it's the tip
		if len(remainingParts) == 0 {
			return data
		}
		// Otherwise, this is tricky for a single CSV cell.
		// For now, let's just return the first element or a comma-separated list if it's the end.
		return fmt.Sprintf("%v", data)
	}

	return ""
}
