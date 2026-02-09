package models

import "time"

// Normalised Purview log
type PurviewEvent struct {
	Timestamp  time.Time              `json:"timestamp"`
	Operation  string                 `json:"operation"`
	UserID     string                 `json:"user_id"`
	ClientIP   string                 `json:"client_ip"`
	SourceFile string                 `json:"source_file"`
	RawData    map[string]interface{} `json:"raw_data"`   // Everything from the CSV row
	AuditData  map[string]interface{} `json:"audit_data"` // Parsed from JSON in AuditData column
	Flattened  map[string]interface{} `json:"flattened"`  // Combined map for Sigma matching
}
