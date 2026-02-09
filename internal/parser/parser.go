package parser

import (
	// Standard library dependencies
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	// Internal dependencies
	"CloudCutter/models"
)

// Get Purview event columns
func GetPurviewEventColumns(events []models.PurviewEvent) []string {
	var columns []string
	columns = append(columns, "Timestamp")
	columns = append(columns, "Operation")
	columns = append(columns, "UserID")
	columns = append(columns, "ClientIP")

	return columns
}

// Reads Purview CSV and returns a slice of PurviewEvent structs
func ParsePurviewCSV(filePath string) []models.PurviewEvent {
	// Open the CSV file
	file, err := os.Open(filePath)

	// Error handling for file opening
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open CSV file: %v\n", err)
		return nil
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)
	reader.FieldsPerRecord = -1 // Allow variable number of fields

	// Read the header row
	headers, err := reader.Read()

	// Error handling for reading headers
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read CSV headers: %v\n", err)
		return nil
	}

	// Map header names to their column indices for easy access
	headerMap := make(map[string]int)
	for index, header := range headers {
		headerMap[strings.ToLower(strings.TrimSpace(header))] = index
	}

	var events []models.PurviewEvent

	for {
		// Read each record from the CSV
		record, err := reader.Read()

		// Error handing for end of file
		if err == io.EOF {
			break // End of file reached
		}

		event := models.PurviewEvent{
			SourceFile: filePath,
			RawData:    make(map[string]interface{}),
			AuditData:  make(map[string]interface{}),
			Flattened:  make(map[string]interface{}),
		}

		// For each column in the header, add the value to RawData and Flattened maps
		for columnName, index := range headerMap {
			// Ensure loop doesn't go out of bounds if the record has fewer columns than the header
			if index < len(record) {
				value := record[index]
				event.RawData[columnName] = value
				event.Flattened[columnName] = value

				// Extract known top-level fields
				switch columnName {
				case "creationtime":
					// Try standard ISO8601 first
					if timeValue, err := time.Parse(time.RFC3339, value); err == nil {
						event.Timestamp = timeValue.UTC()
					} else {
						// Fallback formats often seen in MS logs
						if timeValue, err := time.Parse("2006-01-02T15:04:05.999Z", value); err == nil {
							event.Timestamp = timeValue.UTC()
						}
					}
				case "operation":
					event.Operation = value
				case "userid":
					event.UserID = value
				case "clientip":
					event.ClientIP = value
				}
			}
		}

		// Parse AuditData JSON column if present
		if index, ok := headerMap["auditdata"]; ok && index < len(record) {
			// Get the raw JSON string
			auditDataStr := record[index]

			// If the string is not empty attempt to parse it
			if auditDataStr != "" && auditDataStr != "{}" {
				// Parse the JSON into a map
				var auditMap map[string]interface{}

				// Error handling for JSON parsing
				if err := json.Unmarshal([]byte(auditDataStr), &auditMap); err == nil {
					// Store the parsed audit data in the event struct
					event.AuditData = auditMap

					// Flatten nested JSON fields into the main map
					for key, value := range auditMap {
						keyLower := strings.ToLower(key)
						event.Flattened[keyLower] = value

						// Promote ClientIP
						if event.ClientIP == "" && (keyLower == "clientip" || keyLower == "clientipaddress") {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.ClientIP = stringValue
							}
						}

						// Promote UserID
						if event.UserID == "" && (keyLower == "userid" || keyLower == "userkey") {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.UserID = stringValue
							}
						}
					}
				}
			}
		}

		// Append the event to the slice
		events = append(events, event)
	}

	return events
}
