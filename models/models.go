package models

// Sigma rule structure
type SigmaRule struct {
	Name        string
	Description string
	Severity    string
	Tags        []string
	Conditions  map[string]any
}

// Normalised Purview log
type PurviewEvent struct {
	RecordID            string         `json:"record_id"`
	Date                string         `json:"date"`
	Time                string         `json:"time"`
	Timestamp           string         `json:"timestamp"`
	UserID              string         `json:"user_id"`
	Organisation        string         `json:"organisation"`
	EventSource         string         `json:"event_source"`
	M365Service         string         `json:"m365_service"`
	Operation           string         `json:"operation"`
	OperationProperties string         `json:"operation_properties"`
	ClientIP            string         `json:"client_ip"`
	ClientAppName       string         `json:"client_app_name"`
	Client              string         `json:"client"`
	UserAgent           string         `json:"user_agent"`
	ActorInfo           string         `json:"actor_info"`
	AffectedItems       string         `json:"affected_items"`
	Folders             string         `json:"folders"`
	Folder              string         `json:"folder"`
	DestinationFolder   string         `json:"destination_folder"`
	SourceFile          string         `json:"source_file"`
	RawData             map[string]any `json:"raw_data"`   // Everything from the CSV row
	AuditData           map[string]any `json:"audit_data"` // Parsed from JSON in AuditData column
	Flattened           map[string]any `json:"flattened"`  // Combined map for Sigma matching
}
