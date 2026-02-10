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
	columns = append(columns, "Date")
	columns = append(columns, "Time")
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
				case "recordid":
					event.RecordID = value
				case "creationdate":
					// Remove leading/trailing spaces
					cleanValue := strings.TrimSpace(value)

					// Try RFC3339 (This handles the .0000000Z format perfectly)
					timeValue, err := time.Parse(time.RFC3339, cleanValue)

					if err != nil {
						// Fallback: Only normalise if RFC3339 failed
						normalized := strings.Replace(cleanValue, " ", "T", 1)
						timeValue, err = time.Parse("2006-01-02T15:04:05", normalized)
					}

					if err == nil {
						event.Timestamp = timeValue.UTC().Format(time.RFC3339)
						event.Date = timeValue.UTC().Format("2006-01-02")
						event.Time = timeValue.UTC().Format("15:04:05")
					} else {
						fmt.Fprintf(os.Stderr, "failed to parse time '%s': %v\n", cleanValue, err)
					}
				case "operation":
					event.Operation = value
				case "operationproperties":
					event.OperationProperties = value
				case "userid":
					event.UserID = value
				case "organizationname":
					event.Organisation = value
				case "workload":
					event.M365Service = value
				case "clientip":
					event.ClientIP = value
				case "clientappname":
					event.ClientAppName = value
				case "useragent":
					event.UserAgent = value
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

						// Promote Organisation
						if event.Organisation == "" && keyLower == "organizationname" {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.Organisation = stringValue
							}
						}

						// Promote OperationProperties
						if event.OperationProperties == "" && keyLower == "operationproperties" {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.OperationProperties = stringValue
							}
						}

						// Promote ClientAppName
						if event.ClientAppName == "" && keyLower == "clientappname" {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.ClientAppName = stringValue
							}
						}

						// Promote M365Service
						if event.M365Service == "" && keyLower == "workload" {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.M365Service = stringValue
							}
						}

						// Promote UserAgent
						if event.UserAgent == "" && keyLower == "useragent" {
							if stringValue, typeMatch := value.(string); typeMatch {
								event.UserAgent = stringValue
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
